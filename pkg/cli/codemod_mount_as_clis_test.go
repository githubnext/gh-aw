//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMountAsCLIsToCLIProxyCodemod(t *testing.T) {
	codemod := getMountAsCLIsToCLIProxyCodemod()

	t.Run("renames tools.mount-as-clis to tools.cli-proxy", func(t *testing.T) {
		content := `---
tools:
  mount-as-clis: true
  playwright:
    version: v1.50.0
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"mount-as-clis": true,
				"playwright":    map[string]any{"version": "v1.50.0"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should apply codemod")
		assert.NotContains(t, result, "mount-as-clis:", "Should remove old key")
		assert.Contains(t, result, "cli-proxy: true", "Should add new key")
		assert.Contains(t, result, "playwright:", "Should preserve other tools")
	})

	t.Run("removes features.mcp-cli flag", func(t *testing.T) {
		content := `---
features:
  mcp-cli: true
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"mcp-cli": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should apply codemod")
		assert.NotContains(t, result, "mcp-cli:", "Should remove mcp-cli feature flag")
	})

	t.Run("renames mount-as-clis and removes mcp-cli together", func(t *testing.T) {
		content := `---
tools:
  mount-as-clis: true
features:
  mcp-cli: true
  integrity-reactions: true
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"mount-as-clis": true,
			},
			"features": map[string]any{
				"mcp-cli":             true,
				"integrity-reactions": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should apply codemod")
		assert.NotContains(t, result, "mount-as-clis:", "Should remove old tools key")
		assert.Contains(t, result, "cli-proxy: true", "Should add new tools key")
		assert.NotContains(t, result, "mcp-cli:", "Should remove mcp-cli feature flag")
		assert.Contains(t, result, "integrity-reactions:", "Should preserve other features")
	})

	t.Run("does not apply when neither key present", func(t *testing.T) {
		content := `---
tools:
  cli-proxy: true
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"cli-proxy": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when already migrated")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("does not rename mount-as-clis outside tools block", func(t *testing.T) {
		content := `---
tools:
  playwright: true
steps:
  - name: setup
    run: |
      mount-as-clis: true
---

# Body
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"playwright": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when key is not in tools block")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("preserves false value when renaming", func(t *testing.T) {
		content := `---
tools:
  mount-as-clis: false
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"mount-as-clis": false,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should apply codemod")
		assert.NotContains(t, result, "mount-as-clis:", "Should remove old key")
		assert.Contains(t, result, "cli-proxy: false", "Should preserve false value under new key")
	})
}
