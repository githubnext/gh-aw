//go:build !integration

package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRunBranchesCodemod(t *testing.T) {
	originalResolveFn := resolveCurrentRepoDefaultBranchFn
	t.Cleanup(func() {
		resolveCurrentRepoDefaultBranchFn = originalResolveFn
	})

	codemod := getWorkflowRunBranchesCodemod()

	t.Run("adds current repository default branch for bare workflow_run trigger", func(t *testing.T) {
		resolveCurrentRepoDefaultBranchFn = func() (string, error) {
			return "trunk", nil
		}

		content := `---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
---

# Test
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_run": map[string]any{
					"workflows": []any{"CI"},
					"types":     []any{"completed"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied for bare workflow_run")
		assert.Contains(t, result, "branches:", "Codemod should add branches key")
		assert.Contains(t, result, "- trunk", "Codemod should add repository default branch")
		assert.NotContains(t, result, "- main", "Codemod should not add fallback branches when default branch is available")
		assert.NotContains(t, result, "- master", "Codemod should not add fallback branches when default branch is available")
	})

	t.Run("falls back to main and master when default branch cannot be resolved", func(t *testing.T) {
		resolveCurrentRepoDefaultBranchFn = func() (string, error) {
			return "", errors.New("api unavailable")
		}

		content := `---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
---

# Test
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_run": map[string]any{
					"workflows": []any{"CI"},
					"types":     []any{"completed"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied for bare workflow_run")
		assert.Contains(t, result, "- main", "Codemod should include main fallback branch")
		assert.Contains(t, result, "- master", "Codemod should include master fallback branch")
	})

	t.Run("does not modify workflow_run that already has branches", func(t *testing.T) {
		content := `---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches:
      - main
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_run": map[string]any{
					"workflows": []any{"CI"},
					"types":     []any{"completed"},
					"branches":  []any{"main"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.False(t, applied, "Codemod should not apply when branches already exist")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})

	t.Run("normalizeWorkflowRunBranches falls back when all values are empty", func(t *testing.T) {
		branches := normalizeWorkflowRunBranches([]string{"", "   "})
		assert.Equal(t, []string{"main", "master"}, branches, "Normalization should fall back to defaults for empty branch values")
	})
}
