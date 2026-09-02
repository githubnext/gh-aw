package workflow

import (
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandLinearTool(t *testing.T) {
	t.Run("defaults to read-only and required", func(t *testing.T) {
		tools := map[string]any{
			"linear": map[string]any{
				"token":   "${{ secrets.LINEAR_API_KEY }}",
				"allowed": []any{"*"},
			},
		}

		require.NoError(t, expandLinearTool(tools))
		config := tools["linear"].(map[string]any)
		assert.Equal(t, "http", config["type"])
		assert.Equal(t, constants.LinearMCPReadOnlyURL, config["url"])
		assert.Equal(t, map[string]any{
			"Authorization": "Bearer " + "${{ secrets.LINEAR_API_KEY }}",
		}, config["headers"])
		assert.Equal(t, []any{"*"}, config["allowed"])
		assert.NotContains(t, config, "required")
	})

	t.Run("supports read-write and optional startup", func(t *testing.T) {
		tools := map[string]any{
			"linear": map[string]any{
				"token":     "${{ secrets.LINEAR_OAUTH_TOKEN }}",
				"read-only": false,
				"required":  false,
			},
		}

		require.NoError(t, expandLinearTool(tools))
		config := tools["linear"].(map[string]any)
		assert.Equal(t, constants.LinearMCPURL, config["url"])
		assert.Equal(t, false, config["required"])
	})

	for _, test := range []struct {
		name  string
		value any
		field string
	}{
		{name: "requires object", value: true, field: "tools.linear"},
		{name: "requires token", value: map[string]any{}, field: "tools.linear.token"},
		{name: "rejects literal token", value: map[string]any{"token": "lin_api_key"}, field: "tools.linear.token"},
		{name: "validates read-only", value: map[string]any{"token": "${{ secrets.LINEAR_API_KEY }}", "read-only": "yes"}, field: "tools.linear.read-only"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expandLinearTool(map[string]any{"linear": test.value})
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.field)
		})
	}
}

func TestLinearToolCompilation(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
strict: false
permissions:
  contents: read
engine: copilot
tools:
  bash: true
  cli-proxy: true
  linear:
    token: ${{ secrets.LINEAR_API_KEY }}
    allowed: ["*"]
---

# Linear test
`
	tmpFile, err := os.CreateTemp("", "linear-tool-*.md")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(workflowContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	compiler := NewCompiler()
	compiler.SetSkipValidation(true)
	workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
	require.NoError(t, err)
	yamlContent, _, _, err := compiler.generateYAML(workflowData, tmpFile.Name())
	require.NoError(t, err)

	assert.Contains(t, yamlContent, `"linear": {`)
	assert.Contains(t, yamlContent, `"url": "https://mcp.linear.app/mcp/readonly"`)
	expectedHeader := `"Authorization": "` + "Bearer" + ` \${LINEAR_API_KEY}"`
	assert.Contains(t, yamlContent, expectedHeader)
	assert.Contains(t, yamlContent, `LINEAR_API_KEY: ${{ secrets.LINEAR_API_KEY }}`)
	assert.Contains(t, yamlContent, `mcp.linear.app`)
	assert.Contains(t, yamlContent, "- `linear` — run `linear --help`")
	unexpandedHeader := `Authorization": "` + "Bearer" + ` ${{ secrets.LINEAR_API_KEY }}`
	assert.NotContains(t, yamlContent, unexpandedHeader)
}
