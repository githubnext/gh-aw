//go:build !integration

package cli

import (
	"os"
	"path/filepath"
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
		assert.Contains(t, result, "  bash: false\n  cli-proxy: false")
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

	t.Run("rewrites github gh-proxy when bash is disabled", func(t *testing.T) {
		content := `---
tools:
  bash: false
  cli-proxy: false
  github:
    mode: gh-proxy
    read-only: true
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash":      false,
				"cli-proxy": false,
				"github":    map[string]any{"mode": "gh-proxy", "read-only": true},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "    mode: local")
		assert.NotContains(t, result, "mode: gh-proxy")
		assert.Contains(t, result, "    read-only: true")
	})

	t.Run("matches tools block with trailing comment", func(t *testing.T) {
		content := `---
tools: # security settings
  bash: false
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"bash": false},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "tools: # security settings")
		assert.Contains(t, result, "  cli-proxy: false")
	})

	t.Run("does not treat flow tools value as block header", func(t *testing.T) {
		content := `---
tools: {bash: false}
---

# Test
`
		frontmatter := map[string]any{
			"tools": map[string]any{"bash": false},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
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

	t.Run("apply with context detects imported bash restriction", func(t *testing.T) {
		dir := t.TempDir()
		importContent := `---
tools:
  bash: false
---
`
		importPath := filepath.Join(dir, "restricted-tools.md")
		require.NoError(t, os.WriteFile(importPath, []byte(importContent), 0o644))

		content := `---
imports:
  - restricted-tools.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
---

# Test
`
		workflowPath := filepath.Join(dir, "workflow.md")
		require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o644))

		frontmatter := map[string]any{
			"imports": []any{"restricted-tools.md"},
			"tools": map[string]any{
				"cli-proxy": true,
				"github":    map[string]any{"mode": "gh-proxy"},
			},
		}

		result, applied, err := codemod.ApplyWithContext(content, frontmatter, workflowPath)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  cli-proxy: false")
		assert.Contains(t, result, "    mode: local")
		assert.NotContains(t, result, "cli-proxy: true")
		assert.NotContains(t, result, "mode: gh-proxy")
	})
}
