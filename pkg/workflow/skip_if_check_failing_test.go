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

// TestSkipIfCheckFailingPreActivationJob tests that skip-if-check-failing check is created correctly in pre-activation job
func TestSkipIfCheckFailingPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-if-check-failing-test")
	compiler := NewCompiler()
	cases := []struct {
		name string
		run  func(*testing.T, string, *Compiler)
	}{
		{"pre_activation_job_created_with_skip_if_check_failing_boolean", testSkipIfCheckFailingBoolean},
		{"pre_activation_job_created_with_skip_if_check_failing_object_with_include_and_exclude", testSkipIfCheckFailingObjectWithFilters},
		{"skip_if_check_failing_no_env_vars_when_bare_true", testSkipIfCheckFailingBareTrue},
		{"skip_if_check_failing_combined_with_other_gates", testSkipIfCheckFailingCombinedGates},
		{"skip_if_check_failing_object_without_branch", testSkipIfCheckFailingWithoutBranch},
		{"skip_if_check_failing_null_value_treated_as_true", testSkipIfCheckFailingNullValue},
		{"skip_if_check_failing_allow_pending_sets_env_var", testSkipIfCheckFailingAllowPending},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, tmpDir, compiler)
		})
	}
}

func testSkipIfCheckFailingBoolean(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "skip-if-check-failing-workflow.md", `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing: true
engine: claude
---

# Skip If Check Failing Workflow

This workflow has a skip-if-check-failing configuration.
`)

	assertLockContentContains(t, lockContent, "pre_activation:", "Expected pre_activation job to be created")
	assertLockContentContains(t, lockContent, "Check skip-if-check-failing", "Expected skip-if-check-failing check step to be present")
	assertLockContentContains(t, lockContent, "id: check_skip_if_check_failing", "Expected check_skip_if_check_failing step ID")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_check_failing.outputs.skip_if_check_failing_ok", "Expected activated output to include skip_if_check_failing_ok condition")
	assertLockContentContains(t, lockContent, "# skip-if-check-failing:", "Expected skip-if-check-failing to be commented out in lock file")
	assertLockContentContains(t, lockContent, "Skip-if-check-failing processed as check status gate in pre-activation job", "Expected comment explaining skip-if-check-failing processing")
}

func testSkipIfCheckFailingObjectWithFilters(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "skip-if-check-failing-object-workflow.md", `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing:
    include:
      - build
      - test
    exclude:
      - lint
    branch: main
engine: claude
---

# Skip If Check Failing Object Form

This workflow uses the object form of skip-if-check-failing.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-check-failing", "Expected skip-if-check-failing check step to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_CHECK_INCLUDE: "[\"build\",\"test\"]"`, "Expected GH_AW_SKIP_CHECK_INCLUDE environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_CHECK_EXCLUDE: "[\"lint\"]"`, "Expected GH_AW_SKIP_CHECK_EXCLUDE environment variable with correct value")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_BRANCH: "main"`, "Expected GH_AW_SKIP_BRANCH environment variable with correct value")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_check_failing.outputs.skip_if_check_failing_ok", "Expected activated output to include skip_if_check_failing_ok condition")
}

func testSkipIfCheckFailingBareTrue(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "skip-if-check-failing-bare-workflow.md", `---
on:
  schedule:
    - cron: "*/30 * * * *"
  skip-if-check-failing: true
engine: claude
---

# Bare Skip If Check Failed

Skips if any checks fail on the default branch.
`)

	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_CHECK_INCLUDE", "Expected no GH_AW_SKIP_CHECK_INCLUDE when using bare true form")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_CHECK_EXCLUDE", "Expected no GH_AW_SKIP_CHECK_EXCLUDE when using bare true form")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_BRANCH", "Expected no GH_AW_SKIP_BRANCH when using bare true form")
}

func testSkipIfCheckFailingCombinedGates(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "combined-gates-workflow.md", `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-match: "is:pr is:open label:blocked"
  skip-if-check-failing:
    include:
      - build
  roles: [admin, maintainer]
engine: claude
---

# Combined Gates

This workflow combines multiple gate types.
`)

	assertLockContentContains(t, lockContent, "steps.check_membership.outputs.is_team_member == 'true'", "Expected membership check condition in activated output")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_match.outputs.skip_check_ok == 'true'", "Expected skip_check_ok condition in activated output")
	assertLockContentContains(t, lockContent, "steps.check_skip_if_check_failing.outputs.skip_if_check_failing_ok == 'true'", "Expected skip_if_check_failing_ok condition in activated output")
}

func testSkipIfCheckFailingWithoutBranch(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "skip-if-check-failing-no-branch.md", `---
on:
  pull_request:
    types: [opened]
  skip-if-check-failing:
    exclude:
      - spelling
engine: claude
---

# Skip with exclude only

Skips if non-spelling checks fail.
`)

	assertLockContentContains(t, lockContent, `GH_AW_SKIP_CHECK_EXCLUDE: "[\"spelling\"]"`, "Expected GH_AW_SKIP_CHECK_EXCLUDE environment variable")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_BRANCH", "Expected no GH_AW_SKIP_BRANCH when branch not specified")
}

func testSkipIfCheckFailingNullValue(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "skip-if-check-failing-null-workflow.md", `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing:
engine: claude
---

# Skip If Check Failing Null Value

This workflow uses the bare null form of skip-if-check-failing.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-check-failing", "Expected skip-if-check-failing check step to be present")
	assertLockContentContains(t, lockContent, "id: check_skip_if_check_failing", "Expected check_skip_if_check_failing step ID")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_CHECK_INCLUDE", "Expected no GH_AW_SKIP_CHECK_INCLUDE for bare null form")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_CHECK_EXCLUDE", "Expected no GH_AW_SKIP_CHECK_EXCLUDE for bare null form")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_BRANCH", "Expected no GH_AW_SKIP_BRANCH for bare null form")
}

func testSkipIfCheckFailingAllowPending(t *testing.T, tmpDir string, compiler *Compiler) {
	lockContent := testSkipIfCheckFailingWorkflow(t, tmpDir, compiler, "skip-if-check-failing-allow-pending.md", `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing:
    allow-pending: true
engine: claude
---

# Skip If Check Failing Allow Pending

This workflow allows pending checks.
`)

	assertLockContentContains(t, lockContent, "Check skip-if-check-failing", "Expected skip-if-check-failing check step to be present")
	assertLockContentContains(t, lockContent, `GH_AW_SKIP_CHECK_ALLOW_PENDING: "true"`, "Expected GH_AW_SKIP_CHECK_ALLOW_PENDING env var when allow-pending: true")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_CHECK_INCLUDE", "Expected no GH_AW_SKIP_CHECK_INCLUDE")
	assertLockContentNotContains(t, lockContent, "GH_AW_SKIP_CHECK_EXCLUDE", "Expected no GH_AW_SKIP_CHECK_EXCLUDE")
}

func testSkipIfCheckFailingWorkflow(t *testing.T, tmpDir string, compiler *Compiler, fileName, workflowContent string) string {
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
