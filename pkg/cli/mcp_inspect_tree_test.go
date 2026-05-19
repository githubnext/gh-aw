package cli

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/types"
	"github.com/github/gh-aw/pkg/workflow"
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
		if !strings.Contains(result, part) {
			t.Fatalf("expected tree output to contain %q, got:\n%s", part, result)
		}
	}
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
			if got := resolveWorkflowEngineID(tt.workflowData); got != tt.want {
				t.Fatalf("resolveWorkflowEngineID() = %q, want %q", got, tt.want)
			}
		})
	}
}
