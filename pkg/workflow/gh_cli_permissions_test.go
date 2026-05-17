//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInferPermissionsFromShellScripts_GhPrDiff verifies that `gh pr diff` in a
// shell script is recognized as requiring pull-requests: read.
func TestInferPermissionsFromShellScripts_GhPrDiff(t *testing.T) {
	scripts := []string{
		`gh pr diff "$PR_NUMBER" --name-only | awk '/\.md$/' > /tmp/changed.txt`,
	}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionPullRequests], "gh pr diff should require pull-requests: read")
}

// TestInferPermissionsFromShellScripts_GhPrView verifies pull-requests: read for gh pr view.
func TestInferPermissionsFromShellScripts_GhPrView(t *testing.T) {
	scripts := []string{`gh pr view 123 --json title`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionPullRequests])
}

// TestInferPermissionsFromShellScripts_GhIssueList verifies issues: read for gh issue list.
func TestInferPermissionsFromShellScripts_GhIssueList(t *testing.T) {
	scripts := []string{`gh issue list --label bug --json number`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionIssues])
}

// TestInferPermissionsFromShellScripts_GhWorkflowList verifies actions: read for gh workflow list.
func TestInferPermissionsFromShellScripts_GhWorkflowList(t *testing.T) {
	scripts := []string{`gh workflow list`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionActions])
}

// TestInferPermissionsFromShellScripts_GhRunView verifies actions: read for gh run view.
func TestInferPermissionsFromShellScripts_GhRunView(t *testing.T) {
	scripts := []string{`gh run view $RUN_ID --json conclusion`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionActions])
}

// TestInferPermissionsFromShellScripts_GhAPI verifies pull-requests: read for gh api pulls endpoint.
func TestInferPermissionsFromShellScripts_GhAPI(t *testing.T) {
	scripts := []string{`gh api /repos/owner/repo/pulls/1 --jq .title`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionPullRequests], "gh api /repos/.../pulls should require pull-requests: read")
}

// TestInferPermissionsFromShellScripts_GhAPIIssues verifies issues: read for gh api issues endpoint.
func TestInferPermissionsFromShellScripts_GhAPIIssues(t *testing.T) {
	scripts := []string{`gh api /repos/owner/repo/issues --jq '.[].number'`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionIssues], "gh api /repos/.../issues should require issues: read")
}

// TestInferPermissionsFromShellScripts_NoGhCommand verifies no permissions are inferred when
// there are no gh CLI calls in the script.
func TestInferPermissionsFromShellScripts_NoGhCommand(t *testing.T) {
	scripts := []string{`echo "hello" && ls /tmp`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Empty(t, perms, "no gh commands should produce no permission requirements")
}

// TestInferPermissionsFromShellScripts_MultiLine verifies multi-line shell scripts work correctly.
func TestInferPermissionsFromShellScripts_MultiLine(t *testing.T) {
	scripts := []string{
		`gh pr diff "$PR_NUMBER" --name-only \
  | awk '/\.md$/' \
  > /tmp/gh-aw/docs-review-data/changed-md.txt`,
	}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionPullRequests], "multi-line gh pr diff should require pull-requests: read")
}

// TestInferPermissionsFromShellScripts_MultipleCommands verifies multiple gh commands are aggregated.
func TestInferPermissionsFromShellScripts_MultipleCommands(t *testing.T) {
	scripts := []string{
		`gh pr diff "$PR_NUMBER" --name-only > /tmp/changed.txt
gh issue view $ISSUE_NUMBER --json body > /tmp/issue.json`,
	}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionPullRequests], "should infer pull-requests: read")
	assert.Equal(t, PermissionRead, perms[PermissionIssues], "should infer issues: read")
}

// TestExtractRunScriptsFromJobPreSteps verifies extraction of run scripts from jobs map.
func TestExtractRunScriptsFromJobPreSteps(t *testing.T) {
	jobs := map[string]any{
		"activation": map[string]any{
			"pre-steps": []any{
				map[string]any{
					"name": "Get changed markdown files",
					"run":  `gh pr diff "$PR_NUMBER" --name-only | awk '/\.md$/' > /tmp/changed.txt`,
				},
				map[string]any{
					"name": "Echo step",
					"run":  `echo "hello"`,
				},
			},
		},
	}

	scripts := extractRunScriptsFromJobPreSteps(jobs, "activation")
	require.Len(t, scripts, 2)
	assert.Contains(t, scripts[0], "gh pr diff")
	assert.Contains(t, scripts[1], "echo")
}

// TestExtractRunScriptsFromJobPreSteps_NoPreSteps verifies nil return when pre-steps absent.
func TestExtractRunScriptsFromJobPreSteps_NoPreSteps(t *testing.T) {
	jobs := map[string]any{
		"activation": map[string]any{
			"permissions": map[string]any{"contents": "read"},
		},
	}
	scripts := extractRunScriptsFromJobPreSteps(jobs, "activation")
	assert.Nil(t, scripts)
}

// TestExtractRunScriptsFromJobPreSteps_NonRunSteps verifies that non-run steps (uses: ...) are skipped.
func TestExtractRunScriptsFromJobPreSteps_NonRunSteps(t *testing.T) {
	jobs := map[string]any{
		"activation": map[string]any{
			"pre-steps": []any{
				map[string]any{
					"name": "Checkout",
					"uses": "actions/checkout@v4",
				},
			},
		},
	}
	scripts := extractRunScriptsFromJobPreSteps(jobs, "activation")
	assert.Empty(t, scripts, "uses-only steps should not produce any scripts")
}

// TestActivationJobPermissionsWithGhPrDiffPreStep is an integration test that verifies
// the compiler adds pull-requests: read to the activation job when a pre-step calls
// `gh pr diff`. This reproduces the issue reported for the gh-aw-docs-review workflow.
func TestActivationJobPermissionsWithGhPrDiffPreStep(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-gh-pr-diff")
	testFile := filepath.Join(tmpDir, "docs-review.md")
	testContent := `---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  pull-requests: read
engine: copilot
jobs:
  activation:
    pre-steps:
      - name: Get changed markdown files
        run: |
          gh pr diff "$PR_NUMBER" --name-only \
            | awk '/\.md$/' \
            > /tmp/gh-aw/docs-review-data/changed-md.txt
---

# Docs review workflow with Vale pre-step
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "pull-requests: read",
		"activation job should include pull-requests: read when pre-step calls gh pr diff")
}

// TestActivationJobPermissionsWithGhIssuePreStep verifies issues: read is added when
// an activation pre-step calls `gh issue view`.
func TestActivationJobPermissionsWithGhIssuePreStep(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-gh-issue")
	testFile := filepath.Join(tmpDir, "issue-workflow.md")
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: copilot
jobs:
  activation:
    pre-steps:
      - name: Fetch issue data
        run: |
          gh issue view "$ISSUE_NUMBER" --json body > /tmp/issue.json
---

# Issue workflow with gh issue pre-step
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "issues: read",
		"activation job should include issues: read when pre-step calls gh issue view")
}

// TestActivationJobPermissionsNoPreStepChanges verifies that the activation job permissions
// are unchanged when there are no pre-steps with gh commands.  Even if the workflow-level
// frontmatter declares pull-requests: read, the activation job should NOT receive that
// permission unless its own steps actually need it (the activation job computes its permissions
// independently of the main job's filtered permissions).
func TestActivationJobPermissionsNoPreStepChanges(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-no-gh")
	testFile := filepath.Join(tmpDir, "basic-workflow.md")
	testContent := `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
engine: copilot
---

# Basic workflow without activation pre-steps
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "contents: read",
		"activation job should always include contents: read")
	assert.NotContains(t, activationJobSection, "pull-requests",
		"activation job should NOT include pull-requests when no pre-step requires it")
}
