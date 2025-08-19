package workflow

import (
	"strings"
	"testing"
)

func TestGenerateMCPLogCollection(t *testing.T) {
	tests := []struct {
		name          string
		tools         map[string]any
		expectedSteps []string
	}{
		{
			name: "with github tool only",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []string{"list_repos"},
				},
			},
			expectedSteps: []string{
				"Collect MCP server logs",
				"Upload MCP server logs",
				"mcp-github-${{ github.run_id }}",
				"docker logs mcp-github-${{ github.run_id }}",
				"/tmp/mcp-logs/github.log",
			},
		},
		{
			name: "with custom MCP tool",
			tools: map[string]any{
				"custom-server": map[string]any{
					"mcp": map[string]any{
						"type":    "stdio",
						"command": "echo",
						"args":    []string{"hello"},
					},
					"allowed": []string{"test_function"},
				},
			},
			expectedSteps: []string{
				"Collect MCP server logs",
				"Upload MCP server logs",
				"mcp-custom-server-${{ github.run_id }}",
				"docker logs mcp-custom-server-${{ github.run_id }}",
				"/tmp/mcp-logs/custom-server.log",
			},
		},
		{
			name: "with no MCP tools",
			tools: map[string]any{
				"regular-tool": map[string]any{
					"allowed": []string{"some_function"},
				},
			},
			expectedSteps: []string{},
		},
		{
			name: "with multiple MCP tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []string{"list_repos"},
				},
				"time": map[string]any{
					"mcp": map[string]any{
						"type":      "stdio",
						"container": "mcp/time",
					},
					"allowed": []string{"get_current_time"},
				},
			},
			expectedSteps: []string{
				"Collect MCP server logs",
				"Upload MCP server logs",
				"mcp-github-${{ github.run_id }}",
				"mcp-time-${{ github.run_id }}",
				"/tmp/mcp-logs/github.log",
				"/tmp/mcp-logs/time.log",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			var yaml strings.Builder
			
			// Create a mock engine for testing
			engine := NewClaudeEngine()
			
			compiler.generateMCPLogCollection(&yaml, tt.tools, engine)
			result := yaml.String()

			if len(tt.expectedSteps) == 0 {
				// Should generate no content for workflows without MCP tools
				if result != "" {
					t.Errorf("Expected no MCP log collection steps, but got: %s", result)
				}
				return
			}

			// Check that all expected content is present
			for _, expected := range tt.expectedSteps {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected to find '%s' in generated YAML, but it was missing.\nGenerated:\n%s", expected, result)
				}
			}

			// Verify the basic structure
			if !strings.Contains(result, "mkdir -p /tmp/mcp-logs") {
				t.Error("Expected MCP log collection to create log directory")
			}

			if !strings.Contains(result, "name: mcp-server-logs") {
				t.Error("Expected MCP log upload artifact to be named 'mcp-server-logs'")
			}

			if !strings.Contains(result, "if: always()") {
				t.Error("Expected MCP log collection steps to run always")
			}
		})
	}
}