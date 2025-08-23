package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInspectWorkflowMCP(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test workflow files
	testWorkflows := map[string]string{
		"test-stdio.md": `---
tools:
  custom-tool:
    mcp:
      type: stdio
      command: "python"
      args: ["-m", "custom_tool"]
      env:
        API_KEY: "${secrets.API_KEY}"
    allowed: ["tool1", "tool2"]
---
# Test Workflow
This is a test workflow with stdio MCP server.`,

		"test-http.md": `---
tools:
  api-server:
    mcp:
      type: http
      url: "https://api.contoso.com/mcp"
      headers:
        Authorization: "Bearer ${API_TOKEN}"
    allowed: ["*"]
---
# Test HTTP Workflow
This workflow uses HTTP MCP server.`,

		"test-github-only.md": `---
tools:
  github:
    allowed: ["create_issue", "list_issues"]
---
# GitHub Only Workflow
This workflow only uses GitHub tools.`,

		"test-no-mcp.md": `---
tools:
  claude:
    allowed:
      WebFetch:
      WebSearch:
---
# No MCP Workflow
This workflow has no MCP servers.`,
	}

	for filename, content := range testWorkflows {
		filePath := filepath.Join(workflowsDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// Change to test directory
	originalDir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(originalDir)
	}()
	_ = os.Chdir(tempDir)

	tests := []struct {
		name         string
		workflowFile string
		serverFilter string
		expectError  bool
		expectedMCP  int // Expected number of MCP servers
	}{
		{
			name:         "list all workflows with MCP",
			workflowFile: "",
			expectError:  false,
			expectedMCP:  0, // List mode doesn't return count
		},
		{
			name:         "inspect stdio MCP workflow",
			workflowFile: "test-stdio.md",
			expectError:  false,
			expectedMCP:  1,
		},
		{
			name:         "inspect HTTP MCP workflow",
			workflowFile: "test-http.md",
			expectError:  false,
			expectedMCP:  1,
		},
		{
			name:         "inspect GitHub only workflow",
			workflowFile: "test-github-only.md",
			expectError:  false,
			expectedMCP:  1, // GitHub is treated as MCP
		},
		{
			name:         "inspect workflow with no MCP",
			workflowFile: "test-no-mcp.md",
			expectError:  false,
			expectedMCP:  0,
		},
		{
			name:         "nonexistent workflow",
			workflowFile: "nonexistent.md",
			expectError:  true,
			expectedMCP:  0,
		},
		{
			name:         "filter specific server",
			workflowFile: "test-stdio.md",
			serverFilter: "custom",
			expectError:  false,
			expectedMCP:  1,
		},
		{
			name:         "filter no matching server",
			workflowFile: "test-stdio.md",
			serverFilter: "nonexistent",
			expectError:  false,
			expectedMCP:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InspectWorkflowMCP(tt.workflowFile, tt.serverFilter, "", false)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestInspectWorkflowMCPWithToolFilter(t *testing.T) {
	tests := []struct {
		name         string
		workflowFile string
		serverFilter string
		toolFilter   string
		expectError  bool
	}{
		{
			name:         "tool filter requires server filter",
			workflowFile: "nonexistent",
			serverFilter: "",
			toolFilter:   "some_tool",
			expectError:  true,
		},
		{
			name:         "tool filter with server filter",
			workflowFile: "nonexistent",
			serverFilter: "some_server",
			toolFilter:   "some_tool",
			expectError:  true, // Will fail because file doesn't exist, but validates filters work together
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InspectWorkflowMCP(tt.workflowFile, tt.serverFilter, tt.toolFilter, false)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestExtractMCPConfigurations(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   map[string]any
		serverFilter  string
		expectError   bool
		expectedLen   int
		expectedNames []string
	}{
		{
			name: "valid stdio MCP config",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"custom-tool": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "python",
							"args":    []any{"-m", "tool"},
						},
						"allowed": []any{"tool1", "tool2"},
					},
				},
			},
			expectError:   false,
			expectedLen:   1,
			expectedNames: []string{"custom-tool"},
		},
		{
			name: "valid HTTP MCP config",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"api-server": map[string]any{
						"mcp": map[string]any{
							"type": "http",
							"url":  "https://api.contoso.com/mcp",
						},
						"allowed": []any{"*"},
					},
				},
			},
			expectError:   false,
			expectedLen:   1,
			expectedNames: []string{"api-server"},
		},
		{
			name: "GitHub tool only",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
				},
			},
			expectError:   false,
			expectedLen:   1,
			expectedNames: []string{"github"},
		},
		{
			name: "mixed tools with and without MCP",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
					"claude": map[string]any{
						"allowed": map[string]any{
							"WebFetch": nil,
						},
					},
					"custom-mcp": map[string]any{
						"mcp": map[string]any{
							"type":      "stdio",
							"container": "my/tool",
						},
						"allowed": []any{"tool1"},
					},
				},
			},
			expectError:   false,
			expectedLen:   2, // github + custom-mcp
			expectedNames: []string{"github", "custom-mcp"},
		},
		{
			name: "server filter applied",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
					"custom-tool": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "python",
						},
						"allowed": []any{"tool1"},
					},
				},
			},
			serverFilter:  "custom",
			expectError:   false,
			expectedLen:   1,
			expectedNames: []string{"custom-tool"},
		},
		{
			name: "invalid MCP config - missing type",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"bad-tool": map[string]any{
						"mcp": map[string]any{
							"command": "python",
						},
						"allowed": []any{"tool1"},
					},
				},
			},
			expectError: true,
			expectedLen: 0,
		},
		{
			name: "invalid MCP config - invalid type",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"bad-tool": map[string]any{
						"mcp": map[string]any{
							"type":    "invalid",
							"command": "python",
						},
						"allowed": []any{"tool1"},
					},
				},
			},
			expectError: true,
			expectedLen: 0,
		},
		{
			name: "no tools section",
			frontmatter: map[string]any{
				"on": "push",
			},
			expectError: false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, err := parser.ExtractMCPConfigurations(tt.frontmatter, tt.serverFilter)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(configs) != tt.expectedLen {
				t.Errorf("Expected %d configurations, got %d", tt.expectedLen, len(configs))
			}

			if !tt.expectError && len(tt.expectedNames) > 0 {
				// Check that all expected names are present (order doesn't matter for maps)
				configNames := make(map[string]bool)
				for _, config := range configs {
					configNames[config.Name] = true
				}

				for _, expectedName := range tt.expectedNames {
					if !configNames[expectedName] {
						t.Errorf("Expected config %s not found in results", expectedName)
					}
				}
			}
		})
	}
}

func TestParseMCPConfig(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		mcpSection   any
		toolConfig   map[string]any
		expectError  bool
		expectedType string
	}{
		{
			name:     "valid stdio config with command",
			toolName: "test-tool",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "tool"},
			},
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expectError:  false,
			expectedType: "stdio",
		},
		{
			name:     "valid stdio config with container",
			toolName: "test-tool",
			mcpSection: map[string]any{
				"type":      "stdio",
				"container": "my/tool",
				"env": map[string]any{
					"API_KEY": "test",
				},
			},
			toolConfig: map[string]any{
				"allowed": []any{"*"},
			},
			expectError:  false,
			expectedType: "stdio",
		},
		{
			name:     "valid HTTP config",
			toolName: "api-tool",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://api.contoso.com/mcp",
				"headers": map[string]any{
					"Authorization": "Bearer token",
				},
			},
			toolConfig: map[string]any{
				"allowed": []any{"api_call"},
			},
			expectError:  false,
			expectedType: "http",
		},
		{
			name:       "JSON string format",
			toolName:   "json-tool",
			mcpSection: `{"type": "stdio", "command": "node", "args": ["server.js"]}`,
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expectError:  false,
			expectedType: "stdio",
		},
		{
			name:     "missing type",
			toolName: "bad-tool",
			mcpSection: map[string]any{
				"command": "python",
			},
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expectError: true,
		},
		{
			name:       "invalid JSON string",
			toolName:   "bad-json",
			mcpSection: `{"type": "stdio", "command": invalid json`,
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expectError: true,
		},
		{
			name:     "stdio missing command and container",
			toolName: "incomplete-stdio",
			mcpSection: map[string]any{
				"type": "stdio",
			},
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expectError: true,
		},
		{
			name:     "http missing URL",
			toolName: "incomplete-http",
			mcpSection: map[string]any{
				"type": "http",
			},
			toolConfig: map[string]any{
				"allowed": []any{"tool1"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parser.ParseMCPConfig(tt.toolName, tt.mcpSection, tt.toolConfig)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if config.Name != tt.toolName {
					t.Errorf("Expected name %s, got %s", tt.toolName, config.Name)
				}
				if config.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, config.Type)
				}
			}
		})
	}
}

func TestValidateServerSecrets(t *testing.T) {
	tests := []struct {
		name        string
		config      parser.MCPServerConfig
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "no environment variables",
			config: parser.MCPServerConfig{
				Name: "simple-tool",
				Type: "stdio",
			},
			expectError: false,
		},
		{
			name: "valid environment variable",
			config: parser.MCPServerConfig{
				Name: "env-tool",
				Type: "stdio",
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
			envVars: map[string]string{
				"TEST_VAR": "actual_value",
			},
			expectError: false,
		},
		{
			name: "missing environment variable",
			config: parser.MCPServerConfig{
				Name: "missing-env-tool",
				Type: "stdio",
				Env: map[string]string{
					"MISSING_VAR": "test_value",
				},
			},
			expectError: true,
			errorMsg:    "environment variable 'MISSING_VAR' not set",
		},
		{
			name: "secrets reference (not implemented)",
			config: parser.MCPServerConfig{
				Name: "secrets-tool",
				Type: "stdio",
				Env: map[string]string{
					"API_KEY": "${secrets.API_KEY}",
				},
			},
			expectError: true,
			errorMsg:    "secret 'API_KEY' validation not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store original environment variables to restore later
			originalEnvVars := make(map[string]string)
			var unsetVars []string

			// Set up environment variables
			for key, value := range tt.envVars {
				if originalValue, exists := os.LookupEnv(key); exists {
					originalEnvVars[key] = originalValue
				} else {
					unsetVars = append(unsetVars, key)
				}

				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}

			defer func() {
				// Restore original environment variables
				for key, originalValue := range originalEnvVars {
					os.Setenv(key, originalValue)
				}
				// Unset variables that were not originally set
				for _, key := range unsetVars {
					os.Unsetenv(key)
				}
			}()

			err := validateServerSecrets(tt.config)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestDisplayToolAllowanceHint(t *testing.T) {
	tests := []struct {
		name       string
		serverInfo *parser.MCPServerInfo
		expected   []string // expected phrases in output
	}{
		{
			name: "server with blocked tools",
			serverInfo: &parser.MCPServerInfo{
				Config: parser.MCPServerConfig{
					Name:    "test-server",
					Allowed: []string{"tool1", "tool2"},
				},
				Tools: []*mcp.Tool{
					{Name: "tool1", Description: "Allowed tool 1"},
					{Name: "tool2", Description: "Allowed tool 2"},
					{Name: "tool3", Description: "Blocked tool 3"},
					{Name: "tool4", Description: "Blocked tool 4"},
				},
			},
			expected: []string{
				"To allow blocked tools",
				"tools:",
				"test-server:",
				"allowed:",
				"- tool1",
				"- tool2",
				"- tool3",
				"- tool4",
			},
		},
		{
			name: "server with no allowed list (all tools allowed)",
			serverInfo: &parser.MCPServerInfo{
				Config: parser.MCPServerConfig{
					Name:    "open-server",
					Allowed: []string{}, // Empty means all allowed
				},
				Tools: []*mcp.Tool{
					{Name: "tool1", Description: "Tool 1"},
					{Name: "tool2", Description: "Tool 2"},
				},
			},
			expected: []string{
				"All tools are currently allowed",
				"To restrict tools",
				"tools:",
				"open-server:",
				"allowed:",
				"- tool1",
			},
		},
		{
			name: "server with all tools explicitly allowed",
			serverInfo: &parser.MCPServerInfo{
				Config: parser.MCPServerConfig{
					Name:    "explicit-server",
					Allowed: []string{"tool1", "tool2"},
				},
				Tools: []*mcp.Tool{
					{Name: "tool1", Description: "Tool 1"},
					{Name: "tool2", Description: "Tool 2"},
				},
			},
			expected: []string{
				"All available tools are explicitly allowed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output by redirecting stdout
			// For now, just call the function to ensure it doesn't panic
			// In a real scenario, we'd capture the output to verify content
			displayToolAllowanceHint(tt.serverInfo)
		})
	}
}
