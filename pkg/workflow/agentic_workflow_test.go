package workflow

import (
	"strings"
	"testing"
)

func TestAgenticWorkflowsSyntaxVariations(t *testing.T) {
	tests := []struct {
		name        string
		toolValue   any
		shouldWork  bool
		description string
	}{
		{
			name:        "agentic-workflows with nil (no value)",
			toolValue:   nil,
			shouldWork:  true,
			description: "Should enable agentic-workflows when field is present without value",
		},
		{
			name:        "agentic-workflows with true",
			toolValue:   true,
			shouldWork:  true,
			description: "Should enable agentic-workflows with boolean true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow with the agentic-workflows tool
			frontmatter := map[string]any{
				"on":    "workflow_dispatch",
				"tools": map[string]any{"agentic-workflows": tt.toolValue},
			}

			// Create compiler
			c := NewCompiler(false, "", "test")
			c.SetSkipValidation(true)

			// Extract tools from frontmatter
			tools := extractToolsFromFrontmatter(frontmatter)

			// Merge tools
			mergedTools, err := c.mergeToolsAndMCPServers(tools, make(map[string]any), "")
			if err != nil {
				if tt.shouldWork {
					t.Errorf("Expected tool to work but got error: %v", err)
				}
				return
			}

			if !tt.shouldWork {
				t.Errorf("Expected tool to fail but it succeeded")
				return
			}

			// Verify the agentic-workflows tool is present
			if _, exists := mergedTools["agentic-workflows"]; !exists {
				t.Errorf("Expected agentic-workflows tool to be present in merged tools")
			}
		})
	}
}

func TestAgenticWorkflowsMCPConfigGeneration(t *testing.T) {
	engines := []struct {
		name   string
		engine CodingAgentEngine
	}{
		{"Claude", NewClaudeEngine()},
		{"Copilot", NewCopilotEngine()},
		{"Custom", NewCustomEngine()},
		{"Codex", NewCodexEngine()},
	}

	for _, e := range engines {
		t.Run(e.name, func(t *testing.T) {
			// Create workflow data with agentic-workflows tool
			workflowData := &WorkflowData{
				Tools: map[string]any{
					"agentic-workflows": nil,
				},
			}

			// Generate MCP config
			var yaml strings.Builder
			mcpTools := []string{"agentic-workflows"}

			e.engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)
			result := yaml.String()

			// Verify the MCP config contains agentic-workflows
			if !strings.Contains(result, "agentic_workflows") {
				t.Errorf("Expected MCP config to contain 'agentic_workflows', got: %s", result)
			}

			// Verify it has the correct command
			if !strings.Contains(result, "gh") {
				t.Errorf("Expected MCP config to contain 'gh' command, got: %s", result)
			}

			// Verify it has the mcp-server argument
			if !strings.Contains(result, "mcp-server") {
				t.Errorf("Expected MCP config to contain 'mcp-server' argument, got: %s", result)
			}
		})
	}
}

func TestAgenticWorkflowsHasMCPServers(t *testing.T) {
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
	}

	if !HasMCPServers(workflowData) {
		t.Error("Expected HasMCPServers to return true for agentic-workflows tool")
	}
}
