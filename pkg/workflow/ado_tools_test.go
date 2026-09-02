package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandADOToolConfig(t *testing.T) {
	tools := map[string]any{
		"ado": map[string]any{
			"organization": "contoso",
			"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
			"toolsets":     []any{"repos", "wit"},
		},
	}

	require.NoError(t, expandADOToolConfig(tools))
	config := tools["ado"].(map[string]any)
	assert.Equal(t, "http", config["type"])
	assert.Equal(t, defaultADOMCPURL+"/contoso", config["url"])
	headers := config["headers"].(map[string]any)
	assert.Equal(t, "Bear"+"er ${{ secrets.ADO_MCP_AUTH_TOKEN }}", headers[adoAuthorizationName])
	assert.Equal(t, "true", headers[adoReadonlyHeader])
	assert.Equal(t, "repos,wit", headers[adoToolsetsHeader])
}

func TestExpandADOToolConfigAlwaysReadOnly(t *testing.T) {
	tools := map[string]any{
		"ado": map[string]any{
			"organization": "contoso",
			"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
		},
	}

	require.NoError(t, expandADOToolConfig(tools))
	headers := tools["ado"].(map[string]any)["headers"].(map[string]any)
	assert.Equal(t, "true", headers[adoReadonlyHeader])
	assert.NotContains(t, headers, adoToolsetsHeader)
}

func TestExpandADOToolConfigDisabled(t *testing.T) {
	tools := map[string]any{"ado": false}
	require.NoError(t, expandADOToolConfig(tools))
	assert.NotContains(t, tools, "ado")
}

func TestExpandADOToolConfigRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		message string
	}{
		{"missing organization", map[string]any{"token": "${{ secrets.TOKEN }}"}, "tools.ado.organization"},
		{"organization URL", map[string]any{"organization": "https://dev.azure.com/contoso", "token": "${{ secrets.TOKEN }}"}, "organization name"},
		{"literal token", map[string]any{"organization": "contoso", "token": "token"}, "direct GitHub Actions secret expression"},
		{"unknown field", map[string]any{"organization": "contoso", "token": "${{ secrets.TOKEN }}", "read-only": false}, "tools.ado.read-only is not supported"},
		{"empty toolsets", map[string]any{"organization": "contoso", "token": "${{ secrets.TOKEN }}", "toolsets": []any{}}, "non-empty array"},
		{"invalid toolset", map[string]any{"organization": "contoso", "token": "${{ secrets.TOKEN }}", "toolsets": []any{"issues"}}, `unsupported toolset "issues"`},
		{"duplicate toolset", map[string]any{"organization": "contoso", "token": "${{ secrets.TOKEN }}", "toolsets": []any{"repos", "repos"}}, `duplicate toolset "repos"`},
		{"all with restriction", map[string]any{"organization": "contoso", "token": "${{ secrets.TOKEN }}", "toolsets": []any{"all", "repos"}}, `only "all"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := map[string]any{"ado": tt.config}
			err := expandADOToolConfig(tools)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestADOToolTypedParsing(t *testing.T) {
	tools, err := ParseToolsConfig(map[string]any{
		"ado": map[string]any{
			"organization": "contoso",
			"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
			"toolsets":     []any{"repos", "wit"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, tools.ADO)
	assert.Equal(t, "contoso", tools.ADO.Organization)
	assert.Equal(t, []string{"repos", "wit"}, tools.ADO.Toolsets)
	assert.True(t, tools.HasTool("ado"))
	assert.Contains(t, tools.GetToolNames(), "ado")
	assert.Empty(t, tools.Custom)
}

func TestADOCompilationAcrossEngines(t *testing.T) {
	for _, engine := range []string{"copilot", "claude", "codex", "gemini", "pi"} {
		t.Run(engine, func(t *testing.T) {
			workflow := `---
on:
  workflow_dispatch:
strict: false
engine: ` + engine + `
tools:
  ado:
    organization: contoso
    token: ${{ secrets.ADO_MCP_AUTH_TOKEN }}
    toolsets:
      - repos
      - wit
---

Read Azure DevOps data.
`
			file, err := os.CreateTemp("", "ado-*.md")
			require.NoError(t, err)
			defer os.Remove(file.Name())
			_, err = file.WriteString(workflow)
			require.NoError(t, err)
			require.NoError(t, file.Close())

			compiler := NewCompiler()
			compiler.SetSkipValidation(true)
			data, err := compiler.ParseWorkflowFile(file.Name())
			require.NoError(t, err)
			compiled, _, _, err := compiler.generateYAML(data, file.Name())
			require.NoError(t, err)

			assert.Contains(t, compiled, defaultADOMCPURL+"/contoso")
			assert.Contains(t, compiled, adoReadonlyHeader)
			assert.Contains(t, compiled, "repos,wit")
			assert.Contains(t, compiled, "ADO_MCP_AUTH_TOKEN: ${{ secrets.ADO_MCP_AUTH_TOKEN }}")
			if engine == "codex" {
				assert.Contains(t, compiled, "Bear"+"er ${ADO_MCP_AUTH_TOKEN}")
			} else {
				assert.Contains(t, compiled, "Bear"+`er \${ADO_MCP_AUTH_TOKEN}`)
			}
			assert.NotContains(t, compiled, `"token": "${{ secrets.ADO_MCP_AUTH_TOKEN }}"`)
		})
	}
}
