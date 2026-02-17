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

// TestSkipRolesPreActivationJob tests that skip-roles check is created correctly in pre-activation job
func TestSkipRolesPreActivationJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-roles-test")
	compiler := NewCompiler()

	t.Run("pre_activation_job_created_with_skip_roles", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-roles: [admin, maintainer, write]
engine: copilot
---

# Skip Roles Workflow

This workflow has a skip-roles configuration.
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

		// Verify pre_activation job exists
		assert.Contains(t, lockContentStr, "pre_activation:", "Expected pre_activation job to be created")

		// Verify skip-roles check is present
		assert.Contains(t, lockContentStr, "Check skip-roles", "Expected skip-roles check to be present")

		// Verify the skip roles environment variable is set correctly
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES: admin,maintainer,write", "Expected GH_AW_SKIP_ROLES environment variable with correct value")

		// Verify the check_skip_roles step ID is present
		assert.Contains(t, lockContentStr, "id: check_skip_roles", "Expected check_skip_roles step ID")

		// Verify the activated output includes skip_roles_ok condition
		assert.Contains(t, lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok", "Expected activated output to include skip_roles_ok condition")

		// Verify skip-roles is commented out in the frontmatter
		assert.Contains(t, lockContentStr, "# skip-roles:", "Expected skip-roles to be commented out in lock file")
	})

	t.Run("skip_roles_with_single_role", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-roles: admin
engine: copilot
---

# Skip Roles Single Role

This workflow skips only for admin role.
`
		workflowFile := filepath.Join(tmpDir, "skip-roles-single.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify skip-roles check is present
		assert.Contains(t, lockContentStr, "Check skip-roles", "Expected skip-roles check to be present")

		// Verify single role
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES: admin", "Expected GH_AW_SKIP_ROLES with single role")
	})

	t.Run("no_skip_roles_no_check_created", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
engine: copilot
---

# No Skip Roles Workflow

This workflow has no skip-roles configuration.
`
		workflowFile := filepath.Join(tmpDir, "no-skip-roles.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify skip-roles check is NOT present
		assert.NotContains(t, lockContentStr, "Check skip-roles", "Expected skip-roles check to NOT be present")
		assert.NotContains(t, lockContentStr, "GH_AW_SKIP_ROLES", "Expected GH_AW_SKIP_ROLES to NOT be present")
		assert.NotContains(t, lockContentStr, "check_skip_roles", "Expected check_skip_roles step to NOT be present")
	})

	t.Run("skip_roles_with_roles_field", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  skip-roles: [admin, write]
roles: [maintainer]
engine: copilot
---

# Skip Roles with Roles Field

This workflow has both roles and skip-roles.
`
		workflowFile := filepath.Join(tmpDir, "skip-roles-with-roles.md")
		err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		err = compiler.CompileWorkflow(workflowFile)
		require.NoError(t, err, "Compilation failed")

		lockFile := stringutil.MarkdownToLockFile(workflowFile)
		lockContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "Failed to read lock file")

		lockContentStr := string(lockContent)

		// Verify both membership check and skip-roles check are present
		assert.Contains(t, lockContentStr, "Check team membership", "Expected team membership check to be present")
		assert.Contains(t, lockContentStr, "Check skip-roles", "Expected skip-roles check to be present")

		// Verify GH_AW_REQUIRED_ROLES is set
		assert.Contains(t, lockContentStr, "GH_AW_REQUIRED_ROLES: maintainer", "Expected GH_AW_REQUIRED_ROLES for roles field")

		// Verify GH_AW_SKIP_ROLES is set
		assert.Contains(t, lockContentStr, "GH_AW_SKIP_ROLES: admin,write", "Expected GH_AW_SKIP_ROLES for skip-roles field")

		// Verify both conditions in activated output
		assert.Contains(t, lockContentStr, "steps.check_membership.outputs.is_team_member", "Expected membership check in activated output")
		assert.Contains(t, lockContentStr, "steps.check_skip_roles.outputs.skip_roles_ok", "Expected skip-roles check in activated output")
	})
}

// TestExtractSkipRoles tests the extractSkipRoles function
func TestExtractSkipRoles(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []string
	}{
		{
			name: "skip-roles as array of strings",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-roles": []string{"admin", "write"},
				},
			},
			expected: []string{"admin", "write"},
		},
		{
			name: "skip-roles as single string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-roles": "admin",
				},
			},
			expected: []string{"admin"},
		},
		{
			name: "skip-roles as array of any",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-roles": []any{"admin", "maintainer", "write"},
				},
			},
			expected: []string{"admin", "maintainer", "write"},
		},
		{
			name: "no skip-roles configured",
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
			name: "empty skip-roles array",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-roles": []string{},
				},
			},
			expected: nil,
		},
		{
			name: "skip-roles as empty string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"skip-roles": "",
				},
			},
			expected: nil,
		},
		{
			name: "on as string (no skip-roles possible)",
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
			result := compiler.extractSkipRoles(tt.frontmatter)
			assert.Equal(t, tt.expected, result, "extractSkipRoles result mismatch")
		})
	}
}
