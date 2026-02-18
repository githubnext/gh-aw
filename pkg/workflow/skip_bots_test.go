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

// TestSkipBotsPreActivationJob tests that skip-bots check is created correctly in pre-activation job
func TestSkipBotsPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-bots-test")
	compiler := NewCompiler()

	t.Run("pre_activation_job_created_with_skip_bots", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-bots: [user1, user2, user3]
engine: copilot
---

# Skip Users Workflow

This workflow has a skip-bots configuration.
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

		// Verify pre_activation job exists
		assert.Contains(t, lockContentStr, "pre_activation:", "Expected pre_activation job to be created")

		// Verify skip-bots check is present
		assert.Contains(t, lockContentStr, "Check skip-bots", "Expected skip-bots check to be present")

		// Verify the skip users environment variable is set correctly
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_BOTS: user1,user2,user3", "Expected GH_AW_SKIP_BOTS environment variable with correct value")

		// Verify the check_skip_bots step ID is present
		assert.Contains(t, lockContentStr, "id: check_skip_bots", "Expected check_skip_bots step ID")

		// Verify the activated output includes skip_bots_ok condition
		assert.Contains(t, lockContentStr, "steps.check_skip_bots.outputs.skip_bots_ok", "Expected activated output to include skip_bots_ok condition")

		// Verify skip-bots is commented out in the frontmatter
		assert.Contains(t, lockContentStr, "# skip-bots:", "Expected skip-bots to be commented out in lock file")
	})

	t.Run("skip_bots_with_single_user", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-bots: user1
engine: copilot
---

# Skip Users Single User

This workflow skips only for user1.
`
		workflowFile := filepath.Join(tmpDir, "skip-bots-single.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify skip-bots check is present
		assert.Contains(t, lockContentStr, "Check skip-bots", "Expected skip-bots check to be present")

		// Verify single user
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_BOTS: user1", "Expected GH_AW_SKIP_BOTS with single user")
	})

	t.Run("no_skip_bots_no_check_created", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
engine: copilot
---

# No Skip Users Workflow

This workflow has no skip-bots configuration.
`
		workflowFile := filepath.Join(tmpDir, "no-skip-bots.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify skip-bots check is NOT present
		assert.NotContains(t, lockContentStr, "Check skip-bots", "Expected skip-bots check to NOT be present")
		assert.NotContains(t, lockContentStr, "GH_AW_SKIP_BOTS", "Expected GH_AW_SKIP_BOTS to NOT be present")
		assert.NotContains(t, lockContentStr, "check_skip_bots", "Expected check_skip_bots step to NOT be present")
	})

	t.Run("skip_bots_with_roles_field", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-bots: [user1, user2]
  roles: [maintainer]
engine: copilot
---

# Skip Users with Roles Field

This workflow has both roles and skip-bots.
`
		workflowFile := filepath.Join(tmpDir, "skip-bots-with-roles.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify both membership check and skip-bots check are present
		assert.Contains(t, lockContentStr, "Check team membership", "Expected team membership check to be present")
		assert.Contains(t, lockContentStr, "Check skip-bots", "Expected skip-bots check to be present")

		// Verify GH_AW_REQUIRED_ROLES is set
		assert.Contains(t, lockContentStr, "GH_AW_REQUIRED_ROLES: maintainer", "Expected GH_AW_REQUIRED_ROLES for roles field")

		// Verify GH_AW_SKIP_BOTS is set
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_BOTS: user1,user2", "Expected GH_AW_SKIP_BOTS for skip-bots field")

		// Verify both conditions in activated output
		assert.Contains(t, lockContentStr, "steps.check_membership.outputs.is_team_member", "Expected membership check in activated output")
		assert.Contains(t, lockContentStr, "steps.check_skip_bots.outputs.skip_bots_ok", "Expected skip-bots check in activated output")
	})

	t.Run("skip_bots_and_skip_roles_combined", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-roles: [admin, write]
  skip-bots: [user1, user2]
engine: copilot
---

# Skip Users and Skip Roles Combined

This workflow has both skip-roles and skip-bots.
`
		workflowFile := filepath.Join(tmpDir, "skip-bots-and-roles.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify both skip-roles and skip-bots checks are present
		assert.Contains(t, lockContentStr, "Check skip-roles", "Expected skip-roles check to be present")
		assert.Contains(t, lockContentStr, "Check skip-bots", "Expected skip-bots check to be present")

		// Verify both environment variables are set
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES: admin,write", "Expected GH_AW_SKIP_ROLES for skip-roles field")
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_BOTS: user1,user2", "Expected GH_AW_SKIP_BOTS for skip-bots field")

		// Verify both conditions in activated output
		assert.Contains(t, lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok", "Expected skip-roles check in activated output")
		assert.Contains(t, lockContentStr, "steps.check_skip_bots.outputs.skip_bots_ok", "Expected skip-bots check in activated output")
	})
}

// TestExtractSkipBots tests the extractSkipBots function
func TestExtractSkipBots(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []string
	}{
		{
			name: "skip-bots as array of strings",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-bots": []string{"user1", "user2"},
				},
			},
			expected: []string{"user1", "user2"},
		},
		{
			name: "skip-bots as single string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-bots": "user1",
				},
			},
			expected: []string{"user1"},
		},
		{
			name: "skip-bots as array of any",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-bots": []any{"user1", "user2", "user3"},
				},
			},
			expected: []string{"user1", "user2", "user3"},
		},
		{
			name: "no skip-bots configured",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
				},
			},
			expected: nil,
		},
		{
			name: "empty skip-bots array",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-bots": []string{},
				},
			},
			expected: nil,
		},
		{
			name: "skip-bots as empty string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-bots": "",
				},
			},
			expected: nil,
		},
		{
			name: "on as string (no skip-bots possible)",
			frontmatter: map[string]any{
				"on": "push",
			},
			expected: nil,
		},
		{
			name:        "no on section",
			frontmatter: map[string]any{},
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractSkipBots(tt.frontmatter)
			assert.Equal(t, tt.expected, result, "extractSkipBots result mismatch")
		})
	}
}
