//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/types"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestRenderMCPInspectionTree(t *testing.T) {
	workflowData := &workflow.WorkflowData{
		WorkflowID: "audit-workflows",
		EngineConfig: &workflow.EngineConfig{
			ID: "copilot",
		},
	}
	mcpConfigs := []parser.RegistryMCPServerConfig{
		{BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio"}, Name: "github"},
		{BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "http"}, Name: "playwright"},
	}

	result := renderMCPInspectionTree("/tmp/audit-workflows.md", workflowData, mcpConfigs)

	expected := []string{
		"Workflow: audit-workflows",
		"Engine: copilot",
		"MCP Servers",
		"github (stdio)",
		"playwright (http)",
	}
	for _, part := range expected {
		assert.Contains(t, result, part, "tree output should include expected hierarchy node")
	}
}

func TestRenderMCPInspectionTree_SortsServersDeterministically(t *testing.T) {
	workflowData := &workflow.WorkflowData{
		WorkflowID: "audit-workflows",
		EngineConfig: &workflow.EngineConfig{
			ID: "copilot",
		},
	}
	mcpConfigs := []parser.RegistryMCPServerConfig{
		{BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "http"}, Name: "playwright"},
		{BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio"}, Name: "github"},
		{BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "docker"}, Name: "github"},
	}

	result := renderMCPInspectionTree("/tmp/audit-workflows.md", workflowData, mcpConfigs)
	githubDockerIdx := strings.Index(result, "github (docker)")
	githubStdioIdx := strings.Index(result, "github (stdio)")
	playwrightIdx := strings.Index(result, "playwright (http)")

	assert.NotEqual(t, -1, githubDockerIdx)
	assert.NotEqual(t, -1, githubStdioIdx)
	assert.NotEqual(t, -1, playwrightIdx)
	assert.Less(t, githubDockerIdx, githubStdioIdx)
	assert.Less(t, githubStdioIdx, playwrightIdx)
}

func TestRenderMCPInspectionTree_UnknownEngineFallback(t *testing.T) {
	result := renderMCPInspectionTree("/tmp/audit-workflows.md", &workflow.WorkflowData{}, []parser.RegistryMCPServerConfig{
		{BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio"}, Name: "github"},
	})

	assert.Contains(t, result, "Engine: unknown")
}
