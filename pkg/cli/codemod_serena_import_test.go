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
}
