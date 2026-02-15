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

// TestSkipUsersPreActivationJob tests that skip-users check is created correctly in pre-activation job
func TestSkipUsersPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-users-test")
	compiler := NewCompiler()

	t.Run("pre_activation_job_created_with_skip_users", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-users: [user1, user2, user3]
engine: copilot
---

# Skip Users Workflow

This workflow has a skip-users configuration.
`
		workflowFile := filepath.Join(tmpDir, "skip-users-workflow.md")
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

		// Verify skip-users check is present
		assert.Contains(t, lockContentStr, "Check skip-users", "Expected skip-users check to be present")

		// Verify the skip users environment variable is set correctly
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_USERS: user1,user2,user3", "Expected GH_AW_SKIP_USERS environment variable with correct value")

		// Verify the check_skip_users step ID is present
		assert.Contains(t, lockContentStr, "id: check_skip_users", "Expected check_skip_users step ID")

		// Verify the activated output includes skip_users_ok condition
		assert.Contains(t, lockContentStr, "steps.check_skip_users.outputs.skip_users_ok", "Expected activated output to include skip_users_ok condition")

		// Verify skip-users is commented out in the frontmatter
		assert.Contains(t, lockContentStr, "# skip-users:", "Expected skip-users to be commented out in lock file")
	})

	t.Run("skip_users_with_single_user", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-users: user1
engine: copilot
---

# Skip Users Single User

This workflow skips only for user1.
`
		workflowFile := filepath.Join(tmpDir, "skip-users-single.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify skip-users check is present
		assert.Contains(t, lockContentStr, "Check skip-users", "Expected skip-users check to be present")

		// Verify single user
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_USERS: user1", "Expected GH_AW_SKIP_USERS with single user")
	})

	t.Run("no_skip_users_no_check_created", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
engine: copilot
---

# No Skip Users Workflow

This workflow has no skip-users configuration.
`
		workflowFile := filepath.Join(tmpDir, "no-skip-users.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify skip-users check is NOT present
		assert.NotContains(t, lockContentStr, "Check skip-users", "Expected skip-users check to NOT be present")
		assert.NotContains(t, lockContentStr, "GH_AW_SKIP_USERS", "Expected GH_AW_SKIP_USERS to NOT be present")
		assert.NotContains(t, lockContentStr, "check_skip_users", "Expected check_skip_users step to NOT be present")
	})

	t.Run("skip_users_with_roles_field", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-users: [user1, user2]
roles: [maintainer]
engine: copilot
---

# Skip Users with Roles Field

This workflow has both roles and skip-users.
`
		workflowFile := filepath.Join(tmpDir, "skip-users-with-roles.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify both membership check and skip-users check are present
		assert.Contains(t, lockContentStr, "Check team membership", "Expected team membership check to be present")
		assert.Contains(t, lockContentStr, "Check skip-users", "Expected skip-users check to be present")

		// Verify GH_AW_REQUIRED_ROLES is set
		assert.Contains(t, lockContentStr, "GH_AW_REQUIRED_ROLES: maintainer", "Expected GH_AW_REQUIRED_ROLES for roles field")

		// Verify GH_AW_SKIP_USERS is set
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_USERS: user1,user2", "Expected GH_AW_SKIP_USERS for skip-users field")

		// Verify both conditions in activated output
		assert.Contains(t, lockContentStr, "steps.check_membership.outputs.is_team_member", "Expected membership check in activated output")
		assert.Contains(t, lockContentStr, "steps.check_skip_users.outputs.skip_users_ok", "Expected skip-users check in activated output")
	})

	t.Run("skip_users_and_skip_roles_combined", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-roles: [admin, write]
  skip-users: [user1, user2]
engine: copilot
---

# Skip Users and Skip Roles Combined

This workflow has both skip-roles and skip-users.
`
		workflowFile := filepath.Join(tmpDir, "skip-users-and-roles.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify both skip-roles and skip-users checks are present
		assert.Contains(t, lockContentStr, "Check skip-roles", "Expected skip-roles check to be present")
		assert.Contains(t, lockContentStr, "Check skip-users", "Expected skip-users check to be present")

		// Verify both environment variables are set
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES: admin,write", "Expected GH_AW_SKIP_ROLES for skip-roles field")
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_USERS: user1,user2", "Expected GH_AW_SKIP_USERS for skip-users field")

		// Verify both conditions in activated output
		assert.Contains(t, lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok", "Expected skip-roles check in activated output")
		assert.Contains(t, lockContentStr, "steps.check_skip_users.outputs.skip_users_ok", "Expected skip-users check in activated output")
	})
}

// TestExtractSkipUsers tests the extractSkipUsers function
func TestExtractSkipUsers(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []string
	}{
		{
			name: "skip-users as array of strings",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-users": []string{"user1", "user2"},
				},
			},
			expected: []string{"user1", "user2"},
		},
		{
			name: "skip-users as single string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-users": "user1",
				},
			},
			expected: []string{"user1"},
		},
		{
			name: "skip-users as array of any",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-users": []any{"user1", "user2", "user3"},
				},
			},
			expected: []string{"user1", "user2", "user3"},
		},
		{
			name: "no skip-users configured",
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
			name: "empty skip-users array",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-users": []string{},
				},
			},
			expected: nil,
		},
		{
			name: "skip-users as empty string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-users": "",
				},
			},
			expected: nil,
		},
		{
			name: "on as string (no skip-users possible)",
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
			result := compiler.extractSkipUsers(tt.frontmatter)
			assert.Equal(t, tt.expected, result, "extractSkipUsers result mismatch")
		})
	}
}
