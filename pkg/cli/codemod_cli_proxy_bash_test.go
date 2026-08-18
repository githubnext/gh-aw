//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProxyBashDisabledCodemod(t *testing.T) {
	codemod := getCLIProxyBashDisabledCodemod()

	t.Run("adds cli-proxy false when bash is disabled", func(t *testing.T) {
		content := `---
tools:
  bash: false
  github:
    mode: local
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash":   false,
				"github": map[string]any{"mode": "local"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  cli-proxy: false")
		assert.Contains(t, result, "  bash: false")
	})

	t.Run("disables existing cli-proxy true", func(t *testing.T) {
		content := `---
tools:
  bash: false
  cli-proxy: true
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash":      false,
				"cli-proxy": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "cli-proxy: false")
		assert.NotContains(t, result, "cli-proxy: true")
	})

	t.Run("applies when bash allowlist is empty", func(t *testing.T) {
		content := `---
tools:
  bash: []
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash": []any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "cli-proxy: false")
	})

	t.Run("does not apply when bash is enabled", func(t *testing.T) {
		content := `---
tools:
  bash: ["cat"]
  cli-proxy: true
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash":      []any{"cat"},
				"cli-proxy": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not apply when cli-proxy is already false", func(t *testing.T) {
		content := `---
tools:
  bash: false
  cli-proxy: false
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash":      false,
				"cli-proxy": false,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("is idempotent", func(t *testing.T) {
		content := `---
tools:
  bash: false
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"bash": false},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		require.True(t, applied)

		updatedFrontmatter := map[string]any{
			"tools": map[string]any{"bash": false, "cli-proxy": false},
		}
		second, applied, err := codemod.Apply(result, updatedFrontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, result, second)
	})
}
