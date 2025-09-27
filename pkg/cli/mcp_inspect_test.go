package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

func TestMCPExtractIgnoresSafeOutputs(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		serverFilter string
		expectedLen  int
		description  string
	}{
		{
			name: "only safe-outputs configuration",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{"max": 3},
					"missing-tool": map[string]any{},
				},
			},
			serverFilter: "",
			expectedLen:  0,
			description:  "safe-outputs should be ignored",
		},
		{
			name: "only safe-jobs configuration",
			frontmatter: map[string]any{
				"safe-jobs": map[string]any{
					"custom-job": map[string]any{"enabled": true},
				},
			},
			serverFilter: "",
			expectedLen:  0,
			description:  "safe-jobs should be ignored",
		},
		{
			name: "mixed configuration with safe-outputs and github",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{"max": 3},
				},
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"create_issue", "get_repository"},
					},
				},
			},
			serverFilter: "",
			expectedLen:  1,
			description:  "only github MCP should be returned, safe-outputs should be ignored",
		},
		{
			name: "safe-outputs with specific server filter",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{"max": 3},
				},
			},
			serverFilter: "safe-outputs",
			expectedLen:  0,
			description:  "safe-outputs should be ignored even when specifically filtered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, err := parser.ExtractMCPConfigurations(tt.frontmatter, tt.serverFilter)
			if err != nil {
				t.Fatalf("ExtractMCPConfigurations failed: %v", err)
			}

			if len(configs) != tt.expectedLen {
				t.Errorf("Expected %d MCP configurations, got %d: %s", tt.expectedLen, len(configs), tt.description)
				for i, config := range configs {
					t.Logf("Config %d: %s (%s)", i, config.Name, config.Type)
				}
			}

			// Verify no safe-outputs configurations are returned
			for _, config := range configs {
				if config.Name == "safe-outputs" {
					t.Errorf("safe-outputs configuration should not be returned but was found")
				}
			}
		})
	}
}
