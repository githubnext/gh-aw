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

	t.Run("pre_activation_job_created_with_skip_if_check_failing_boolean", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing: true
engine: claude
---

# Skip If Check Failing Workflow

This workflow has a skip-if-check-failing configuration.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failing-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify pre_activation job exists
		if !strings.Contains(lockContentStr, "pre_activation:") {
			t.Error("Expected pre_activation job to be created")
		}

		// Verify skip-if-check-failing check step is present
		if !strings.Contains(lockContentStr, "Check skip-if-check-failing") {
			t.Error("Expected skip-if-check-failing check step to be present")
		}

		// Verify the step ID is set
		if !strings.Contains(lockContentStr, "id: check_skip_if_check_failing") {
			t.Error("Expected check_skip_if_check_failing step ID")
		}

		// Verify the activated output includes the check condition
		if !strings.Contains(lockContentStr, "steps.check_skip_if_check_failing.outputs.skip_if_check_failing_ok") {
			t.Error("Expected activated output to include skip_if_check_failing_ok condition")
		}

		// Verify skip-if-check-failing is commented out in the frontmatter
		if !strings.Contains(lockContentStr, "# skip-if-check-failing:") {
			t.Error("Expected skip-if-check-failing to be commented out in lock file")
		}

		if !strings.Contains(lockContentStr, "Skip-if-check-failing processed as check status gate in pre-activation job") {
			t.Error("Expected comment explaining skip-if-check-failing processing")
		}
	})

	t.Run("pre_activation_job_created_with_skip_if_check_failing_object_with_include_and_exclude", func(t *testing.T) {
		workflowContent := `---
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
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failing-object-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify skip-if-check-failing check step is present
		if !strings.Contains(lockContentStr, "Check skip-if-check-failing") {
			t.Error("Expected skip-if-check-failing check step to be present")
		}

		// Verify include list is passed as JSON env var
		if !strings.Contains(lockContentStr, `GH_AW_SKIP_CHECK_INCLUDE: "[\"build\",\"test\"]"`) {
			t.Error("Expected GH_AW_SKIP_CHECK_INCLUDE environment variable with correct value")
		}

		// Verify exclude list is passed as JSON env var
		if !strings.Contains(lockContentStr, `GH_AW_SKIP_CHECK_EXCLUDE: "[\"lint\"]"`) {
			t.Error("Expected GH_AW_SKIP_CHECK_EXCLUDE environment variable with correct value")
		}

		// Verify branch is passed
		if !strings.Contains(lockContentStr, `GH_AW_SKIP_BRANCH: "main"`) {
			t.Error("Expected GH_AW_SKIP_BRANCH environment variable with correct value")
		}

		// Verify condition is in activated output
		if !strings.Contains(lockContentStr, "steps.check_skip_if_check_failing.outputs.skip_if_check_failing_ok") {
			t.Error("Expected activated output to include skip_if_check_failing_ok condition")
		}
	})

	t.Run("skip_if_check_failing_no_env_vars_when_bare_true", func(t *testing.T) {
		workflowContent := `---
on:
  schedule:
    - cron: "*/30 * * * *"
  skip-if-check-failing: true
engine: claude
---

# Bare Skip If Check Failed

Skips if any checks fail on the default branch.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failing-bare-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// When bare true, no env vars should be set (no include/exclude/branch)
		if strings.Contains(lockContentStr, "GH_AW_SKIP_CHECK_INCLUDE") {
			t.Error("Expected no GH_AW_SKIP_CHECK_INCLUDE when using bare true form")
		}
		if strings.Contains(lockContentStr, "GH_AW_SKIP_CHECK_EXCLUDE") {
			t.Error("Expected no GH_AW_SKIP_CHECK_EXCLUDE when using bare true form")
		}
		if strings.Contains(lockContentStr, "GH_AW_SKIP_BRANCH") {
			t.Error("Expected no GH_AW_SKIP_BRANCH when using bare true form")
		}
	})

	t.Run("skip_if_check_failing_combined_with_other_gates", func(t *testing.T) {
		workflowContent := `---
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
`
		workflowFile := filepath.Join(tmpDir, "combined-gates-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// All conditions should appear in the activated output
		if !strings.Contains(lockContentStr, "steps.check_membership.outputs.is_team_member == 'true'") {
			t.Error("Expected membership check condition in activated output")
		}
		if !strings.Contains(lockContentStr, "steps.check_skip_if_match.outputs.skip_check_ok == 'true'") {
			t.Error("Expected skip_check_ok condition in activated output")
		}
		if !strings.Contains(lockContentStr, "steps.check_skip_if_check_failing.outputs.skip_if_check_failing_ok == 'true'") {
			t.Error("Expected skip_if_check_failing_ok condition in activated output")
		}
	})

	t.Run("skip_if_check_failing_object_without_branch", func(t *testing.T) {
		workflowContent := `---
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
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failing-no-branch.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		if !strings.Contains(lockContentStr, `GH_AW_SKIP_CHECK_EXCLUDE: "[\"spelling\"]"`) {
			t.Error("Expected GH_AW_SKIP_CHECK_EXCLUDE environment variable")
		}
		if strings.Contains(lockContentStr, "GH_AW_SKIP_BRANCH") {
			t.Error("Expected no GH_AW_SKIP_BRANCH when branch not specified")
		}
	})

	t.Run("skip_if_check_failing_null_value_treated_as_true", func(t *testing.T) {
		// skip-if-check-failing: (no value / YAML null) should behave identically to skip-if-check-failing: true
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing:
engine: claude
---

# Skip If Check Failing Null Value

This workflow uses the bare null form of skip-if-check-failing.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failing-null-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Should produce the check step, just like skip-if-check-failing: true
		if !strings.Contains(lockContentStr, "Check skip-if-check-failing") {
			t.Error("Expected skip-if-check-failing check step to be present")
		}
		if !strings.Contains(lockContentStr, "id: check_skip_if_check_failing") {
			t.Error("Expected check_skip_if_check_failing step ID")
		}
		// No env vars since no include/exclude/branch
		if strings.Contains(lockContentStr, "GH_AW_SKIP_CHECK_INCLUDE") {
			t.Error("Expected no GH_AW_SKIP_CHECK_INCLUDE for bare null form")
		}
		if strings.Contains(lockContentStr, "GH_AW_SKIP_CHECK_EXCLUDE") {
			t.Error("Expected no GH_AW_SKIP_CHECK_EXCLUDE for bare null form")
		}
		if strings.Contains(lockContentStr, "GH_AW_SKIP_BRANCH") {
			t.Error("Expected no GH_AW_SKIP_BRANCH for bare null form")
		}
	})

	t.Run("skip_if_check_failing_allow_pending_sets_env_var", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failing:
    allow-pending: true
engine: claude
---

# Skip If Check Failing Allow Pending

This workflow allows pending checks.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failing-allow-pending.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		if !strings.Contains(lockContentStr, "Check skip-if-check-failing") {
			t.Error("Expected skip-if-check-failing check step to be present")
		}
		if !strings.Contains(lockContentStr, `GH_AW_SKIP_CHECK_ALLOW_PENDING: "true"`) {
			t.Error("Expected GH_AW_SKIP_CHECK_ALLOW_PENDING env var when allow-pending: true")
		}
		// No include/exclude/branch since only allow-pending was set
		if strings.Contains(lockContentStr, "GH_AW_SKIP_CHECK_INCLUDE") {
			t.Error("Expected no GH_AW_SKIP_CHECK_INCLUDE")
		}
		if strings.Contains(lockContentStr, "GH_AW_SKIP_CHECK_EXCLUDE") {
			t.Error("Expected no GH_AW_SKIP_CHECK_EXCLUDE")
		}
	})
}
