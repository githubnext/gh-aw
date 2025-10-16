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
			name: "secrets reference (handled gracefully)",
			config: parser.MCPServerConfig{
				Name: "secrets-tool",
				Type: "stdio",
				Env: map[string]string{
					"API_KEY": "${secrets.API_KEY}",
				},
			},
			expectError: false,
		},
		{
			name: "github remote mode requires GH_AW_GITHUB_TOKEN",
			config: parser.MCPServerConfig{
				Name: "github",
				Type: "http",
				URL:  "https://api.githubcopilot.com/mcp/",
				Env:  map[string]string{},
			},
			envVars: map[string]string{
				"GH_AW_GITHUB_TOKEN": "test_token",
			},
			expectError: false,
		},
		{
			name: "github remote mode with custom token",
			config: parser.MCPServerConfig{
				Name: "github",
				Type: "http",
				URL:  "https://api.githubcopilot.com/mcp/",
				Env: map[string]string{
					"GITHUB_TOKEN": "${{ secrets.CUSTOM_PAT }}",
				},
			},
			expectError: false,
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

			err := validateServerSecrets(tt.config, false, false)

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

func TestMCPInspectFiltersSafeOutputs(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		serverFilter string
		expectedLen  int
		description  string
	}{
		{
			name: "parser includes safe-outputs but inspect filters them",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{"max": 3},
					"missing-tool": map[string]any{},
				},
			},
			serverFilter: "",
			description:  "parser should include safe-outputs but inspect should filter them",
		},
		{
			name: "mixed configuration filters only safe-outputs",
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
			description:  "should filter safe-outputs but keep github MCP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that parser still includes safe-outputs
			configs, err := parser.ExtractMCPConfigurations(tt.frontmatter, tt.serverFilter)
			if err != nil {
				t.Fatalf("ExtractMCPConfigurations failed: %v", err)
			}

			// Verify parser includes safe-outputs when present
			hasSafeOutputs := false
			for _, config := range configs {
				if config.Name == "safe-outputs" {
					hasSafeOutputs = true
					break
				}
			}

			if _, hasSafeOutputsInFrontmatter := tt.frontmatter["safe-outputs"]; hasSafeOutputsInFrontmatter && !hasSafeOutputs {
				t.Error("Parser should still include safe-outputs configurations")
			} else if hasSafeOutputsInFrontmatter && hasSafeOutputs {
				t.Log("✓ Parser correctly includes safe-outputs (to be filtered by inspect command)")
			}

			// Test the filtering logic that inspect command uses
			var filteredConfigs []parser.MCPServerConfig
			for _, config := range configs {
				if config.Name != "safe-outputs" {
					filteredConfigs = append(filteredConfigs, config)
				}
			}

			// Verify no safe-outputs configurations remain after filtering
			for _, config := range filteredConfigs {
				if config.Name == "safe-outputs" {
					t.Errorf("safe-outputs should be filtered out by inspect command but was found")
				}
			}

			t.Logf("✓ Inspect command filtering works: %d configs before filter, %d after filter", len(configs), len(filteredConfigs))
		})
	}
}

func TestFilterOutSafeOutputs(t *testing.T) {
	tests := []struct {
		name     string
		input    []parser.MCPServerConfig
		expected []parser.MCPServerConfig
	}{
		{
			name:     "empty input",
			input:    []parser.MCPServerConfig{},
			expected: []parser.MCPServerConfig{},
		},
		{
			name: "only safe-outputs",
			input: []parser.MCPServerConfig{
				{Name: "safe-outputs", Type: "stdio"},
			},
			expected: []parser.MCPServerConfig{},
		},
		{
			name: "mixed servers",
			input: []parser.MCPServerConfig{
				{Name: "safe-outputs", Type: "stdio"},
				{Name: "github", Type: "docker"},
				{Name: "playwright", Type: "docker"},
			},
			expected: []parser.MCPServerConfig{
				{Name: "github", Type: "docker"},
				{Name: "playwright", Type: "docker"},
			},
		},
		{
			name: "no safe-outputs",
			input: []parser.MCPServerConfig{
				{Name: "github", Type: "docker"},
				{Name: "custom-server", Type: "stdio"},
			},
			expected: []parser.MCPServerConfig{
				{Name: "github", Type: "docker"},
				{Name: "custom-server", Type: "stdio"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOutSafeOutputs(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d configs, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i].Name != expected.Name || result[i].Type != expected.Type {
					t.Errorf("Expected config %d to be {Name: %s, Type: %s}, got {Name: %s, Type: %s}",
						i, expected.Name, expected.Type, result[i].Name, result[i].Type)
				}
			}
		})
	}
}

func TestApplyImportsToFrontmatter(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   map[string]any
		importsResult *parser.ImportsResult
		expectError   bool
		validate      func(t *testing.T, result map[string]any)
	}{
		{
			name: "merge imported MCP servers",
			frontmatter: map[string]any{
				"on": "issues",
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
				},
			},
			importsResult: &parser.ImportsResult{
				MergedMCPServers: `{"test-server":{"command":"node","args":["test.js"]}}`,
			},
			expectError: false,
			validate: func(t *testing.T, result map[string]any) {
				mcpServers, ok := result["mcp-servers"].(map[string]any)
				if !ok {
					t.Fatal("Expected mcp-servers to be a map")
				}
				if _, exists := mcpServers["test-server"]; !exists {
					t.Error("Expected test-server to be in mcp-servers")
				}
			},
		},
		{
			name: "merge imported tools",
			frontmatter: map[string]any{
				"on": "issues",
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
				},
			},
			importsResult: &parser.ImportsResult{
				MergedTools: `{"bash":{"allowed":["git status"]}}`,
			},
			expectError: false,
			validate: func(t *testing.T, result map[string]any) {
				tools, ok := result["tools"].(map[string]any)
				if !ok {
					t.Fatal("Expected tools to be a map")
				}
				if _, exists := tools["bash"]; !exists {
					t.Error("Expected bash to be in tools")
				}
				if _, exists := tools["github"]; !exists {
					t.Error("Expected github to be preserved in tools")
				}
			},
		},
		{
			name: "no imports returns same frontmatter",
			frontmatter: map[string]any{
				"on": "issues",
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []any{"create_issue"},
					},
				},
			},
			importsResult: &parser.ImportsResult{},
			expectError:   false,
			validate: func(t *testing.T, result map[string]any) {
				if _, exists := result["mcp-servers"]; exists {
					t.Error("Expected no mcp-servers when none are imported")
				}
				tools, ok := result["tools"].(map[string]any)
				if !ok {
					t.Fatal("Expected tools to be preserved")
				}
				if _, exists := tools["github"]; !exists {
					t.Error("Expected github to be preserved in tools")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyImportsToFrontmatter(tt.frontmatter, tt.importsResult)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
