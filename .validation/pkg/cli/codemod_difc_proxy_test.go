//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDIFCProxyToIntegrityProxyCodemod(t *testing.T) {
	codemod := getDIFCProxyToIntegrityProxyCodemod()

	t.Run("removes features.difc-proxy: true (no-op since proxy is now default)", func(t *testing.T) {
		content := `---
engine: copilot
features:
  difc-proxy: true
tools:
  github:
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"difc-proxy": true,
			},
			"tools": map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.NotContains(t, result, "difc-proxy", "Should remove features.difc-proxy")
		assert.NotContains(t, result, "integrity-proxy", "Should not add integrity-proxy when was true")
		assert.Contains(t, result, "min-integrity: approved", "Should preserve min-integrity")
	})

	t.Run("removes features.difc-proxy: false and adds tools.github.integrity-proxy: false", func(t *testing.T) {
		content := `---
engine: copilot
features:
  difc-proxy: false
tools:
  github:
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"difc-proxy": false,
			},
			"tools": map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.NotContains(t, result, "difc-proxy", "Should remove features.difc-proxy")
		assert.Contains(t, result, "integrity-proxy: false", "Should add integrity-proxy: false to tools.github")
		assert.Contains(t, result, "min-integrity: approved", "Should preserve min-integrity")
	})

	t.Run("removes features.difc-proxy: false but does NOT add integrity-proxy when tools.github is absent", func(t *testing.T) {
		content := `---
engine: copilot
features:
  difc-proxy: false
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"difc-proxy": false,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.NotContains(t, result, "difc-proxy", "Should remove features.difc-proxy")
		assert.NotContains(t, result, "integrity-proxy", "Should not add integrity-proxy when tools.github absent")
	})

	t.Run("removes features.difc-proxy: false but does NOT add integrity-proxy when tools.github is boolean true", func(t *testing.T) {
		content := `---
engine: copilot
features:
  difc-proxy: false
tools:
  github: true
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"difc-proxy": false,
			},
			"tools": map[string]any{
				"github": true,
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.NotContains(t, result, "difc-proxy", "Should remove features.difc-proxy")
		assert.NotContains(t, result, "integrity-proxy", "Should not add integrity-proxy when tools.github is boolean")
	})

	t.Run("does not modify workflows without features.difc-proxy", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  github:
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when features.difc-proxy absent")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("does not modify workflows without features section", func(t *testing.T) {
		content := `---
engine: copilot
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.False(t, applied, "Should not apply when features absent")
		assert.Equal(t, content, result, "Content should be unchanged")
	})

	t.Run("preserves other features when removing difc-proxy", func(t *testing.T) {
		content := `---
engine: copilot
features:
  action-mode: true
  difc-proxy: true
tools:
  github:
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"action-mode": true,
				"difc-proxy":  true,
			},
			"tools": map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.NotContains(t, result, "difc-proxy", "Should remove difc-proxy")
		assert.Contains(t, result, "action-mode: true", "Should preserve other features")
	})

	t.Run("integrity-proxy: false is added inside tools.github block", func(t *testing.T) {
		content := `---
engine: copilot
features:
  difc-proxy: false
tools:
  github:
    mode: local
    toolsets: [default]
    min-integrity: approved
    allowed-repos: all
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"difc-proxy": false,
			},
			"tools": map[string]any{
				"github": map[string]any{
					"mode":          "local",
					"toolsets":      []any{"default"},
					"min-integrity": "approved",
					"allowed-repos": "all",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.Contains(t, result, "integrity-proxy: false", "Should add integrity-proxy: false")
		assert.Contains(t, result, "min-integrity: approved", "Should preserve min-integrity")
		assert.Contains(t, result, "allowed-repos: all", "Should preserve allowed-repos")
		assert.NotContains(t, result, "difc-proxy", "Should remove difc-proxy from features")
		// Verify integrity-proxy uses same indentation as other fields (4-space context)
		assert.Contains(t, result, "    integrity-proxy: false", "integrity-proxy: false should use same indentation as other fields")
	})

	t.Run("string 'false' value treated as false (adds integrity-proxy: false)", func(t *testing.T) {
		content := `---
engine: copilot
features:
  difc-proxy: "false"
tools:
  github:
    min-integrity: approved
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"features": map[string]any{
				"difc-proxy": "false",
			},
			"tools": map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Should not error")
		assert.True(t, applied, "Should have applied the codemod")
		assert.NotContains(t, result, "difc-proxy", "Should remove features.difc-proxy")
		assert.Contains(t, result, "integrity-proxy: false", "String 'false' should be treated as false")
	})
}
