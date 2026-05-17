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

// TestDetectWriteCommandsInShellScripts_GhPrCreate verifies that `gh pr create` is detected as a write command.
func TestDetectWriteCommandsInShellScripts_GhPrCreate(t *testing.T) {
	scripts := []string{`gh pr create --title "Fix bug" --body "details"`}
	cmds := detectWriteCommandsInShellScripts(scripts)
	require.Len(t, cmds, 1)
	assert.Equal(t, "gh pr create", cmds[0])
}

// TestDetectWriteCommandsInShellScripts_GhIssueClose verifies that `gh issue close` is detected.
func TestDetectWriteCommandsInShellScripts_GhIssueClose(t *testing.T) {
	scripts := []string{`gh issue close $ISSUE_NUMBER`}
	cmds := detectWriteCommandsInShellScripts(scripts)
	require.Len(t, cmds, 1)
	assert.Equal(t, "gh issue close", cmds[0])
}

// TestDetectWriteCommandsInShellScripts_ReadCommandNotDetected verifies that a read command
// (e.g. `gh pr diff`) is NOT flagged as a write command.
func TestDetectWriteCommandsInShellScripts_ReadCommandNotDetected(t *testing.T) {
	scripts := []string{`gh pr diff "$PR_NUMBER" --name-only`}
	cmds := detectWriteCommandsInShellScripts(scripts)
	assert.Empty(t, cmds, "gh pr diff is a read command and should not be detected as write")
}

// TestDetectWriteCommandsInShellScripts_Deduplicated verifies that duplicate write commands
// are reported only once.
func TestDetectWriteCommandsInShellScripts_Deduplicated(t *testing.T) {
	scripts := []string{
		`gh pr create --title "Fix 1"
gh pr create --title "Fix 2"`,
	}
	cmds := detectWriteCommandsInShellScripts(scripts)
	assert.Len(t, cmds, 1, "duplicate write commands should be deduplicated")
	assert.Equal(t, "gh pr create", cmds[0])
}

// TestDetectWriteCommandsInShellScripts_MultipleWriteCommands verifies detection of
// multiple distinct write commands.
func TestDetectWriteCommandsInShellScripts_MultipleWriteCommands(t *testing.T) {
	scripts := []string{
		`gh pr merge $PR_NUMBER --squash
gh issue comment $ISSUE_NUMBER --body "done"`,
	}
	cmds := detectWriteCommandsInShellScripts(scripts)
	assert.Len(t, cmds, 2)
	assert.Contains(t, cmds, "gh pr merge")
	assert.Contains(t, cmds, "gh issue comment")
}

// TestInferPermissionsFromShellScripts_GhCacheList verifies actions: read for gh cache list.
func TestInferPermissionsFromShellScripts_GhCacheList(t *testing.T) {
	scripts := []string{`gh cache list --json key`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionActions], "gh cache list should require actions: read")
}

// TestInferPermissionsFromShellScripts_GhRepoView verifies contents: read for gh repo view.
func TestInferPermissionsFromShellScripts_GhRepoView(t *testing.T) {
	scripts := []string{`gh repo view owner/repo --json description`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionContents], "gh repo view should require contents: read")
}

// TestInferPermissionsFromShellScripts_GhLabelList verifies issues: read for gh label list.
func TestInferPermissionsFromShellScripts_GhLabelList(t *testing.T) {
	scripts := []string{`gh label list --json name`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionIssues], "gh label list should require issues: read")
}

// TestInferPermissionsFromShellScripts_GhIssueComment verifies that `gh issue comment`
// (a write command) still causes issues: read to be inferred so the permission is present
// in the activation job — the write-command check is separate.
func TestInferPermissionsFromShellScripts_GhIssueComment(t *testing.T) {
	scripts := []string{`gh issue comment $ISSUE_NUMBER --body "hello"`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionIssues], "write commands still require at minimum read-level permission for the scope")
}

// TestInferPermissionsFromShellScripts_GhAPIReleases verifies contents: read for gh api releases.
func TestInferPermissionsFromShellScripts_GhAPIReleases(t *testing.T) {
	scripts := []string{`gh api /repos/owner/repo/releases --jq '.[0].tag_name'`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionContents], "gh api /repos/.../releases should require contents: read")
}

// TestInferPermissionsFromShellScripts_GhAPILabels verifies issues: read for gh api labels endpoint.
func TestInferPermissionsFromShellScripts_GhAPILabels(t *testing.T) {
	scripts := []string{`gh api /repos/owner/repo/labels`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionIssues], "gh api /repos/.../labels should require issues: read")
}

// TestActivationJobWriteCommandInPreStepReturnsError verifies that the compiler returns
// an error when an activation pre-step calls a write gh command.
func TestActivationJobWriteCommandInPreStepReturnsError(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-write-cmd-error")
	testFile := filepath.Join(tmpDir, "bad-workflow.md")
	testContent := `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
  pull-requests: read
engine: copilot
jobs:
  activation:
    pre-steps:
      - name: Create PR comment
        run: |
          gh pr comment "$PR_NUMBER" --body "Starting review..."
---

# Workflow whose activation pre-step illegally calls a write gh command
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.Error(t, err, "compiler should reject write gh commands in activation pre-steps")
	assert.Contains(t, err.Error(), "gh pr comment", "error should mention the offending command")
	assert.Contains(t, err.Error(), "write", "error should explain the write-permission restriction")
}

// TestActivationJobPermissionsWithGhCachePreStep verifies actions: read is added when
// an activation pre-step calls `gh cache list`.
func TestActivationJobPermissionsWithGhCachePreStep(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-gh-cache")
	testFile := filepath.Join(tmpDir, "cache-workflow.md")
	testContent := `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
  actions: read
engine: copilot
jobs:
  activation:
    pre-steps:
      - name: List caches
        run: |
          gh cache list --json key > /tmp/caches.json
---

# Workflow that lists caches in activation pre-step
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "actions: read",
		"activation job should include actions: read when pre-step calls gh cache list")
}

// TestInferPermissionsFromShellScripts_GhCodespaceList verifies that `gh codespace list`
// returns the GitHub App-only codespaces: read permission (no GITHUB_TOKEN equivalent).
func TestInferPermissionsFromShellScripts_GhCodespaceList(t *testing.T) {
	scripts := []string{`gh codespace list --json name`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionCodespaces],
		"gh codespace list should require codespaces: read (GitHub App-only)")
}

// TestInferPermissionsFromShellScripts_GhAPIOrgsMembers verifies that `gh api /orgs/.../members`
// returns the GitHub App-only members: read permission.
func TestInferPermissionsFromShellScripts_GhAPIOrgsMembers(t *testing.T) {
	scripts := []string{`gh api /orgs/myorg/members --jq '.[].login'`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionMembers],
		"gh api /orgs/.../members should require members: read (GitHub App-only)")
}

// TestInferPermissionsFromShellScripts_AppAndActionsPermissions verifies that a script
// combining standard and App-only gh commands returns both sets of permissions.
func TestInferPermissionsFromShellScripts_AppAndActionsPermissions(t *testing.T) {
	scripts := []string{
		`gh pr diff "$PR_NUMBER" --name-only
gh codespace list --json name`,
	}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionPullRequests],
		"gh pr diff should require pull-requests: read")
	assert.Equal(t, PermissionRead, perms[PermissionCodespaces],
		"gh codespace list should require codespaces: read (GitHub App-only)")
}

// TestInferPermissionsFromShellScripts_GhRepoWriteHasAppAdminPerm verifies that `gh repo archive`
// (a write command) is still inferred to need administration: read (GitHub App-only) at minimum.
func TestInferPermissionsFromShellScripts_GhRepoWriteHasAppAdminPerm(t *testing.T) {
	scripts := []string{`gh repo archive owner/repo`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionAdministration],
		"gh repo archive (write) should infer administration: read for GitHub App")
}

// TestInferPermissionsFromShellScripts_GhAPIRepoEnvironments verifies environments: read
// (GitHub App-only) for the environments REST API path.
func TestInferPermissionsFromShellScripts_GhAPIRepoEnvironments(t *testing.T) {
	scripts := []string{`gh api /repos/owner/repo/environments --jq '.[].name'`}
	perms := inferPermissionsFromShellScripts(scripts)
	assert.Equal(t, PermissionRead, perms[PermissionEnvironments],
		"gh api /repos/.../environments should require environments: read (GitHub App-only)")
}
