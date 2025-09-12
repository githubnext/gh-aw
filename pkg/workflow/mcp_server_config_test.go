package workflow

import (
	"strings"
	"testing"
)

func TestMCPServerConfigProvider(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		expectCount  int
		expectTools  []string
		expectConfig map[string]string // tool name -> expected config type
	}{
		{
			name: "no tools",
			frontmatter: map[string]any{
				"on": map[string]any{"workflow_dispatch": nil},
			},
			expectCount: 0,
			expectTools: []string{},
		},
		{
			name: "github tool only",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
				},
			},
			expectCount: 1,
			expectTools: []string{"github"},
			expectConfig: map[string]string{
				"github": "stdio",
			},
		},
		{
			name: "custom MCP tool",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"notionApi": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "docker",
							"args":    []any{"run", "--rm", "-i", "mcp/notion"},
							"env": map[string]any{
								"NOTION_TOKEN": "{{ secrets.NOTION_TOKEN }}",
							},
						},
						"allowed": []any{"create_page"},
					},
				},
			},
			expectCount: 1,
			expectTools: []string{"notionApi"},
			expectConfig: map[string]string{
				"notionApi": "stdio",
			},
		},
		{
			name: "mixed tools",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
					"playwright": map[string]any{
						"allowed_domains": []any{"localhost"},
					},
					"customApi": map[string]any{
						"mcp": map[string]any{
							"type": "http",
							"url":  "https://api.example.com/mcp",
						},
						"allowed": []any{"get_data"},
					},
				},
			},
			expectCount: 3,
			expectTools: []string{"customApi", "github", "playwright"}, // sorted alphabetically
			expectConfig: map[string]string{
				"github":     "stdio",
				"playwright": "stdio",
				"customApi":  "http",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMCPServerConfigProvider()
			err := provider.ComputeMCPServerConfigurations(tt.frontmatter, nil)
			if err != nil {
				t.Fatalf("ComputeMCPServerConfigurations failed: %v", err)
			}

			// Check tool count
			if len(provider.GetMCPTools()) != tt.expectCount {
				t.Errorf("Expected %d tools, got %d", tt.expectCount, len(provider.GetMCPTools()))
			}

			// Check tool names
			tools := provider.GetMCPTools()
			if len(tools) != len(tt.expectTools) {
				t.Errorf("Expected tools %v, got %v", tt.expectTools, tools)
			} else {
				for i, expectedTool := range tt.expectTools {
					if tools[i] != expectedTool {
						t.Errorf("Expected tool[%d] = %s, got %s", i, expectedTool, tools[i])
					}
				}
			}

			// Check configurations
			configurations := provider.GetConfigurations()
			for _, config := range configurations {
				if expectedType, exists := tt.expectConfig[config.Name]; exists {
					if config.Type != expectedType {
						t.Errorf("Expected %s to have type %s, got %s", config.Name, expectedType, config.Type)
					}
				}
			}

			// Test conversion to parser configs
			parserConfigs := provider.ToParserMCPServerConfigs()
			if len(parserConfigs) != tt.expectCount {
				t.Errorf("Expected %d parser configs, got %d", tt.expectCount, len(parserConfigs))
			}
		})
	}
}

func TestMCPServerConfigurationRendering(t *testing.T) {
	// Test Claude rendering
	config := &MCPServerConfiguration{
		Name:    "testApi",
		Type:    "stdio",
		Command: "docker",
		Args:    []string{"run", "--rm", "-i", "test/api"},
		Env: map[string]string{
			"API_TOKEN": "{{ secrets.API_TOKEN }}",
		},
	}

	claudeResult, err := config.renderForClaude()
	if err != nil {
		t.Fatalf("renderForClaude failed: %v", err)
	}

	expectedSubstrings := []string{
		`"testApi": {`,
		`"command": "docker"`,
		`"run"`,
		`"--rm"`,
		`"-i"`,
		`"test/api"`,
		`"API_TOKEN": "{{ secrets.API_TOKEN }}"`,
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(claudeResult, substr) {
			t.Errorf("Expected Claude result to contain %q, but it didn't.\nResult: %s", substr, claudeResult)
		}
	}

	// Test Codex rendering
	codexResult, err := config.renderForCodex()
	if err != nil {
		t.Fatalf("renderForCodex failed: %v", err)
	}

	expectedCodexSubstrings := []string{
		"[mcp_servers.testApi]",
		`command = "docker"`,
		`args = [`,
		`"run",`,
		`"--rm",`,
		`"-i",`,
		`"test/api",`,
		`env = { "API_TOKEN" = "{{ secrets.API_TOKEN }}" }`,
	}

	for _, substr := range expectedCodexSubstrings {
		if !strings.Contains(codexResult, substr) {
			t.Errorf("Expected Codex result to contain %q, but it didn't.\nResult: %s", substr, codexResult)
		}
	}
}
