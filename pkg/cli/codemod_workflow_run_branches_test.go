//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRunBranchesCodemod(t *testing.T) {
	codemod := getWorkflowRunBranchesCodemod()

	t.Run("adds branches for bare workflow_run trigger", func(t *testing.T) {
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
		assert.Contains(t, result, "- main", "Codemod should add main branch")
		assert.Contains(t, result, "- master", "Codemod should add master branch")
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
}
