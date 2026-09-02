package workflow

import (
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandADOToolConfig(t *testing.T) {
	tools := map[string]any{
		"ado": map[string]any{
			"organization": "contoso",
			"toolsets":     []any{"repos", "wit"},
			"allowed":      []any{"core_list_projects", "repo_repository", "wit_work_item"},
		},
	}

	require.NoError(t, expandADOToolConfig(tools))
	config := tools["ado"].(map[string]any)
	assert.Equal(t, "http", config["type"])
	assert.Equal(t, "https://mcp.dev.azure.com/contoso", config["url"])
	assert.Equal(t, []any{"core_list_projects", "repo_repository", "wit_work_item"}, config["allowed"])
	headers := config["headers"].(map[string]any)
	assert.Equal(t, "true", headers["X-MCP-Readonly"])
	assert.Equal(t, "repos,wit", headers["X-MCP-Toolsets"])
	assert.NotContains(t, config, "organization")
	assert.NotContains(t, config, "toolsets")
}

func TestExpandADOToolConfigWithoutOptionalFilters(t *testing.T) {
	tools := map[string]any{
		"ado": map[string]any{"organization": "contoso"},
	}

	require.NoError(t, expandADOToolConfig(tools))
	config := tools["ado"].(map[string]any)
	headers := config["headers"].(map[string]any)
	assert.Equal(t, map[string]any{"X-MCP-Readonly": "true"}, headers)
	assert.NotContains(t, config, "allowed")
}

func TestExpandADOToolConfigDisablesInheritedConfiguration(t *testing.T) {
	tools := map[string]any{"ado": false}
	require.NoError(t, expandADOToolConfig(tools))
	assert.NotContains(t, tools, "ado")
}

func TestValidateADOToolConfigRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  any
		message string
	}{
		{name: "not an object", config: true, message: "must be an object"},
		{name: "missing organization", config: map[string]any{}, message: "organization is required"},
		{name: "invalid organization", config: map[string]any{"organization": "https://dev.azure.com/contoso"}, message: "must contain only"},
		{name: "unsupported field", config: map[string]any{"organization": "contoso", "read-only": false}, message: "read-only is not supported"},
		{name: "empty toolsets", config: map[string]any{"organization": "contoso", "toolsets": []any{}}, message: "non-empty array"},
		{name: "unknown toolset", config: map[string]any{"organization": "contoso", "toolsets": []any{"boards"}}, message: `unsupported toolset "boards"`},
		{name: "all with another toolset", config: map[string]any{"organization": "contoso", "toolsets": []any{"all", "repos"}}, message: `only "all"`},
		{name: "duplicate toolset", config: map[string]any{"organization": "contoso", "toolsets": []any{"repos", "repos"}}, message: `duplicate toolset "repos"`},
		{name: "empty allowed", config: map[string]any{"organization": "contoso", "allowed": []any{}}, message: "non-empty array"},
		{name: "invalid allowed", config: map[string]any{"organization": "contoso", "allowed": []any{"wit-work-item"}}, message: `invalid tool name "wit-work-item"`},
		{name: "duplicate allowed", config: map[string]any{"organization": "contoso", "allowed": []any{"wit_work_item", "wit_work_item"}}, message: `duplicate tool "wit_work_item"`},
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

func TestADOToolIsParsedAsFirstClassTool(t *testing.T) {
	tools := NewTools(map[string]any{
		"ado": map[string]any{
			"organization": "contoso",
			"toolsets":     []any{"repos", "wit"},
			"allowed":      []any{"core_list_projects"},
		},
	})

	require.NotNil(t, tools.ADO)
	assert.Equal(t, "contoso", tools.ADO.Organization)
	assert.Equal(t, []string{"repos", "wit"}, tools.ADO.Toolsets)
	assert.Equal(t, []string{"core_list_projects"}, tools.ADO.Allowed)
	assert.NotContains(t, tools.Custom, "ado")
}

func TestADORequiredDomainsAreAdded(t *testing.T) {
	tools := map[string]any{
		"ado": map[string]any{
			"type": "http",
			"url":  "https://mcp.dev.azure.com/contoso",
		},
	}

	domains := GetAllowedDomainsForEngine(constants.CopilotEngine, &NetworkPermissions{}, tools, nil)
	for _, domain := range adoRequiredDomains {
		assert.Contains(t, domains, domain)
	}
	assert.Contains(t, domains, "mcp.dev.azure.com")
}

func TestADOCompilationGeneratesReadOnlyRemoteMCP(t *testing.T) {
	workflow := `---
on:
  workflow_dispatch:
permissions:
  contents: read
strict: false
engine: copilot
tools:
  ado:
    organization: contoso
    toolsets:
      - repos
      - wit
    allowed:
      - core_list_projects
      - repo_repository
      - wit_work_item
---

Read Azure DevOps data.
`
	file, err := os.CreateTemp("", "ado-tools-*.md")
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

	assert.Contains(t, compiled, "https://mcp.dev.azure.com/contoso")
	assert.Contains(t, compiled, `"X-MCP-Readonly": "true"`)
	assert.Contains(t, compiled, `"X-MCP-Toolsets": "repos,wit"`)
	assert.Contains(t, compiled, `"core_list_projects"`)
	assert.Contains(t, compiled, `"repo_repository"`)
	assert.Contains(t, compiled, `"wit_work_item"`)
	assert.NotContains(t, compiled, "read-only")
}
