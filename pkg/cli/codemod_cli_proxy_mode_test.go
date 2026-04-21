//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliProxyFeatureToGitHubModeCodemod(t *testing.T) {
	codemod := getCliProxyFeatureToGitHubModeCodemod()

	t.Run("migrates features.cli-proxy true and adds tools.github.mode gh-proxy", func(t *testing.T) {
		content := `---
features:
  cli-proxy: true
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"cli-proxy": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "cli-proxy:")
		assert.Contains(t, result, "tools:")
		assert.Contains(t, result, "github:")
		assert.Contains(t, result, "mode: gh-proxy")
	})

	t.Run("does not apply when cli-proxy is false", func(t *testing.T) {
		content := `---
features:
  cli-proxy: false
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"cli-proxy": false,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("removes legacy flag but preserves existing github mode", func(t *testing.T) {
		content := `---
features:
  cli-proxy: true
tools:
  github:
    mode: local
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"cli-proxy": true,
			},
			"tools": map[string]any{
				"github": map[string]any{
					"mode": "local",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "cli-proxy:")
		assert.Contains(t, result, "mode: local")
		assert.NotContains(t, result, "mode: gh-proxy")
	})

	t.Run("adds github block under existing tools block", func(t *testing.T) {
		content := `---
features:
  cli-proxy: true
tools:
  playwright:
    version: v1.50.0
---

# Test
`
		frontmatter := map[string]any{
			"features": map[string]any{
				"cli-proxy": true,
			},
			"tools": map[string]any{
				"playwright": map[string]any{
					"version": "v1.50.0",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "playwright:")
		assert.Contains(t, result, "github:")
		assert.Contains(t, result, "mode: gh-proxy")
	})
}
