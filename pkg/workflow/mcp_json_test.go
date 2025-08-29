package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetMCPConfigJSONString(t *testing.T) {
	tests := []struct {
		name       string
		toolConfig map[string]any
		expected   map[string]any
		wantErr    bool
	}{
		{
			name: "mcp as map",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type":    "stdio",
					"command": "python",
					"args":    []any{"-m", "test"},
				},
			},
			expected: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "test"},
			},
			wantErr: false,
		},
		{
			name: "mcp as JSON string",
			toolConfig: map[string]any{
				"mcp": `{"type": "stdio", "command": "python", "args": ["-m", "test"]}`,
			},
			expected: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "test"},
			},
			wantErr: false,
		},
		{
			name: "mcp as invalid JSON string",
			toolConfig: map[string]any{
				"mcp": `{"type": "stdio", "command": "python", invalid`,
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "no mcp section",
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expected: map[string]any{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getMCPConfig(tt.toolConfig, "test")

			if tt.wantErr != (err != nil) {
				t.Errorf("getMCPConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Convert expected to JSON and back to ensure proper comparison
				expectedJSON, _ := json.Marshal(tt.expected)
				resultJSON, _ := json.Marshal(result)

				if string(expectedJSON) != string(resultJSON) {
					t.Errorf("getMCPConfig() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestHasMCPConfigJSONString(t *testing.T) {
	tests := []struct {
		name       string
		toolConfig map[string]any
		expected   bool
		mcpType    string
	}{
		{
			name: "mcp as map with valid type",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type": "stdio",
				},
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "mcp as JSON string with valid type",
			toolConfig: map[string]any{
				"mcp": `{"type": "stdio", "command": "python"}`,
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "mcp as JSON string with invalid type",
			toolConfig: map[string]any{
				"mcp": `{"type": "invalid", "command": "python"}`,
			},
			expected: false,
			mcpType:  "",
		},
		{
			name: "mcp as invalid JSON string",
			toolConfig: map[string]any{
				"mcp": `{"type": "stdio", invalid`,
			},
			expected: false,
			mcpType:  "",
		},
		{
			name: "no mcp section",
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expected: false,
			mcpType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMcp, mcpType := hasMCPConfig(tt.toolConfig)

			if hasMcp != tt.expected {
				t.Errorf("hasMCPConfig() hasMcp = %v, want %v", hasMcp, tt.expected)
			}

			if mcpType != tt.mcpType {
				t.Errorf("hasMCPConfig() mcpType = %v, want %v", mcpType, tt.mcpType)
			}
		})
	}
}

func TestValidateMCPConfigs(t *testing.T) {
	tests := []struct {
		name    string
		tools   map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid MCP configs",
			tools: map[string]any{
				"trelloApi": map[string]any{
					"mcp": map[string]any{
						"type":    "stdio",
						"command": "python",
					},
					"allowed": []any{"create_card"},
				},
				"notionApi": map[string]any{
					"mcp":     `{"type": "http", "url": "https://mcp.notion.com"}`,
					"allowed": []any{"*"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON in MCP config",
			tools: map[string]any{
				"badApi": map[string]any{
					"mcp":     `{"type": "stdio", "command": "test", invalid json`,
					"allowed": []any{"*"},
				},
			},
			wantErr: true,
			errMsg:  "invalid JSON in mcp configuration",
		},
		{
			name: "missing type in MCP config",
			tools: map[string]any{
				"missingType": map[string]any{
					"mcp": map[string]any{
						"command": "python",
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "missing property 'type'",
		},
		{
			name: "missing type in JSON string MCP config",
			tools: map[string]any{
				"missingTypeJson": map[string]any{
					"mcp":     `{"command": "python"}`,
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "missing property 'type'",
		},
		{
			name: "invalid type in MCP config",
			tools: map[string]any{
				"invalidType": map[string]any{
					"mcp": map[string]any{
						"type":    "invalid",
						"command": "python",
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "value must be one of",
		},
		{
			name: "non-string type in MCP config",
			tools: map[string]any{
				"nonStringType": map[string]any{
					"mcp": map[string]any{
						"type":    123,
						"command": "python",
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "got number, want string",
		},
		{
			name: "http type missing URL",
			tools: map[string]any{
				"httpMissingUrl": map[string]any{
					"mcp": map[string]any{
						"type": "http",
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "missing property 'url'",
		},
		{
			name: "stdio type missing command",
			tools: map[string]any{
				"stdioMissingCommand": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "must specify either 'command' or 'container'",
		},
		{
			name: "http type with non-string URL",
			tools: map[string]any{
				"httpNonStringUrl": map[string]any{
					"mcp": map[string]any{
						"type": "http",
						"url":  123,
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "got number, want string",
		},
		{
			name: "stdio type with non-string command",
			tools: map[string]any{
				"stdioNonStringCommand": map[string]any{
					"mcp": map[string]any{
						"type":    "stdio",
						"command": []string{"python"},
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "got array, want string",
		},
		{
			name: "valid tools without MCP",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"ls", "cat"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed valid and invalid MCP configs",
			tools: map[string]any{
				"goodApi": map[string]any{
					"mcp":     `{"type": "stdio", "command": "good"}`,
					"allowed": []any{"tool1"},
				},
				"badApi": map[string]any{
					"mcp": map[string]any{
						"type": "http",
						// missing url
					},
					"allowed": []any{"tool2"},
				},
			},
			wantErr: true,
			errMsg:  "missing property 'url'",
		},
		{
			name: "network permissions with HTTP type should fail",
			tools: map[string]any{
				"httpWithNetPerms": map[string]any{
					"mcp": map[string]any{
						"type": "http",
						"url":  "https://example.com",
					},
					"permissions": map[string]any{
						"network": map[string]any{
							"allowed": []any{"example.com"},
						},
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "network egress permissions do not apply to remote 'type: http' servers",
		},
		{
			name: "network permissions with stdio non-container should fail",
			tools: map[string]any{
				"stdioNonContainerWithNetPerms": map[string]any{
					"mcp": map[string]any{
						"type":    "stdio",
						"command": "python",
					},
					"permissions": map[string]any{
						"network": map[string]any{
							"allowed": []any{"example.com"},
						},
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "network egress permissions only apply to stdio MCP servers that specify a 'container'",
		},
		{
			name: "network permissions with stdio container should pass",
			tools: map[string]any{
				"stdioContainerWithNetPerms": map[string]any{
					"mcp": map[string]any{
						"type":      "stdio",
						"container": "mcp/fetch",
					},
					"permissions": map[string]any{
						"network": map[string]any{
							"allowed": []any{"example.com"},
						},
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: false,
		},
		{
			name: "network permissions in mcp section with HTTP type should fail",
			tools: map[string]any{
				"httpWithMcpNetPerms": map[string]any{
					"mcp": map[string]any{
						"type": "http",
						"url":  "https://example.com",
						"permissions": map[string]any{
							"network": map[string]any{
								"allowed": []any{"example.com"},
							},
						},
					},
					"allowed": []any{"tool1"},
				},
			},
			wantErr: true,
			errMsg:  "network egress permissions do not apply to remote 'type: http' servers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPConfigs(tt.tools)

			if tt.wantErr != (err != nil) {
				t.Errorf("ValidateMCPConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateMCPConfigs() error = %v, expected to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}
