package parser

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExtractMCPConfigurations(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		serverFilter string
		expected     []MCPServerConfig
		expectError  bool
	}{
		{
			name:        "Empty frontmatter",
			frontmatter: map[string]any{},
			expected:    []MCPServerConfig{},
		},
		{
			name: "No tools section",
			frontmatter: map[string]any{
				"name": "test-workflow",
				"on":   "push",
			},
			expected: []MCPServerConfig{},
		},
		{
			name: "GitHub tool default configuration",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "github",
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
						"ghcr.io/github/github-mcp-server:sha-09deac4",
					},
					Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}"},
					Allowed: []string{},
				},
			},
		},
		{
			name: "GitHub tool with custom configuration",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed":              []any{"issue_create", "pull_request_list"},
						"docker_image_version": "latest",
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "github",
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
						"ghcr.io/github/github-mcp-server:latest",
					},
					Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}"},
					Allowed: []string{"issue_create", "pull_request_list"},
				},
			},
		},
		{
			name: "Custom MCP server with stdio type",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"custom-server": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "/usr/local/bin/mcp-server",
							"args":    []any{"--config", "/etc/config.json"},
							"env": map[string]any{
								"API_KEY": "secret-key",
								"DEBUG":   "1",
							},
						},
						"allowed": []any{"tool1", "tool2"},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "custom-server",
					Type:    "stdio",
					Command: "/usr/local/bin/mcp-server",
					Args:    []string{"--config", "/etc/config.json"},
					Env: map[string]string{
						"API_KEY": "secret-key",
						"DEBUG":   "1",
					},
					Allowed: []string{"tool1", "tool2"},
				},
			},
		},
		{
			name: "Custom MCP server with container",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"docker-server": map[string]any{
						"mcp": map[string]any{
							"type":      "stdio",
							"container": "myregistry/mcp-server:v1.0",
							"env": map[string]any{
								"DATABASE_URL": "postgresql://localhost/db",
							},
						},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:      "docker-server",
					Type:      "stdio",
					Container: "myregistry/mcp-server:v1.0",
					Command:   "docker",
					Args:      []string{"run", "--rm", "-i", "-e", "DATABASE_URL", "myregistry/mcp-server:v1.0"},
					Env:       map[string]string{"DATABASE_URL": "postgresql://localhost/db"},
					Allowed:   []string{},
				},
			},
		},
		{
			name: "HTTP MCP server",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"http-server": map[string]any{
						"mcp": map[string]any{
							"type": "http",
							"url":  "https://api.example.com/mcp",
							"headers": map[string]any{
								"Authorization": "Bearer token123",
								"Content-Type":  "application/json",
							},
						},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name: "http-server",
					Type: "http",
					URL:  "https://api.example.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer token123",
						"Content-Type":  "application/json",
					},
					Env:     map[string]string{},
					Allowed: []string{},
				},
			},
		},
		{
			name: "MCP config as JSON string",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"json-server": map[string]any{
						"mcp": `{"type": "stdio", "command": "python", "args": ["-m", "server"]}`,
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "json-server",
					Type:    "stdio",
					Command: "python",
					Args:    []string{"-m", "server"},
					Env:     map[string]string{},
					Allowed: []string{},
				},
			},
		},
		{
			name: "Server filter - matching",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{},
					"custom": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "custom-server",
						},
					},
				},
			},
			serverFilter: "github",
			expected: []MCPServerConfig{
				{
					Name:    "github",
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
						"ghcr.io/github/github-mcp-server:sha-09deac4",
					},
					Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}"},
					Allowed: []string{},
				},
			},
		},
		{
			name: "Server filter - no match",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{},
					"custom": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "custom-server",
						},
					},
				},
			},
			serverFilter: "nomatch",
			expected:     []MCPServerConfig{},
		},
		{
			name: "Non-MCP tool ignored",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"regular-tool": map[string]any{
						"enabled": true,
					},
					"mcp-tool": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "mcp-server",
						},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "mcp-tool",
					Type:    "stdio",
					Command: "mcp-server",
					Env:     map[string]string{},
					Allowed: []string{},
				},
			},
		},
		{
			name: "Invalid tools section",
			frontmatter: map[string]any{
				"tools": "not a map",
			},
			expectError: true,
		},
		{
			name: "Invalid MCP config",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"invalid": map[string]any{
						"mcp": map[string]any{
							"type": "unsupported",
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMCPConfigurations(tt.frontmatter, tt.serverFilter)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d configs, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("Missing config at index %d", i)
					continue
				}

				actual := result[i]
				if actual.Name != expected.Name {
					t.Errorf("Config %d: expected name %q, got %q", i, expected.Name, actual.Name)
				}
				if actual.Type != expected.Type {
					t.Errorf("Config %d: expected type %q, got %q", i, expected.Type, actual.Type)
				}
				if actual.Command != expected.Command {
					t.Errorf("Config %d: expected command %q, got %q", i, expected.Command, actual.Command)
				}
				if !reflect.DeepEqual(actual.Args, expected.Args) {
					t.Errorf("Config %d: expected args %v, got %v", i, expected.Args, actual.Args)
				}
				// For GitHub configurations, just check that GITHUB_PERSONAL_ACCESS_TOKEN exists
				// The actual value depends on environment and may be a real token or placeholder
				if actual.Name == "github" {
					if _, hasToken := actual.Env["GITHUB_PERSONAL_ACCESS_TOKEN"]; !hasToken {
						t.Errorf("Config %d: GitHub config missing GITHUB_PERSONAL_ACCESS_TOKEN", i)
					}
				} else {
					if !reflect.DeepEqual(actual.Env, expected.Env) {
						t.Errorf("Config %d: expected env %v, got %v", i, expected.Env, actual.Env)
					}
				}
				// Compare allowed tools, handling nil vs empty slice equivalence
				actualAllowed := actual.Allowed
				if actualAllowed == nil {
					actualAllowed = []string{}
				}
				expectedAllowed := expected.Allowed
				if expectedAllowed == nil {
					expectedAllowed = []string{}
				}
				if !reflect.DeepEqual(actualAllowed, expectedAllowed) {
					t.Errorf("Config %d: expected allowed %v, got %v", i, expectedAllowed, actualAllowed)
				}
			}
		})
	}
}

func TestParseMCPConfig(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		mcpSection  any
		toolConfig  map[string]any
		expected    MCPServerConfig
		expectError bool
	}{
		{
			name:     "Stdio with command and args",
			toolName: "test-server",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": "/usr/bin/server",
				"args":    []any{"--verbose", "--config=/etc/config.yml"},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:    "test-server",
				Type:    "stdio",
				Command: "/usr/bin/server",
				Args:    []string{"--verbose", "--config=/etc/config.yml"},
				Env:     map[string]string{},
				Allowed: []string{},
			},
		},
		{
			name:     "Stdio with container",
			toolName: "docker-server",
			mcpSection: map[string]any{
				"type":      "stdio",
				"container": "myregistry/server:latest",
				"env": map[string]any{
					"DEBUG":   "1",
					"API_URL": "https://api.example.com",
				},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:      "docker-server",
				Type:      "stdio",
				Container: "myregistry/server:latest",
				Command:   "docker",
				Args:      []string{"run", "--rm", "-i", "-e", "DEBUG", "-e", "API_URL", "myregistry/server:latest"},
				Env: map[string]string{
					"DEBUG":   "1",
					"API_URL": "https://api.example.com",
				},
				Allowed: []string{},
			},
		},
		{
			name:     "HTTP server",
			toolName: "http-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://mcp.example.com/api",
				"headers": map[string]any{
					"Authorization": "Bearer token123",
					"User-Agent":    "gh-aw/1.0",
				},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name: "http-server",
				Type: "http",
				URL:  "https://mcp.example.com/api",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"User-Agent":    "gh-aw/1.0",
				},
				Env:     map[string]string{},
				Allowed: []string{},
			},
		},
		{
			name:     "With allowed tools",
			toolName: "server-with-allowed",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": "server",
			},
			toolConfig: map[string]any{
				"allowed": []any{"tool1", "tool2", "tool3"},
			},
			expected: MCPServerConfig{
				Name:    "server-with-allowed",
				Type:    "stdio",
				Command: "server",
				Env:     map[string]string{},
				Allowed: []string{"tool1", "tool2", "tool3"},
			},
		},
		{
			name:     "JSON string config",
			toolName: "json-server",
			mcpSection: `{
				"type": "stdio",
				"command": "python",
				"args": ["-m", "mcp_server"],
				"env": {
					"PYTHON_PATH": "/opt/python"
				}
			}`,
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:    "json-server",
				Type:    "stdio",
				Command: "python",
				Args:    []string{"-m", "mcp_server"},
				Env: map[string]string{
					"PYTHON_PATH": "/opt/python",
				},
				Allowed: []string{},
			},
		},
		{
			name:     "Stdio with environment variables",
			toolName: "env-server",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": "server",
				"env": map[string]any{
					"LOG_LEVEL": "debug",
					"PORT":      "8080",
				},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:    "env-server",
				Type:    "stdio",
				Command: "server",
				Env: map[string]string{
					"LOG_LEVEL": "debug",
					"PORT":      "8080",
				},
				Allowed: []string{},
			},
		},
		// Error cases
		{
			name:        "Missing type field",
			toolName:    "no-type",
			mcpSection:  map[string]any{"command": "server"},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:        "Invalid type",
			toolName:    "invalid-type",
			mcpSection:  map[string]any{"type": 123},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:        "Unsupported type",
			toolName:    "unsupported",
			mcpSection:  map[string]any{"type": "websocket"},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:        "Stdio missing command and container",
			toolName:    "no-command",
			mcpSection:  map[string]any{"type": "stdio"},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:        "HTTP missing URL",
			toolName:    "no-url",
			mcpSection:  map[string]any{"type": "http"},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:        "Invalid JSON string",
			toolName:    "invalid-json",
			mcpSection:  `{"invalid": json}`,
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:        "Invalid config format",
			toolName:    "invalid-format",
			mcpSection:  123,
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:     "Invalid command type",
			toolName: "invalid-command",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": 123, // Should be string
			},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:     "Invalid URL type",
			toolName: "invalid-url",
			mcpSection: map[string]any{
				"type": "http",
				"url":  123, // Should be string
			},
			toolConfig:  map[string]any{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMCPConfig(tt.toolName, tt.mcpSection, tt.toolConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Expected name %q, got %q", tt.expected.Name, result.Name)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Expected type %q, got %q", tt.expected.Type, result.Type)
			}
			if result.Command != tt.expected.Command {
				t.Errorf("Expected command %q, got %q", tt.expected.Command, result.Command)
			}
			if result.Container != tt.expected.Container {
				t.Errorf("Expected container %q, got %q", tt.expected.Container, result.Container)
			}
			if result.URL != tt.expected.URL {
				t.Errorf("Expected URL %q, got %q", tt.expected.URL, result.URL)
			}
			// For Docker containers, the environment variable order in args may vary
			// due to map iteration order, so check for presence rather than exact order
			if result.Container != "" {
				// Check that all expected elements are present in args
				expectedElements := make(map[string]bool)
				for _, arg := range tt.expected.Args {
					expectedElements[arg] = true
				}
				actualElements := make(map[string]bool)
				for _, arg := range result.Args {
					actualElements[arg] = true
				}
				if !reflect.DeepEqual(expectedElements, actualElements) {
					t.Errorf("Expected args elements %v, got %v", tt.expected.Args, result.Args)
				}
			} else {
				if !reflect.DeepEqual(result.Args, tt.expected.Args) {
					t.Errorf("Expected args %v, got %v", tt.expected.Args, result.Args)
				}
			}
			if !reflect.DeepEqual(result.Headers, tt.expected.Headers) {
				t.Errorf("Expected headers %v, got %v", tt.expected.Headers, result.Headers)
			}
			if !reflect.DeepEqual(result.Env, tt.expected.Env) {
				t.Errorf("Expected env %v, got %v", tt.expected.Env, result.Env)
			}
			// Compare allowed tools, handling nil vs empty slice equivalence
			actualAllowed := result.Allowed
			if actualAllowed == nil {
				actualAllowed = []string{}
			}
			expectedAllowed := tt.expected.Allowed
			if expectedAllowed == nil {
				expectedAllowed = []string{}
			}
			if !reflect.DeepEqual(actualAllowed, expectedAllowed) {
				t.Errorf("Expected allowed %v, got %v", expectedAllowed, actualAllowed)
			}
		})
	}
}

// TestMCPConfigTypes tests the struct types for proper JSON serialization
func TestMCPConfigTypes(t *testing.T) {
	// Test that our structs can be properly marshaled/unmarshaled
	config := MCPServerConfig{
		Name:    "test-server",
		Type:    "stdio",
		Command: "test-command",
		Args:    []string{"arg1", "arg2"},
		Env:     map[string]string{"KEY": "value"},
		Headers: map[string]string{"Content-Type": "application/json"},
		Allowed: []string{"tool1", "tool2"},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal config: %v", err)
	}

	// Unmarshal from JSON
	var decoded MCPServerConfig
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Errorf("Failed to unmarshal config: %v", err)
	}

	// Compare
	if !reflect.DeepEqual(config, decoded) {
		t.Errorf("Config changed after marshal/unmarshal cycle")
	}
}
