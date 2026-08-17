//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatedWorkflow mirrors the shape of a gh-aw compiled lock file: pre_activation carries a
// static author_association guard and every other job depends on it transitively.
const gatedWorkflow = `
name: AI Moderator
on:
  issue_comment:
    types: [created]
jobs:
  pre_activation:
    if: >
      (!(github.event_name == 'issue_comment') || !contains(fromJSON('["OWNER","MEMBER","COLLABORATOR"]'), github.event.comment.author_association))
    runs-on: ubuntu-slim
    steps:
      - run: echo pre
  activation:
    needs: pre_activation
    if: needs.pre_activation.outputs.activated == 'true'
    runs-on: ubuntu-slim
    steps:
      - run: echo activation
  agent:
    needs: activation
    runs-on: ubuntu-slim
    steps:
      - run: echo agent
  conclusion:
    needs:
      - activation
      - agent
    runs-on: ubuntu-slim
    steps:
      - run: echo conclusion
`

// ungatedWorkflow has no author_association check anywhere, so findings must be preserved.
const ungatedWorkflow = `
name: Ungated
on:
  issue_comment:
    types: [created]
jobs:
  pre_activation:
    runs-on: ubuntu-slim
    steps:
      - run: echo pre
  agent:
    needs: pre_activation
    runs-on: ubuntu-slim
    steps:
      - run: echo agent
`

const workflowRunActorAllowlistWorkflow = `
name: Dev Hawk
on:
  workflow_run:
    workflows: [Dev]
    types: [completed]
jobs:
  activation:
    if: >
      github.event.workflow_run.event == 'workflow_dispatch' &&
      contains(fromJSON('["trusted-user","trusted-bot"]'), github.event.workflow_run.actor.login)
    runs-on: ubuntu-slim
    steps:
      - run: echo activation
  agent:
    needs: activation
    runs-on: ubuntu-slim
    steps:
      - run: echo agent
`

func writeWorkflow(t *testing.T, gitRoot string, name string, content string) {
	t.Helper()
	dir := filepath.Join(gitRoot, ".github", "workflows")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

func TestTrustedActivationGatedJobs(t *testing.T) {
	t.Run("gates propagate through the needs graph", func(t *testing.T) {
		gitRoot := t.TempDir()
		writeWorkflow(t, gitRoot, "gated.lock.yml", gatedWorkflow)

		gated := trustedActivationGatedJobs(filepath.Join(gitRoot, ".github", "workflows", "gated.lock.yml"))

		assert.Contains(t, gated, "pre_activation")
		assert.Contains(t, gated, "activation")
		assert.Contains(t, gated, "agent")
		assert.Contains(t, gated, "conclusion")
	})

	t.Run("workflow without a gate has no gated jobs", func(t *testing.T) {
		gitRoot := t.TempDir()
		writeWorkflow(t, gitRoot, "ungated.lock.yml", ungatedWorkflow)

		gated := trustedActivationGatedJobs(filepath.Join(gitRoot, ".github", "workflows", "ungated.lock.yml"))

		assert.Empty(t, gated)
	})

	t.Run("workflow_run actor allowlist propagates through the needs graph", func(t *testing.T) {
		gitRoot := t.TempDir()
		writeWorkflow(t, gitRoot, "workflow-run-allowlist.lock.yml", workflowRunActorAllowlistWorkflow)

		gated := trustedActivationGatedJobs(filepath.Join(gitRoot, ".github", "workflows", "workflow-run-allowlist.lock.yml"))

		assert.Contains(t, gated, "activation")
		assert.Contains(t, gated, "agent")
	})

	t.Run("missing file yields no gated jobs", func(t *testing.T) {
		assert.Empty(t, trustedActivationGatedJobs(filepath.Join(t.TempDir(), "missing.lock.yml")))
	})

	t.Run("empty path yields no gated jobs", func(t *testing.T) {
		assert.Empty(t, trustedActivationGatedJobs(""))
	})
}

func TestFilterRunnerGuardFindings(t *testing.T) {
	gitRoot := t.TempDir()
	writeWorkflow(t, gitRoot, "gated.lock.yml", gatedWorkflow)
	writeWorkflow(t, gitRoot, "ungated.lock.yml", ungatedWorkflow)

	findings := []runnerGuardFinding{
		{RuleID: "RGS-004", File: "gated.lock.yml", JobID: "agent"},
		{RuleID: "RGS-004", File: ".github/workflows/gated.lock.yml", JobID: "conclusion"},
		{RuleID: "RGS-004", File: "ungated.lock.yml", JobID: "agent"},
		{RuleID: "RGS-005", File: "gated.lock.yml", JobID: "agent"},
		{RuleID: "RGS-004", File: "gated.lock.yml", JobID: ""},
		{RuleID: "RGS-004", File: "gated.lock.yml", JobID: "unknown_job"},
	}

	filtered := filterRunnerGuardFindings(findings, gitRoot)

	require.Len(t, filtered, 4)
	assert.Equal(t, "ungated.lock.yml", filtered[0].File)
	assert.Equal(t, "RGS-005", filtered[1].RuleID)
	assert.Empty(t, filtered[2].JobID)
	assert.Equal(t, "unknown_job", filtered[3].JobID)
}

func TestFilterRunnerGuardFindingsKeepsFindingsForUnresolvableFiles(t *testing.T) {
	gitRoot := t.TempDir()

	findings := []runnerGuardFinding{
		{RuleID: "RGS-004", File: "../outside.lock.yml", JobID: "agent"},
		{RuleID: "RGS-004", File: "does-not-exist.lock.yml", JobID: "agent"},
	}

	assert.Len(t, filterRunnerGuardFindings(findings, gitRoot), 2)
}

func TestJobNeeds(t *testing.T) {
	assert.Equal(t, []string{"a"}, jobNeeds("a"))
	assert.Equal(t, []string{"a", "b"}, jobNeeds([]any{"a", "b", 42}))
	assert.Equal(t, []string{"a"}, jobNeeds([]string{"a"}))
	assert.Nil(t, jobNeeds(nil))
}

func TestHasWorkflowRunActorAllowlistCheck(t *testing.T) {
	assert.True(t, hasWorkflowRunActorAllowlistCheck("${{ github.event.workflow_run.event == 'workflow_dispatch' && contains(fromJSON('[\"owner\"]'), github.event.workflow_run.actor.login) }}"))
	assert.True(t, hasWorkflowRunActorAllowlistCheck("${{ github.event.workflow_run.event == 'workflow_dispatch' && !contains(fromJSON('[\"blocked\"]'), github.event.workflow_run.actor.login) && contains(fromJSON('[\"owner\"]'), github.event.workflow_run.actor.login) }}"))
	assert.False(t, hasWorkflowRunActorAllowlistCheck("${{ github.event.workflow_run.event == 'workflow_dispatch' && !contains(fromJSON('[\"owner\"]'), github.event.workflow_run.actor.login) }}"))
	assert.False(t, hasWorkflowRunActorAllowlistCheck("${{ github.event.workflow_run.event == 'workflow_dispatch' && contains(fromJSON('[\"owner\"]'), github.event.workflow_run.actor.login) == false }}"))
	assert.False(t, hasWorkflowRunActorAllowlistCheck("${{ contains(fromJSON('[\"owner\"]'), github.event.comment.user.login) }}"))
}
