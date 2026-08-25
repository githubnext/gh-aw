//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestCompileSteeringIssue(t *testing.T) {
	tmpDir := testutil.TempDir(t, "steering-issue")
	workflowPath := filepath.Join(tmpDir, "steering.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
engine: copilot
strict: false
permissions:
  issues: read
safe-outputs:
  create-pull-request:
    steer: true
    branch-prefix: "signed/"
---

# Steering test

Create a change and open a pull request.
`), 0o644))

	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	compiled, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
	require.NoError(t, err)
	yaml := string(compiled)

	activation := extractJobSection(yaml, "activation")
	agent := extractJobSection(yaml, "agent")
	safeOutputs := extractJobSection(yaml, "safe_outputs")
	conclusion := extractJobSection(yaml, "conclusion")

	assert.Contains(t, activation, "id: create-steering-issue")
	assert.Contains(t, activation, "issues: write")
	assert.NotContains(t, activation, "contents: write")
	assert.NotContains(t, activation, "pull-requests: write")
	assert.NotContains(t, activation, "checks: write")
	assert.NotContains(t, agent, "gh-aw/pre-created")
	assert.NotContains(t, safeOutputs, "pre_created_pull_request")
	assert.Contains(t, conclusion, "complete_steering_issue.cjs")
	assert.Contains(t, conclusion, "GH_AW_STEERING_ISSUE_NUMBER")
}

func TestCompileRejectsLegacyPreCreatePullRequest(t *testing.T) {
	tmpDir := testutil.TempDir(t, "legacy-pre-create-pull-request")
	workflowPath := filepath.Join(tmpDir, "pre-create.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
engine: copilot
strict: false
safe-outputs:
  create-pull-request:
    pre-create: true
---

# Legacy pre-create test
`), 0o644))

	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)
	err := compiler.CompileWorkflow(workflowPath)
	require.ErrorContains(t, err, "pre-create")
}
