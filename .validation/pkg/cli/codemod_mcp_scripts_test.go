//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMCPScriptsModeCodemod(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	assert.Equal(t, "mcp-scripts-mode-removal", codemod.ID)
	assert.Equal(t, "Remove deprecated mcp-scripts.mode field", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.2.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestMCPScriptsModeCodemod_RemovesMode(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	content := `---
on: workflow_dispatch
mcp-scripts:
  mode: http
  max-size: 100KB
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"mcp-scripts": map[string]any{
			"mode":     "http",
			"max-size": "100KB",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "mode:")
	assert.Contains(t, result, "max-size: 100KB")
}

func TestMCPScriptsModeCodemod_NoMCPScriptsField(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMCPScriptsModeCodemod_NoModeField(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	content := `---
on: workflow_dispatch
mcp-scripts:
  max-size: 100KB
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"mcp-scripts": map[string]any{
			"max-size": "100KB",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMCPScriptsModeCodemod_PreservesIndentation(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	content := `---
on: workflow_dispatch
mcp-scripts:
  mode: http
  max-size: 100KB
  timeout: 30s
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"mcp-scripts": map[string]any{
			"mode":     "http",
			"max-size": "100KB",
			"timeout":  "30s",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "mode:")
	assert.Contains(t, result, "  max-size: 100KB")
	assert.Contains(t, result, "  timeout: 30s")
}

func TestMCPScriptsModeCodemod_PreservesComments(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	content := `---
on: workflow_dispatch
mcp-scripts:
  mode: http  # HTTP mode is now the default
  max-size: 100KB  # Maximum size for inputs
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"mcp-scripts": map[string]any{
			"mode":     "http",
			"max-size": "100KB",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "mode:")
	assert.Contains(t, result, "max-size: 100KB  # Maximum size for inputs")
}

func TestMCPScriptsModeCodemod_PreservesMarkdown(t *testing.T) {
	codemod := getMCPScriptsModeCodemod()

	content := `---
on: workflow_dispatch
mcp-scripts:
  mode: http
---

# Test Workflow

This workflow uses mcp-scripts.`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"mcp-scripts": map[string]any{
			"mode": "http",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# Test Workflow")
	assert.Contains(t, result, "This workflow uses mcp-scripts.")
}
