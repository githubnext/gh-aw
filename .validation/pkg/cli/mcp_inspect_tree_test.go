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

func TestResolveWorkflowEngineID(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *workflow.WorkflowData
		want         string
	}{
		{
			name:         "nil workflow data",
			workflowData: nil,
			want:         "unknown",
		},
		{
			name: "engine config id",
			workflowData: &workflow.WorkflowData{
				EngineConfig: &workflow.EngineConfig{ID: "copilot"},
				AI:           "claude",
			},
			want: "copilot",
		},
		{
			name: "fallback to ai",
			workflowData: &workflow.WorkflowData{
				AI: "claude",
			},
			want: "claude",
		},
		{
			name:         "unknown",
			workflowData: &workflow.WorkflowData{},
			want:         "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveWorkflowEngineID(tt.workflowData))
		})
	}
}
