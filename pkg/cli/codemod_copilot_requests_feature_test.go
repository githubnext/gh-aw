//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopilotRequestsFeatureToPermissionsCodemod(t *testing.T) {
	codemod := getCopilotRequestsFeatureToPermissionsCodemod()

	t.Run("migrates enabled feature to permissions", func(t *testing.T) {
		content := `---
features:
  copilot-requests: true
permissions:
  contents: read
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"copilot-requests": true,
			},
			"permissions": map[string]any{
				"contents": "read",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "copilot-requests: true")
		assert.Contains(t, result, "copilot-requests: write")
	})

	t.Run("adds permissions block when missing", func(t *testing.T) {
		content := `---
features:
  copilot-requests: true
on:
  workflow_dispatch:
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"copilot-requests": true,
			},
			"on": map[string]any{
				"workflow_dispatch": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "permissions:")
		assert.Contains(t, result, "copilot-requests: write")
	})

	t.Run("removes disabled feature without adding permission", func(t *testing.T) {
		content := `---
features:
  copilot-requests: false
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"copilot-requests": false,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "copilot-requests: false")
		assert.NotContains(t, result, "copilot-requests: write")
	})

	t.Run("skips migration when permissions shorthand is not safely updatable", func(t *testing.T) {
		content := `---
features:
  copilot-requests: true
permissions: read-all
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"copilot-requests": true,
			},
			"permissions": "read-all",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("handles empty permissions object with inline comment", func(t *testing.T) {
		content := `---
features:
  copilot-requests: true
permissions: {} # empty
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"copilot-requests": true,
			},
			"permissions": map[string]any{},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "copilot-requests: true")
		assert.Contains(t, result, "permissions:")
		assert.Contains(t, result, "copilot-requests: write")
		assert.NotContains(t, result, "permissions: {} # empty")
	})
}
