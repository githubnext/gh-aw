//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenCodeSharedImportCodemod(t *testing.T) {
	codemod := getOpenCodeSharedImportCodemod()

	t.Run("adds an import for scalar engine while preserving body and comments", func(t *testing.T) {
		content := `---
# OpenCode workflow
engine: opencode
model: copilot/claude-sonnet-4.5
---

# Workflow body
`

		result, applied, err := codemod.Apply(content, map[string]any{"engine": "opencode", "model": "copilot/claude-sonnet-4.5"})

		require.NoError(t, err)
		assert.True(t, applied)
		assert.Equal(t, `---
# OpenCode workflow
engine: opencode
imports:
  - github/gh-aw/.github/workflows/shared/opencode.md@main
model: copilot/claude-sonnet-4.5
---

# Workflow body`, result)
	})

	t.Run("adds an import for object engine", func(t *testing.T) {
		content := `---
engine:
  id: opencode
  display-name: OpenCode
---
`

		result, applied, err := codemod.Apply(content, map[string]any{
			"engine": map[string]any{"id": "opencode", "display-name": "OpenCode"},
		})

		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "imports:\n  - github/gh-aw/.github/workflows/shared/opencode.md@main")
	})

	t.Run("appends to existing imports", func(t *testing.T) {
		content := `---
engine: opencode
imports:
  - shared/mood.md
strict: true
---
`

		result, applied, err := codemod.Apply(content, map[string]any{
			"engine":  "opencode",
			"imports": []any{"shared/mood.md"},
			"strict":  true,
		})

		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "  - shared/mood.md\n  - github/gh-aw/.github/workflows/shared/opencode.md@main\nstrict: true")
	})

	t.Run("does not duplicate existing string or uses imports", func(t *testing.T) {
		for _, imports := range []any{
			[]any{"shared/opencode.md"},
			[]any{map[string]any{"uses": "shared/opencode.md"}},
			[]any{openCodeSharedImport},
			[]any{map[string]any{"uses": openCodeSharedImport}},
		} {
			content := `---
engine: opencode
imports:
  - shared/opencode.md
---
`
			result, applied, err := codemod.Apply(content, map[string]any{"engine": "opencode", "imports": imports})
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, content, result)
		}
	})

	t.Run("does not modify workflows using another engine", func(t *testing.T) {
		content := "---\nengine: copilot\n---\n"
		result, applied, err := codemod.Apply(content, map[string]any{"engine": "copilot"})

		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("is idempotent", func(t *testing.T) {
		content := "---\nengine: opencode\n---\n"
		first, applied, err := codemod.Apply(content, map[string]any{"engine": "opencode"})
		require.NoError(t, err)
		assert.True(t, applied)

		parsed, err := parser.ExtractFrontmatterFromContent(first)
		require.NoError(t, err)
		second, applied, err := codemod.Apply(first, parsed.Frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, first, second)
	})
}
