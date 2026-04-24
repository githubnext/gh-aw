//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerenaToSharedImportCodemod(t *testing.T) {
	codemod := getSerenaToSharedImportCodemod()

	t.Run("migrates tools.serena short syntax to imports", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  serena: ["go", "typescript"]
strict: false
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"serena": []any{"go", "typescript"},
			},
			"strict": false,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied when tools.serena is present")
		assert.NotContains(t, result, "serena:", "Codemod should remove tools.serena configuration")
		assert.Contains(t, result, "imports:", "Codemod should add imports block")
		assert.Contains(t, result, "- uses: shared/mcp/serena.md", "Codemod should add Serena shared import")
		assert.Contains(t, result, "languages: [\"go\", \"typescript\"]", "Codemod should preserve short syntax languages")

		parsed, parseErr := parser.ExtractFrontmatterFromContent(result)
		require.NoError(t, parseErr, "Result should contain valid frontmatter")
		_, hasTools := parsed.Frontmatter["tools"]
		assert.False(t, hasTools, "Codemod should remove empty tools key from frontmatter")
	})

	t.Run("migrates tools.serena long syntax languages object to imports", func(t *testing.T) {
		content := `---
engine: copilot
tools:
  serena:
    languages:
      go:
        version: "1.21"
      typescript:
strict: false
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"tools": map[string]any{
				"serena": map[string]any{
					"languages": map[string]any{
						"go": map[string]any{
							"version": "1.21",
						},
						"typescript": nil,
					},
				},
			},
			"strict": false,
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied for long syntax tools.serena")
		assert.NotContains(t, result, "serena:", "Codemod should remove tools.serena block")
		assert.Contains(t, result, "- uses: shared/mcp/serena.md", "Codemod should add Serena shared import")
		assert.Contains(t, result, "languages: [\"go\", \"typescript\"]", "Codemod should convert language object keys to array")
	})

	t.Run("removes tools.serena when shared import already exists without adding duplicate", func(t *testing.T) {
		content := `---
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go", "typescript"]
tools:
  serena: ["go", "typescript"]
  github:
    toolsets: [default]
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"imports": []any{
				map[string]any{
					"uses": "shared/mcp/serena.md",
					"with": map[string]any{
						"languages": []any{"go", "typescript"},
					},
				},
			},
			"tools": map[string]any{
				"serena": []any{"go", "typescript"},
				"github": map[string]any{
					"toolsets": []any{"default"},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied when tools.serena is present")
		assert.NotContains(t, result, "\n  serena:", "Codemod should remove tools.serena field")
		assert.Contains(t, result, "github:", "Codemod should preserve other tools.* entries")
		assert.Equal(t, 1, strings.Count(result, "shared/mcp/serena.md"), "Codemod should not add a duplicate Serena import")
	})

	t.Run("does not modify workflows without tools.serena", func(t *testing.T) {
		content := `---
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go"]
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": "copilot",
			"imports": []any{
				map[string]any{
					"uses": "shared/mcp/serena.md",
					"with": map[string]any{
						"languages": []any{"go"},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.False(t, applied, "Codemod should not be applied when tools.serena is absent")
		assert.Equal(t, content, result, "Content should remain unchanged when no migration is needed")
	})

	t.Run("migrates engine.tools.serena and preserves engine sibling fields", func(t *testing.T) {
		content := `---
engine:
  tools:
    serena:
      languages: ["typescript"]
  model: gpt-5.2
  id: copilot
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"tools": map[string]any{
					"serena": map[string]any{
						"languages": []any{"typescript"},
					},
				},
				"model": "gpt-5.2",
				"id":    "copilot",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied when engine.tools.serena is present")
		assert.Contains(t, result, "imports:", "Codemod should add imports block")
		assert.Contains(t, result, "languages: [\"typescript\"]", "Codemod should preserve engine.tools.serena languages")
		assert.NotContains(t, result, "\n  tools:", "Codemod should remove now-empty engine.tools block")

		parsed, parseErr := parser.ExtractFrontmatterFromContent(result)
		require.NoError(t, parseErr, "Result should contain valid frontmatter")
		engineAny, hasEngine := parsed.Frontmatter["engine"]
		require.True(t, hasEngine, "Result should preserve engine block")
		engine, ok := engineAny.(map[string]any)
		require.True(t, ok, "Engine should remain an object when sibling fields remain")
		assert.Equal(t, "gpt-5.2", engine["model"], "Engine model should remain under engine block")
		assert.Equal(t, "copilot", engine["id"], "Engine id should remain under engine block")
		_, hasEngineTools := engine["tools"]
		assert.False(t, hasEngineTools, "Engine tools should be removed after migration")
	})

	t.Run("updates github/gh-aw source pin from commit SHA to main when migrating serena", func(t *testing.T) {
		content := `---
source: github/gh-aw/.github/workflows/duplicate-code-detector.md@852cb06ad52958b402ed982b69957ffc57ca0619
engine: copilot
tools:
  serena: ["typescript"]
---

# Test Workflow
`
		frontmatter := map[string]any{
			"source": "github/gh-aw/.github/workflows/duplicate-code-detector.md@852cb06ad52958b402ed982b69957ffc57ca0619",
			"engine": "copilot",
			"tools": map[string]any{
				"serena": []any{"typescript"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied when tools.serena is present")
		assert.Contains(t, result, "source: github/gh-aw/.github/workflows/duplicate-code-detector.md@main", "Codemod should update pinned gh-aw source to main")
		assert.Contains(t, result, "- uses: shared/mcp/serena.md", "Codemod should still add shared Serena import")
	})

	t.Run("falls back to engine.tools.serena when top-level tools.serena is invalid", func(t *testing.T) {
		content := `---
engine:
  tools:
    serena:
      languages: ["typescript"]
  id: copilot
tools:
  serena: {}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"engine": map[string]any{
				"tools": map[string]any{
					"serena": map[string]any{
						"languages": []any{"typescript"},
					},
				},
				"id": "copilot",
			},
			"tools": map[string]any{
				"serena": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should fall back to engine.tools.serena when top-level tools.serena is invalid")
		assert.Contains(t, result, "- uses: shared/mcp/serena.md", "Codemod should add shared Serena import")
		assert.Contains(t, result, "languages: [\"typescript\"]", "Codemod should use languages from engine.tools.serena")
	})
}
