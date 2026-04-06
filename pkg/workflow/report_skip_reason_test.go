//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportSkipReasonStep tests that the report-skip-reason step is generated correctly
// in the pre-activation job whenever blocking conditions are present, so that operators
// can see the denial reason from the PR / workflow-run surface without opening raw logs.
func TestReportSkipReasonStep(t *testing.T) {
	tmpDir := testutil.TempDir(t, "report-skip-reason-test")
	compiler := NewCompiler()

	t.Run("report_skip_reason_present_with_membership_check", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened]
  roles: [admin, maintainer, write]
engine: copilot
---

# PR Review Workflow

Review pull requests.
`
		workflowFile := filepath.Join(tmpDir, "membership-workflow.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify the report-skip-reason step is present
		assert.Contains(t, lockContentStr, "id: report-skip-reason", "Expected report-skip-reason step ID")
		assert.Contains(t, lockContentStr, "Report skip reason", "Expected Report skip reason step name")

		// Verify the step has always() condition
		assert.Contains(t, lockContentStr, "if: always()", "Expected always() condition on report-skip-reason step")

		// Verify membership check env vars are passed
		assert.Contains(t, lockContentStr, "GH_AW_IS_TEAM_MEMBER:", "Expected GH_AW_IS_TEAM_MEMBER env var")
		assert.Contains(t, lockContentStr, "GH_AW_MEMBERSHIP_RESULT:", "Expected GH_AW_MEMBERSHIP_RESULT env var")
		assert.Contains(t, lockContentStr, "GH_AW_MEMBERSHIP_ERROR_MESSAGE:", "Expected GH_AW_MEMBERSHIP_ERROR_MESSAGE env var")

		// Verify the script calls report_pre_activation_skip.cjs
		assert.Contains(t, lockContentStr, "report_pre_activation_skip.cjs", "Expected report_pre_activation_skip.cjs in script")
	})

	t.Run("report_skip_reason_present_with_stop_time", func(t *testing.T) {
		workflowContent := `---
on:
  workflow_dispatch: null
  stop-after: "+48h"
  roles: [admin, maintainer]
engine: copilot
---

# Stop-Time Workflow

This workflow has a stop-after configuration.
`
		workflowFile := filepath.Join(tmpDir, "stop-time-workflow.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		assert.Contains(t, lockContentStr, "id: report-skip-reason", "Expected report-skip-reason step ID")
		assert.Contains(t, lockContentStr, "GH_AW_STOP_TIME_OK:", "Expected GH_AW_STOP_TIME_OK env var")
	})

	t.Run("report_skip_reason_present_with_skip_bots", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-bots: [renovate, dependabot]
engine: copilot
---

# Skip-Bots Workflow

This workflow skips bot actors.
`
		workflowFile := filepath.Join(tmpDir, "skip-bots-workflow.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		assert.Contains(t, lockContentStr, "id: report-skip-reason", "Expected report-skip-reason step ID")
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_BOTS_OK:", "Expected GH_AW_SKIP_BOTS_OK env var")
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_BOTS_ERROR_MESSAGE:", "Expected GH_AW_SKIP_BOTS_ERROR_MESSAGE env var")
	})

	t.Run("report_skip_reason_present_with_skip_roles", func(t *testing.T) {
		workflowContent := `---
on:
  pull_request:
    types: [opened]
  skip-roles: [admin]
engine: copilot
---

# Skip-Roles Workflow

This workflow skips admin actors.
`
		workflowFile := filepath.Join(tmpDir, "skip-roles-workflow.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		assert.Contains(t, lockContentStr, "id: report-skip-reason", "Expected report-skip-reason step ID")
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES_OK:", "Expected GH_AW_SKIP_ROLES_OK env var")
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES_ERROR_MESSAGE:", "Expected GH_AW_SKIP_ROLES_ERROR_MESSAGE env var")
	})

	t.Run("report_skip_reason_not_present_without_conditions", func(t *testing.T) {
		// When activation is unconditionally true (roles: all + only on.steps), there are no
		// blocking conditions so the report-skip-reason step should NOT be generated.
		workflowContent := `---
on:
  pull_request:
    types: [opened]
  roles: all
  steps:
    - name: Custom gate
      id: custom_gate
      run: echo "gate_result=ok" >> $GITHUB_OUTPUT
engine: copilot
if: needs.pre_activation.outputs.custom_gate_result == 'ok'
---

# On-Steps-Only Workflow

Uses on.steps to gate activation with no blocking conditions.
`
		workflowFile := filepath.Join(tmpDir, "on-steps-only-workflow.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// No blocking conditions → no report-skip-reason step
		assert.NotContains(t, lockContentStr, "report-skip-reason", "report-skip-reason step should not appear for on.steps-only workflows")
	})
}
