//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubModeHomogeneousEnumCodemod(t *testing.T) {
	codemod := getGitHubModeHomogeneousEnumCodemod()

	t.Run("rewrites mode gh-proxy to cli", func(t *testing.T) {
		content := `---
tools:
  github:
    mode: gh-proxy
    allowed-repos: [owner/repo]
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{
					"mode":          "gh-proxy",
					"allowed-repos": []any{"owner/repo"},
				},
			},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "gh-proxy")
		assert.Contains(t, result, "mode: cli")
		assert.Contains(t, result, "allowed-repos:")
	})

	t.Run("rewrites mode local/remote to mcp-local/mcp-remote", func(t *testing.T) {
		for value, expected := range map[string]string{"local": "mcp-local", "remote": "mcp-remote"} {
			content := "---\ntools:\n  github:\n    mode: " + value + "\n    toolsets: [all]\n---\n\n# Test\n"
			frontmatter := map[string]any{
				"tools": map[string]any{
					"github": map[string]any{"mode": value, "toolsets": []any{"all"}},
				},
			}
			result, applied, err := codemod.Apply(content, frontmatter)
			require.NoError(t, err)
			assert.True(t, applied, "value %s should migrate", value)
			assert.Contains(t, result, "mode: "+expected)
			assert.Contains(t, result, "toolsets:")
		}
	})

	t.Run("folds legacy type into mode", func(t *testing.T) {
		content := `---
tools:
  github:
    type: remote
    toolsets: [all]
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{"type": "remote", "toolsets": []any{"all"}},
			},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "type: remote")
		assert.Contains(t, result, "mode: mcp-remote")
	})

	t.Run("removes duplicate type when mode is present", func(t *testing.T) {
		content := `---
tools:
  github:
    mode: local
    type: remote
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{"mode": "local", "type": "remote"},
			},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "type:")
		assert.Equal(t, 1, strings.Count(result, "mode:"))
		assert.Contains(t, result, "mode: mcp-local")
	})

	t.Run("removes legacy type beside homogeneous mode", func(t *testing.T) {
		content := `---
tools:
  github:
    type: remote
    mode: cli
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{"mode": "cli", "type": "remote"},
			},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "type:")
		assert.Equal(t, 1, strings.Count(result, "mode:"))
		assert.Contains(t, result, "mode: cli")
	})

	t.Run("does not apply for already-homogeneous values", func(t *testing.T) {
		content := `---
tools:
  github:
    mode: cli
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"github": map[string]any{"mode": "cli"}},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("preserves trailing comment on mode line", func(t *testing.T) {
		content := `---
tools:
  github:
    mode: gh-proxy  # pre-authenticated gh
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"github": map[string]any{"mode": "gh-proxy"}},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "mode: cli")
		assert.Contains(t, result, "# pre-authenticated gh")
	})
}

func TestCliProxyToMCPModeCodemod(t *testing.T) {
	codemod := getCliProxyToMCPModeCodemod()

	t.Run("rewrites cli-proxy true to mcp-mode cli", func(t *testing.T) {
		content := `---
tools:
  cli-proxy: true
  github:
    mode: cli
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"cli-proxy": true,
				"github":    map[string]any{"mode": "cli"},
			},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "cli-proxy:")
		assert.Contains(t, result, "mcp-mode: cli")
		assert.Contains(t, result, "github:")
	})

	t.Run("rewrites cli-proxy false to mcp-mode default", func(t *testing.T) {
		content := `---
tools:
  cli-proxy: false
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"cli-proxy": false},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "mcp-mode: default")
	})

	t.Run("does not apply when cli-proxy absent", func(t *testing.T) {
		content := `---
tools:
  mcp-mode: cli
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"mcp-mode": "cli"},
		}
		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})
}

func TestCliProxyFeatureToGitHubModeCodemodProducesCLI(t *testing.T) {
	codemod := getCliProxyFeatureToGitHubModeCodemod()
	content := `---
tools:
  github:
    toolsets: [default]
features:
  cli-proxy: true
---

# Test
`
	frontmatter := map[string]any{
		"tools": map[string]any{
			"github": map[string]any{"toolsets": []any{"default"}},
		},
		"features": map[string]any{"cli-proxy": true},
	}
	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "cli-proxy:")
	assert.Contains(t, result, "mode: cli")
}
