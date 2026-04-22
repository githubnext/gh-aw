//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependabotPermissionsCodemod(t *testing.T) {
	codemod := getDependabotPermissionsCodemod()

	t.Run("adds missing vulnerability-alerts permission", func(t *testing.T) {
		content := `---
on:
  workflow_dispatch:
tools:
  github:
    toolsets: [dependabot]
permissions:
  contents: read
---
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_dispatch": map[string]any{},
			},
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets": []any{"dependabot"},
				},
			},
			"permissions": map[string]any{
				"contents": "read",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should apply when required permission is missing")
		assert.Contains(t, result, "vulnerability-alerts: read", "Codemod should add vulnerability-alerts permission")
	})

	t.Run("does not modify when permission already present", func(t *testing.T) {
		content := `---
tools:
  github:
    toolsets: [dependabot]
permissions:
  contents: read
  vulnerability-alerts: read
---
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{
					"toolsets": []any{"dependabot"},
				},
			},
			"permissions": map[string]any{
				"contents":             "read",
				"vulnerability-alerts": "read",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.False(t, applied, "Codemod should not apply when permission is already sufficient")
		assert.Equal(t, content, result, "Content should remain unchanged")
	})
}
