//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlaywrightMCPModeToCLICodemod(t *testing.T) {
	codemod := getPlaywrightMCPModeToCLICodemod()

	assert.Equal(t, "playwright-mcp-mode-to-cli", codemod.ID)
	assert.Equal(t, "Migrate playwright MCP mode to CLI mode", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.9.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestPlaywrightMCPModeToCLICodemod_AddsModeWhenMissing(t *testing.T) {
	codemod := getPlaywrightMCPModeToCLICodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
---

# Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "playwright:\n    mode: cli")
}

func TestPlaywrightMCPModeToCLICodemod_ReplacesMCPMode(t *testing.T) {
	codemod := getPlaywrightMCPModeToCLICodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    mode: mcp
---

# Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"mode": "mcp",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "mode: cli")
	assert.NotContains(t, result, "mode: mcp")
}

func TestPlaywrightMCPModeToCLICodemod_NoChangeWhenAlreadyCLI(t *testing.T) {
	codemod := getPlaywrightMCPModeToCLICodemod()

	content := `---
on: workflow_dispatch
tools:
  playwright:
    mode: cli
---

# Workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"tools": map[string]any{
			"playwright": map[string]any{
				"mode": "cli",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
