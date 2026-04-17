//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsUnknownToMCPServersCodemod(t *testing.T) {
	codemod := getToolsUnknownToMCPServersCodemod()

	t.Run("migrates serena list from tools to mcp-servers", func(t *testing.T) {
		content := `---
tools:
  serena: ['typescript']
---

# Workflow`

		frontmatter := map[string]any{
			"tools": map[string]any{
				"serena": []any{"typescript"},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should apply without errors")
		assert.True(t, modified, "Codemod should detect and migrate unknown tool entry")
		assert.NotContains(t, result, "  serena: ['typescript']", "Original tools.serena entry should be removed")
		assert.Contains(t, result, "mcp-servers:", "mcp-servers block should be created")
		assert.Contains(t, result, "  serena:", "Serena should be moved under mcp-servers")
		assert.Contains(t, result, "    command: \"...\" # TODO: fill in command", "Placeholder command should be inserted")
		assert.Contains(t, result, "    - typescript", "List value should be wrapped under tools list")
	})

	t.Run("migrates tavily object and keeps built-in tools intact", func(t *testing.T) {
		content := `---
tools:
  github: null
  tavily:
    tools: [search, search_news]
---

# Workflow`

		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": nil,
				"tavily": map[string]any{
					"tools": []any{"search", "search_news"},
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should apply without errors")
		assert.True(t, modified, "Codemod should migrate unknown tavily entry")
		assert.Contains(t, result, "tools:\n  github: null", "Built-in tools should remain in tools section")
		assert.Contains(t, result, "mcp-servers:", "mcp-servers block should be present")
		assert.Contains(t, result, "  tavily:", "Tavily should be moved under mcp-servers")
		assert.Contains(t, result, "    command: \"...\" # TODO: fill in command", "Command placeholder should be added when missing")
		assert.Contains(t, result, "    tools: [search, search_news]", "Existing tavily config should be preserved")
	})

	t.Run("does not modify recognized built-in tool names", func(t *testing.T) {
		content := `---
tools:
  github: null
  bash: true
---

# Workflow`

		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": nil,
				"bash":   true,
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not error on built-in tools")
		assert.False(t, modified, "Codemod should not modify built-in tool entries")
		assert.Equal(t, content, result, "Content should remain unchanged for built-in tool entries")
	})

	t.Run("appends migrated entries to existing mcp-servers", func(t *testing.T) {
		content := `---
tools:
  serena: ['typescript']
mcp-servers:
  existing:
    command: "node server.js"
---

# Workflow`

		frontmatter := map[string]any{
			"tools": map[string]any{
				"serena": []any{"typescript"},
			},
			"mcp-servers": map[string]any{
				"existing": map[string]any{
					"command": "node server.js",
				},
			},
		}

		result, modified, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should apply without errors")
		assert.True(t, modified, "Codemod should migrate unknown tool into existing mcp-servers block")
		assert.Contains(t, result, "  existing:", "Existing server config should be preserved")
		assert.Contains(t, result, "  serena:", "Migrated server should be appended to existing mcp-servers block")
	})
}
