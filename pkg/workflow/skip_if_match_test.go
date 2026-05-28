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

// TestSkipIfMatchPreActivationJob tests that skip-if-match check is created correctly in pre-activation job
func TestSkipIfMatchPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-if-match-test")
	compiler := NewCompiler()
	cases := []struct {
		name string
		run  func(*testing.T, string, *Compiler)
	}{
		{"pre_activation_job_created_with_skip_if_match", testSkipIfMatchPreActivation},
		{"pre_activation_job_with_multiple_checks", testSkipIfMatchMultipleChecks},
		{"skip_if_match_without_roles", testSkipIfMatchWithoutRoles},
		{"skip_if_match_object_format_with_max", testSkipIfMatchObjectWithMax},
		{"skip_if_match_object_format_without_max", testSkipIfMatchObjectWithoutMax},
		{"skip_if_match_with_scope_none", testSkipIfMatchScopeNone},
		{"skip_if_match_with_github_token", testSkipIfMatchWithGitHubToken},
		{"skip_if_match_with_github_app", testSkipIfMatchWithGitHubApp},
		{"skip_if_match_with_github_app_ignore_if_missing", testSkipIfMatchWithGitHubAppIgnoreIfMissing},
		{"skip_if_match_imported_from_shared_on_section", testSkipIfMatchImportedFromSharedOnSection},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, tmpDir, compiler)
		})
	}
}

func testSkipIfMatchPreActivation(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-if-match-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-match: "is:issue is:open label:in-progress"
engine: claude
---

# Skip If Match Workflow

This workflow has a skip-if-match configuration.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created")
	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:issue is:open label:in-progress"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, "id: check_skip_if_match", "Expected check_skip_if_match step ID")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_match.outputs.skip_check_ok", "Expected activated output to include skip_check_ok condition")
	assertLockContentContains(t, lockContent, "# skip-if-match:", "Expected skip-if-match to be commented out in lock file")
	assertLockContentContains(t, lockContent, "Skip-if-match processed as search check in pre-activation job", "Expected comment explaining skip-if-match processing")
}

func testSkipIfMatchMultipleChecks(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "multiple-checks-workflow.md", `---
on:
  workflow_dispatch: null
  stop-after: "+48h"
  skip-if-match: "is:pr is:open"
  roles: [admin, maintainer]
engine: claude
---

# Multiple Checks Workflow

This workflow has both stop-after and skip-if-match.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created")
	assertLockContentContains(t, lockContent, "Check stop-time limit", "Expected stop-time check to be present")
	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, "steps.check_membership.outputs.is_team_member == 'true'", "Expected activated output to include all three conditions")
	assertLockContentContains(t, lockContent, "steps.check_stop_time.outputs.stop_time_ok == 'true'", "Expected activated output to include all three conditions")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_match.outputs.skip_check_ok == 'true'", "Expected activated output to include all three conditions")
}

func testSkipIfMatchWithoutRoles(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-no-roles-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-match: "is:issue label:bug"
engine: claude
---

# Skip If Match Without Roles

This workflow has skip-if-match but no role restrictions.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created even without role checks")
	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_match.outputs.skip_check_ok", "Expected activated output to include skip_check_ok condition")
}

func testSkipIfMatchObjectWithMax(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-object-format-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-match:
    query: "is:pr is:open"
    max: 3
engine: claude
---

# Skip If Match Object Format

This workflow uses object format with max parameter.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:pr is:open"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_MAX_MATCHES: "3"`, "Expected GH_AW_SKIP_MAX_MATCHES environment variable with value 3")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_match.outputs.skip_check_ok", "Expected activated output to include skip_check_ok condition")
}

func testSkipIfMatchObjectWithoutMax(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-object-no-max-workflow.md", `---
on:
  workflow_dispatch:
  skip-if-match:
    query: "is:issue is:open label:urgent"
engine: claude
---

# Skip If Match Object Format Without Max

This workflow uses object format but omits max (defaults to 1).
`)

	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:issue is:open label:urgent"`, "Expected GH_AW_SKIP_QUERY environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_MAX_MATCHES: "1"`, "Expected GH_AW_SKIP_MAX_MATCHES environment variable with default value 1")
}

func testSkipIfMatchScopeNone(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-match-scope-none-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-match:
    query: "org:myorg label:blocked is:issue is:open"
    scope: none
engine: claude
---

# Skip If Match With Scope None

This workflow uses scope:none for org-wide search.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_SCOPE: "none"`, "Expected GH_AW_SKIP_SCOPE environment variable set to none")
	assertLockContentContains(t, lockContent, "# scope:", "Expected scope to be commented out in lock file")
}

func testSkipIfMatchWithGitHubToken(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-match-github-token-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-match:
    query: "org:myorg label:blocked is:issue is:open"
    scope: none
  github-token: ${{ secrets.CROSS_ORG_TOKEN }}
engine: claude
---

# Skip If Match With Custom Token

This workflow uses a custom token for org-wide search.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present")
	assertLockContentContains(t, lockContent, "github-token: ${{ secrets.CROSS_ORG_TOKEN }}", "Expected github-token to be set in with section for skip-if-match step")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_SCOPE: "none"`, "Expected GH_AW_SKIP_SCOPE environment variable set to none")
}

func testSkipIfMatchWithGitHubApp(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-match-github-app-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-match:
    query: "org:myorg label:blocked is:issue is:open"
    scope: none
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
    owner: myorg
engine: claude
---

# Skip If Match With GitHub App

This workflow uses a GitHub App token for org-wide search.
`)

	assertLockContentContains(t, lockContent, "Generate GitHub App token for skip-if checks", "Expected unified GitHub App token mint step to be present")
	assertLockContentContains(t, lockContent, "client-id: ${{ secrets.WORKFLOW_APP_ID }}", "Expected client-id in the GitHub App token mint step")
	assertLockContentContains(t, lockContent, "private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}", "Expected private-key in the GitHub App token mint step")
	assertLockContentContains(t, lockContent, "owner: myorg", "Expected owner to be set in GitHub App token mint step")
	assertLockContentContains(t, lockContent, "github-token: ${{ steps.pre-activation-app-token.outputs.token }}", "Expected minted app token (pre-activation-app-token) to be used in skip-if-match step")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_SCOPE: "none"`, "Expected GH_AW_SKIP_SCOPE environment variable set to none")
}

func testSkipIfMatchWithGitHubAppIgnoreIfMissing(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-match-github-app-ignore-workflow.md", `---
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-match:
    query: "org:myorg label:blocked is:issue is:open"
    scope: none
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
    ignore-if-missing: true
engine: claude
---

# Skip If Match With GitHub App (ignore-if-missing)
`)

	assertLockContentContains(t, lockContent, "if: ${{ secrets.WORKFLOW_APP_ID != '' && secrets.WORKFLOW_APP_PRIVATE_KEY != '' }}", "Expected guard to check app secrets directly when ignore-if-missing is enabled")
	assertLockContentNotContains(t, lockContent, "GH_AW_APP_CLIENT_ID:", "Did not expect step-local GH_AW_APP_CLIENT_ID env in mint step guard")
	assertLockContentNotContains(t, lockContent, "GH_AW_APP_PRIVATE_KEY:", "Did not expect step-local GH_AW_APP_PRIVATE_KEY env in mint step guard")
	assertLockContentContains(t, lockContent, "github-token: ${{ steps.pre-activation-app-token.outputs.token || secrets.GITHUB_TOKEN }}", "Expected skip-if-match to fall back to GITHUB_TOKEN when app minting is skipped")
}

func testSkipIfMatchImportedFromSharedOnSection(t *testing.T, tmpDir string, compiler *Compiler) {
	sharedFile := filepath.Join(tmpDir, "shared-skip.md")
	sharedContent := `---
on:
  skip-if-match: 'is:issue is:open in:title "[dedup]"'
---
`
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	lockContent := testSkipIfMatchWorkflow(t, tmpDir, compiler, "skip-match-imported-workflow.md", `---
on:
  schedule:
    - cron: "0 8 * * *"
engine: claude
imports:
  - ./shared-skip.md
---

# Imported skip-if-match
`)

	assertLockContentContains(t, lockContent, "Check skip-if-match query", "Expected skip-if-match check to be present from imported on.skip-if-match")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_QUERY: "is:issue is:open in:title \"[dedup]\""`, "Expected GH_AW_SKIP_QUERY to use the imported shared skip-if-match query")
}

func testSkipIfMatchWorkflow(t *testing.T, tmpDir string, compiler *Compiler, fileName, workflowContent string) string {
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

func assertLockContentNotContains(t *testing.T, lockContent, expected, message string) {
	t.Helper()
	if strings.Contains(lockContent, expected) {
		t.Error(message)
	}
}
