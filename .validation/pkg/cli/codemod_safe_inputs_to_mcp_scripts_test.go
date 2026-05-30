//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSafeInputsToMCPScriptsCodemod(t *testing.T) {
	codemod := getSafeInputsToMCPScriptsCodemod()

	assert.Equal(t, "safe-inputs-to-mcp-scripts", codemod.ID)
	assert.Equal(t, "Rename safe-inputs to mcp-scripts", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.3.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSafeInputsToMCPScriptsCodemod_RenamesKey(t *testing.T) {
	codemod := getSafeInputsToMCPScriptsCodemod()

	content := `---
on: workflow_dispatch
safe-inputs:
  tools:
    my-tool:
      description: A test tool
      script: |
        return "hello";
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-inputs": map[string]any{
			"tools": map[string]any{
				"my-tool": map[string]any{
					"description": "A test tool",
					"script":      "return \"hello\";",
				},
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "safe-inputs:", "safe-inputs key should be removed")
	assert.Contains(t, result, "mcp-scripts:", "mcp-scripts key should be present")
	assert.Contains(t, result, "my-tool:", "Tool definition should be preserved")
	assert.Contains(t, result, "A test tool", "Tool description should be preserved")
}

func TestSafeInputsToMCPScriptsCodemod_NoSafeInputsField(t *testing.T) {
	codemod := getSafeInputsToMCPScriptsCodemod()

	content := `---
on: workflow_dispatch
engine: copilot
---

# No safe-inputs`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Codemod should not be applied when safe-inputs is absent")
	assert.Equal(t, content, result, "Content should not be modified")
}

func TestSafeInputsToMCPScriptsCodemod_PreservesIndentedContent(t *testing.T) {
	codemod := getSafeInputsToMCPScriptsCodemod()

	content := `---
on: workflow_dispatch
safe-inputs:
  tools:
    gh:
      description: Run gh CLI
      run: gh $ARGS
      env:
        ARGS: ${{ inputs.args }}
    query:
      description: Query GitHub API
      script: return await octokit.request(inputs.path);
      inputs:
        path:
          type: string
          description: API path
          required: true
---`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"safe-inputs": map[string]any{
			"tools": map[string]any{},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "safe-inputs:", "safe-inputs key should be removed")
	assert.Contains(t, result, "mcp-scripts:", "mcp-scripts key should be present")
	// Check nested content is preserved
	assert.Contains(t, result, "gh:")
	assert.Contains(t, result, "query:")
	assert.Contains(t, result, "ARGS:")
	assert.Contains(t, result, "path:")
}

func TestSafeInputsToMCPScriptsCodemod_PreservesMarkdown(t *testing.T) {
	codemod := getSafeInputsToMCPScriptsCodemod()

	content := `---
on: workflow_dispatch
safe-inputs:
  tools:
    my-tool:
      description: A tool
      script: return 42;
---

# Workflow description

This workflow does something with safe-inputs tools.`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"safe-inputs": map[string]any{},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	// Frontmatter key is renamed
	assert.Contains(t, result, "mcp-scripts:")
	// Markdown body is preserved (references to safe-inputs in markdown are not modified by this codemod)
	assert.Contains(t, result, "# Workflow description")
	assert.Contains(t, result, "This workflow does something with safe-inputs tools.")
}
