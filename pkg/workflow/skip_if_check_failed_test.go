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

// TestSkipIfCheckFailedPreActivationJob tests that skip-if-check-failed check is created correctly in pre-activation job
func TestSkipIfCheckFailedPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-if-check-failed-test")

	compiler := NewCompiler()

	t.Run("pre_activation_job_created_with_skip_if_check_failed_boolean", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failed: true
engine: claude
---

# Skip If Check Failed Workflow

This workflow has a skip-if-check-failed configuration.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failed-workflow.md")
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

		// Verify skip-if-check-failed check step is present
		if !strings.Contains(lockContentStr, "Check skip-if-check-failed") {
			t.Error("Expected skip-if-check-failed check step to be present")
		}

		// Verify the step ID is set
		if !strings.Contains(lockContentStr, "id: check_skip_if_check_failed") {
			t.Error("Expected check_skip_if_check_failed step ID")
		}

		// Verify the activated output includes the check condition
		if !strings.Contains(lockContentStr, "steps.check_skip_if_check_failed.outputs.skip_if_check_failed_ok") {
			t.Error("Expected activated output to include skip_if_check_failed_ok condition")
		}

		// Verify skip-if-check-failed is commented out in the frontmatter
		if !strings.Contains(lockContentStr, "# skip-if-check-failed:") {
			t.Error("Expected skip-if-check-failed to be commented out in lock file")
		}

		if !strings.Contains(lockContentStr, "Skip-if-check-failed processed as check status gate in pre-activation job") {
			t.Error("Expected comment explaining skip-if-check-failed processing")
		}
	})

	t.Run("pre_activation_job_created_with_skip_if_check_failed_object_with_include_and_exclude", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failed:
    include:
      - build
      - test
    exclude:
      - lint
    branch: main
engine: claude
---

# Skip If Check Failed Object Form

This workflow uses the object form of skip-if-check-failed.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failed-object-workflow.md")
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

		// Verify skip-if-check-failed check step is present
		if !strings.Contains(lockContentStr, "Check skip-if-check-failed") {
			t.Error("Expected skip-if-check-failed check step to be present")
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
		if !strings.Contains(lockContentStr, "steps.check_skip_if_check_failed.outputs.skip_if_check_failed_ok") {
			t.Error("Expected activated output to include skip_if_check_failed_ok condition")
		}
	})

	t.Run("skip_if_check_failed_no_env_vars_when_bare_true", func(t *testing.T) {
		workflowContent := `---
on:
  schedule:
    - cron: "*/30 * * * *"
  skip-if-check-failed: true
engine: claude
---

# Bare Skip If Check Failed

Skips if any checks fail on the default branch.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failed-bare-workflow.md")
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

	t.Run("skip_if_check_failed_combined_with_other_gates", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-match: "is:pr is:open label:blocked"
  skip-if-check-failed:
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
		if !strings.Contains(lockContentStr, "steps.check_skip_if_check_failed.outputs.skip_if_check_failed_ok == 'true'") {
			t.Error("Expected skip_if_check_failed_ok condition in activated output")
		}
	})

	t.Run("skip_if_check_failed_object_without_branch", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened]
  skip-if-check-failed:
    exclude:
      - spelling
engine: claude
---

# Skip with exclude only

Skips if non-spelling checks fail.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failed-no-branch.md")
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

	t.Run("skip_if_check_failed_null_value_treated_as_true", func(t *testing.T) {
		// skip-if-check-failed: (no value / YAML null) should behave identically to skip-if-check-failed: true
		workflowContent := `---
on:
  pull_request:
    types: [opened, synchronize]
  skip-if-check-failed:
engine: claude
---

# Skip If Check Failed Null Value

This workflow uses the bare null form of skip-if-check-failed.
`
		workflowFile := filepath.Join(tmpDir, "skip-if-check-failed-null-workflow.md")
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

		// Should produce the check step, just like skip-if-check-failed: true
		if !strings.Contains(lockContentStr, "Check skip-if-check-failed") {
			t.Error("Expected skip-if-check-failed check step to be present")
		}
		if !strings.Contains(lockContentStr, "id: check_skip_if_check_failed") {
			t.Error("Expected check_skip_if_check_failed step ID")
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
}
