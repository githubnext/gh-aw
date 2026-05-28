//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestSkipIfNoMatchPreActivationJob tests that skip-if-no-match check is created correctly in pre-activation job
func TestSkipIfNoMatchPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-if-no-match-test")
	compiler := NewCompiler()
	cases := []struct {
		name string
		run  func(*testing.T, string, *Compiler)
	}{
		{"pre_activation_job_created_with_skip_if_no_match", testSkipIfNoMatchPreActivation},
		{"pre_activation_job_with_multiple_checks", testSkipIfNoMatchMultipleChecks},
		{"skip_if_no_match_without_roles", testSkipIfNoMatchWithoutRoles},
		{"skip_if_no_match_object_format_with_min", testSkipIfNoMatchObjectWithMin},
		{"skip_if_no_match_object_format_without_min", testSkipIfNoMatchObjectWithoutMin},
		{"skip_if_match_and_skip_if_no_match_together", testSkipIfNoMatchCombinedChecks},
		{"skip_if_no_match_with_scope_none", testSkipIfNoMatchScopeNone},
		{"skip_if_no_match_with_github_token", testSkipIfNoMatchWithGitHubToken},
		{"skip_if_no_match_with_github_app", testSkipIfNoMatchWithGitHubApp},
		{"unified_app_token_step_for_both_skip_checks", testSkipIfNoMatchUnifiedAppToken},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, tmpDir, compiler)
		})
	}
}

func testSkipIfNoMatchPreActivation(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-if-no-match-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-no-match: "is:pr is:open label:ready-to-deploy"
engine: claude
---

# Skip If No Match Workflow

This workflow has a skip-if-no-match configuration.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created")
	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:pr is:open label:ready-to-deploy"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_MIN_MATCHES: "1"`, "Expected GH_AW_SKIP_MIN_MATCHES environment variable with default value 1")
	assertLockContentContains(t, lockContent, "id: check_skip_if_no_match", "Expected check_skip_if_no_match step ID")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_no_match.outputs.skip_no_match_check_ok", "Expected activated output to include skip_no_match_check_ok condition")
	assertLockContentContains(t, lockContent, "# skip-if-no-match:", "Expected skip-if-no-match to be commented out in lock file")
	assertLockContentContains(t, lockContent, "Skip-if-no-match processed as search check in pre-activation job", "Expected comment explaining skip-if-no-match processing")
}

func testSkipIfNoMatchMultipleChecks(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "multiple-checks-workflow.md", `---
on:
  workflow_dispatch: null
  stop-after: "+48h"
  skip-if-no-match: "is:pr is:open label:urgent"
  roles: [admin, maintainer]
engine: claude
---

# Multiple Checks Workflow

This workflow has both stop-after and skip-if-no-match.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created")
	assertLockContentContains(t, lockContent, "Check stop-time limit", "Expected stop-time check to be present")
	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, "steps.check_membership.outputs.is_team_member == 'true'", "Expected activated output to include all three conditions")
	assertLockContentContains(t, lockContent, "steps.check_stop_time.outputs.stop_time_ok == 'true'", "Expected activated output to include all three conditions")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_no_match.outputs.skip_no_match_check_ok == 'true'", "Expected activated output to include all three conditions")
}

func testSkipIfNoMatchWithoutRoles(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-no-match-no-roles-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-no-match: "is:pr label:deployment"
engine: claude
---

# Skip If No Match Without Roles

This workflow has skip-if-no-match but no role restrictions.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created even without role checks")
	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_no_match.outputs.skip_no_match_check_ok", "Expected activated output to include skip_no_match_check_ok condition")
}

func testSkipIfNoMatchObjectWithMin(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-no-match-object-format-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-no-match:
    query: "is:issue is:open label:urgent"
    min: 3
engine: claude
---

# Skip If No Match Object Format

This workflow uses object format with min parameter.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:issue is:open label:urgent"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_MIN_MATCHES: "3"`, "Expected GH_AW_SKIP_MIN_MATCHES environment variable with value 3")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_no_match.outputs.skip_no_match_check_ok", "Expected activated output to include skip_no_match_check_ok condition")
}

func testSkipIfNoMatchObjectWithoutMin(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-no-match-object-no-min-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-no-match:
    query: "is:pr is:open"
engine: claude
---

# Skip If No Match Object Format Without Min

This workflow uses object format but omits min (defaults to 1).
`)

	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:pr is:open"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_MIN_MATCHES: "1"`, "Expected GH_AW_SKIP_MIN_MATCHES environment variable with default value 1")
}

func testSkipIfNoMatchCombinedChecks(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "combined-skip-checks-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-match: "is:issue is:open label:blocked"
  skip-if-no-match: "is:pr is:open label:ready"
engine: claude
---

# Combined Skip Checks Workflow

This workflow uses both skip-if-match and skip-if-no-match.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_match.outputs.skip_check_ok", "Expected activated output to include skip_check_ok condition")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_no_match.outputs.skip_no_match_check_ok", "Expected activated output to include skip_no_match_check_ok condition")
}

func testSkipIfNoMatchScopeNone(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-no-match-scope-none-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-no-match:
    query: "org:myorg label:agent-fix is:issue is:open"
    scope: none
engine: claude
---

# Skip If No Match With Scope None

This workflow uses scope:none for org-wide search.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "org:myorg label:agent-fix is:issue is:open"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_SCOPE: "none"`, "Expected GH_AW_SKIP_SCOPE environment variable set to none")
	assertLockContentContains(t, lockContent, "# scope:", "Expected scope to be commented out in lock file")
}

func testSkipIfNoMatchWithGitHubToken(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-no-match-github-token-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-no-match:
    query: "org:myorg label:agent-fix is:issue is:open"
    scope: none
  github-token: ${{ secrets.CROSS_ORG_TOKEN }}
engine: claude
---

# Skip If No Match With Custom Token

This workflow uses a custom token for org-wide search.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	assertLockContentContains(t, lockContent, "github-token: ${{ secrets.CROSS_ORG_TOKEN }}", "Expected github-token to be set in with section for skip-if-no-match step")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_SCOPE: "none"`, "Expected GH_AW_SKIP_SCOPE environment variable set to none")
}

func testSkipIfNoMatchWithGitHubApp(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "skip-no-match-github-app-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-no-match:
    query: "org:myorg label:agent-fix is:issue is:open"
    scope: none
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
    owner: myorg
engine: claude
---

# Skip If No Match With GitHub App

This workflow uses a GitHub App token for org-wide search.
`)

	assertLockContentContains(t, lockContent, "Generate GitHub App token for skip-if checks", "Expected unified GitHub App token mint step to be present")
	assertLockContentContains(t, lockContent, "client-id: ${{ secrets.WORKFLOW_APP_ID }}", "Expected client-id in the GitHub App token mint step")
	assertLockContentContains(t, lockContent, "private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}", "Expected private-key in the GitHub App token mint step")
	assertLockContentContains(t, lockContent, "owner: myorg", "Expected owner to be set in GitHub App token mint step")
	assertLockContentContains(t, lockContent, "github-token: ${{ steps.pre-activation-app-token.outputs.token }}", "Expected minted app token (pre-activation-app-token) to be used in skip-if-no-match step")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_SCOPE: "none"`, "Expected GH_AW_SKIP_SCOPE environment variable set to none")
}

func testSkipIfNoMatchUnifiedAppToken(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfNoMatchWorkflow(t, tmpDir, compiler, "unified-app-token-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-match:
    query: "org:myorg label:blocked is:issue is:open"
    scope: none
  skip-if-no-match:
    query: "org:myorg label:agent-fix is:issue is:open"
    scope: none
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
    owner: myorg
engine: claude
---

# Unified App Token For Both Skip Checks

Both skip-if-match and skip-if-no-match share one mint step.
`)

	if mintStepCount := strings.Count(lockContent, "Generate GitHub App token for skip-if checks"); mintStepCount != 1 {
		t.Errorf("Expected exactly 1 unified mint step, got %d", mintStepCount)
	}
	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, "Check skip-if-no-match query", "Expected skip-if-no-match check to be present")
	if strings.Count(lockContent, "github-token: ${{ steps.pre-activation-app-token.outputs.token }}") != 2 {
		t.Error("Expected both skip-if steps to reference the unified pre-activation-app-token step")
	}
}

func testSkipIfNoMatchWorkflow(t *testing.T, tmpDir string, compiler *Compiler, fileName, workflowContent string) string {
	t.Helper()
	workflowFile := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	return string(lockContent)
}

func assertLockContentContains(t *testing.T, lockContent, expected, message string) {
	t.Helper()
	if !strings.Contains(lockContent, expected) {
		t.Error(message)
	}
}
