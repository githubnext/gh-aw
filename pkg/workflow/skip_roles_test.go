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

// TestSkipRolesPreActivationJob tests that skip-roles check is created correctly in pre-activation job
func TestSkipRolesPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-roles-test")

	compiler := NewCompiler()

	t.Run("pre_activation_job_with_skip_roles", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
  skip-roles: [admin, maintainer, write]
roles: all
engine: claude
---

# Skip Roles Test Workflow

This workflow should skip for admin, maintainer, and write roles.
`
		workflowFile := filepath.Join(tmpDir, "skip-roles-workflow.md")
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
			t.Error("Expected pre_activation job to be created with skip-roles")
		}

		// Verify skip-roles check is in pre_activation job
		if !strings.Contains(lockContentStr, "Check if user role should be skipped") {
			t.Error("Expected skip-roles check to be present in pre_activation job")
		}

		// Verify the check_skip_roles.cjs script is being used
		if !strings.Contains(lockContentStr, "check_skip_roles.cjs") {
			t.Error("Expected check_skip_roles.cjs script to be used")
		}

		// Verify GH_AW_SKIP_ROLES environment variable is set
		if !strings.Contains(lockContentStr, "GH_AW_SKIP_ROLES: admin,maintainer,write") {
			t.Error("Expected GH_AW_SKIP_ROLES environment variable to be set with correct roles")
		}

		// Verify activated output includes skip-roles check
		if !strings.Contains(lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok") {
			t.Error("Expected activated output to include skip_roles_ok check")
		}

		// Verify the activated expression includes skip-roles condition
		expectedActivated := "steps.check_skip_roles.outputs.skip_roles_ok == 'true'"
		if !strings.Contains(lockContentStr, expectedActivated) {
			t.Error("Expected activated output to include skip-roles condition")
		}
	})

	t.Run("skip_roles_with_stop_time", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
  stop-after: "+24h"
  skip-roles: [admin]
roles: all
engine: claude
---

# Skip Roles with Stop Time

This workflow combines skip-roles and stop-time.
`
		workflowFile := filepath.Join(tmpDir, "skip-roles-stop-time.md")
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

		// Verify both checks are present
		if !strings.Contains(lockContentStr, "Check if user role should be skipped") {
			t.Error("Expected skip-roles check to be present")
		}

		if !strings.Contains(lockContentStr, "Check stop-time limit") {
			t.Error("Expected stop-time check to be present")
		}

		// Verify activated output combines both conditions
		if !strings.Contains(lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok") {
			t.Error("Expected activated output to include skip_roles_ok")
		}

		if !strings.Contains(lockContentStr, "steps.check_stop_time.outputs.stop_time_ok") {
			t.Error("Expected activated output to include stop_time_ok")
		}
	})

	t.Run("no_skip_roles_configured", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
roles: all
engine: claude
---

# No Skip Roles

This workflow has no skip-roles configured.
`
		workflowFile := filepath.Join(tmpDir, "no-skip-roles.md")
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

		// Verify skip-roles check is NOT present when not configured
		if strings.Contains(lockContentStr, "Check if user role should be skipped") {
			t.Error("Skip-roles check should not be present when not configured")
		}

		if strings.Contains(lockContentStr, "check_skip_roles.cjs") {
			t.Error("check_skip_roles.cjs should not be present when skip-roles is not configured")
		}

		// Since roles: all is set and no other checks are needed, no pre_activation job should be created
		// Actually, since it's issue_comment and roles: all (no permission checks), there should be no pre_activation
		// Only command workflows or workflows with stop-time/skip-if-match/rate-limit would have pre_activation
	})

	t.Run("skip_roles_with_permission_check", func(t *testing.T) {
		workflowContent := `---
on:
  issue_comment:
    types: [created]
  skip-roles: [admin, maintainer]
roles: [write, triage]
engine: claude
---

# Skip Roles with Permission Check

This workflow requires write or triage permission, but skips admin and maintainer.
`
		workflowFile := filepath.Join(tmpDir, "skip-roles-permission.md")
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

		// Verify both membership check and skip-roles check are present
		if !strings.Contains(lockContentStr, "Check team membership") {
			t.Error("Expected membership check to be present")
		}

		if !strings.Contains(lockContentStr, "Check if user role should be skipped") {
			t.Error("Expected skip-roles check to be present")
		}

		// Verify activated output combines both conditions
		if !strings.Contains(lockContentStr, "steps.check_membership.outputs.is_team_member") {
			t.Error("Expected activated output to include is_team_member check")
		}

		if !strings.Contains(lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok") {
			t.Error("Expected activated output to include skip_roles_ok check")
		}
	})
}
