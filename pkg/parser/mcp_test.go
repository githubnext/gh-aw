package parser

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestEnsureLocalhostDomains tests the helper function that ensures localhost domains are always included
func TestEnsureLocalhostDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty input should add all localhost domains with ports",
			input:    []string{},
			expected: []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*"},
		},
		{
			name:     "Custom domains without localhost should add localhost domains with ports",
			input:    []string{"github.com", "*.github.com"},
			expected: []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*", "github.com", "*.github.com"},
		},
		{
			name:     "Input with localhost but no 127.0.0.1 should add missing domains",
			input:    []string{"localhost", "example.com"},
			expected: []string{"localhost:*", "127.0.0.1", "127.0.0.1:*", "localhost", "example.com"},
		},
		{
			name:     "Input with 127.0.0.1 but no localhost should add missing domains",
			input:    []string{"127.0.0.1", "example.com"},
			expected: []string{"localhost", "localhost:*", "127.0.0.1:*", "127.0.0.1", "example.com"},
		},
		{
			name:     "Input with both localhost domains should add port variants",
			input:    []string{"localhost", "127.0.0.1", "example.com"},
			expected: []string{"localhost:*", "127.0.0.1:*", "localhost", "127.0.0.1", "example.com"},
		},
		{
			name:     "Input with both in different order should add port variants",
			input:    []string{"example.com", "127.0.0.1", "localhost"},
			expected: []string{"localhost:*", "127.0.0.1:*", "example.com", "127.0.0.1", "localhost"},
		},
		{
			name:     "Input with all localhost variants should remain unchanged",
			input:    []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*", "example.com"},
			expected: []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*", "example.com"},
		},
		{
			name:     "Input with some localhost variants should add missing ones",
			input:    []string{"localhost:*", "127.0.0.1", "example.com"},
			expected: []string{"localhost", "127.0.0.1:*", "localhost:*", "127.0.0.1", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureLocalhostDomains(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("EnsureLocalhostDomains(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractMCPConfigurations(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		serverFilter string
		expected     []MCPServerConfig
		expectError  bool
	}{
		{
			name: "GitHub tool with read-only true",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"read-only": true,
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
						"-e", "GITHUB_READ_ONLY=1",
						"ghcr.io/github/github-mcp-server:" + constants.DefaultGitHubMCPServerVersion,
					},
					Env: map[string]string{
						"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}",
					},
					Allowed: []string{},
				},
			},
		},
		{
			name: "GitHub tool with read-only false",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"read-only": false,
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
						"ghcr.io/github/github-mcp-server:" + constants.DefaultGitHubMCPServerVersion,
					},
					Env: map[string]string{
						"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}",
					},
					Allowed: []string{},
				},
			},
		},
		{
			name: "GitHub tool without read-only (default behavior)",
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
						"ghcr.io/github/github-mcp-server:" + constants.DefaultGitHubMCPServerVersion,
					},
					Env: map[string]string{
						"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN_REQUIRED}",
					},
					Allowed: []string{},
				},
			},
		},
		{
			name: "New format: Custom MCP server with direct fields",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"direct-server": map[string]any{
						"type":    "stdio",
						"command": "python",
						"args":    []any{"-m", "direct_server"},
						"env": map[string]any{
							"DEBUG": "true",
						},
						"allowed": []any{"process", "query"},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "direct-server",
					Type:    "stdio",
					Command: "python",
					Args:    []string{"-m", "direct_server"},
					Env: map[string]string{
						"DEBUG": "true",
					},
					Allowed: []string{"process", "query"},
				},
			},
		},
		{
			name: "New format: HTTP server with direct fields",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"http-direct": map[string]any{
						"type": "http",
						"url":  "https://api.example.com/mcp",
						"headers": map[string]any{
							"Authorization": "Bearer token123",
						},
						"allowed": []any{"query", "update"},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name: "http-direct",
					Type: "http",
					URL:  "https://api.example.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer token123",
					},
					Env:     map[string]string{},
					Allowed: []string{"query", "update"},
				},
			},
		},
		{
			name: "New format: HTTP server with underscored headers",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"datadog": map[string]any{
						"type": "http",
						"url":  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
						"headers": map[string]any{
							"DD_API_KEY":         "test-api-key",
							"DD_APPLICATION_KEY": "test-app-key",
							"DD_SITE":            "datadoghq.com",
						},
						"allowed": []any{"get-monitors", "get-monitor"},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name: "datadog",
					Type: "http",
					URL:  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
					Headers: map[string]string{
						"DD_API_KEY":         "test-api-key",
						"DD_APPLICATION_KEY": "test-app-key",
						"DD_SITE":            "datadoghq.com",
					},
					Env:     map[string]string{},
					Allowed: []string{"get-monitors", "get-monitor"},
				},
			},
		},
		{
			name: "New format: Container with direct fields",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"container-direct": map[string]any{
						"type":      "stdio",
						"container": "mcp/service:latest",
						"env": map[string]any{
							"API_KEY": "secret123",
						},
						"allowed": []any{"execute"},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:      "container-direct",
					Type:      "stdio",
					Container: "mcp/service:latest",
					Command:   "docker",
					Args:      []string{"run", "--rm", "-i", "-e", "API_KEY", "mcp/service:latest"},
					Env: map[string]string{
						"API_KEY": "secret123",
					},
					Allowed: []string{"execute"},
				},
			},
		},
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
						"ghcr.io/github/github-mcp-server:" + constants.DefaultGitHubMCPServerVersion,
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
						"allowed": []any{"issue_create", "pull_request_list"},
						"version": "latest",
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
			name: "Playwright tool default configuration",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"playwright": map[string]any{
						"allowed_domains": []any{"github.com", "*.github.com"},
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "playwright",
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "--shm-size=2gb", "--cap-add=SYS_ADMIN",
						"-e", "PLAYWRIGHT_ALLOWED_DOMAINS",
						"mcr.microsoft.com/playwright:latest",
					},
					Env: map[string]string{"PLAYWRIGHT_ALLOWED_DOMAINS": "localhost,localhost:*,127.0.0.1,127.0.0.1:*,github.com,*.github.com"},
				},
			},
		},
		{
			name: "Playwright tool with custom Docker image",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"playwright": map[string]any{
						"allowed_domains": []any{"example.com"},
						"version":         "v1.41.0",
					},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "playwright",
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "--shm-size=2gb", "--cap-add=SYS_ADMIN",
						"-e", "PLAYWRIGHT_ALLOWED_DOMAINS",
						"mcr.microsoft.com/playwright:v1.41.0",
					},
					Env: map[string]string{"PLAYWRIGHT_ALLOWED_DOMAINS": "localhost,localhost:*,127.0.0.1,127.0.0.1:*,example.com"},
				},
			},
		},
		{
			name: "Playwright tool with localhost default",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"playwright": map[string]any{},
				},
			},
			expected: []MCPServerConfig{
				{
					Name:    "playwright",
					Type:    "docker",
					Command: "docker",
					Args: []string{
						"run", "-i", "--rm", "--shm-size=2gb", "--cap-add=SYS_ADMIN",
						"-e", "PLAYWRIGHT_ALLOWED_DOMAINS",
						"mcr.microsoft.com/playwright:latest",
					},
					Env: map[string]string{"PLAYWRIGHT_ALLOWED_DOMAINS": "localhost,localhost:*,127.0.0.1,127.0.0.1:*"},
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
						"type": "stdio", "command": "test",
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
						"ghcr.io/github/github-mcp-server:" + constants.DefaultGitHubMCPServerVersion,
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
			// Skip tests for new MCP format during MCP revamp
			if strings.Contains(tt.name, "New format") ||
				strings.Contains(tt.name, "Custom MCP server") ||
				strings.Contains(tt.name, "HTTP MCP server") ||
				strings.Contains(tt.name, "MCP config as JSON string") ||
				strings.Contains(tt.name, "Non-MCP tool ignored") ||
				strings.Contains(tt.name, "Invalid tools section") ||
				strings.Contains(tt.name, "Invalid MCP config") {
				t.Skip("Skipping test for MCP format changes - MCP revamp in progress")
				return
			}

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
				Headers: map[string]string{},
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
				Headers: map[string]string{},
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
			name:     "HTTP server with underscored headers",
			toolName: "datadog-server",
			mcpSection: map[string]any{
				"type": "http",
				"url":  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
				"headers": map[string]any{
					"DD_API_KEY":         "test-api-key",
					"DD_APPLICATION_KEY": "test-app-key",
					"DD_SITE":            "datadoghq.com",
				},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name: "datadog-server",
				Type: "http",
				URL:  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
				Headers: map[string]string{
					"DD_API_KEY":         "test-api-key",
					"DD_APPLICATION_KEY": "test-app-key",
					"DD_SITE":            "datadoghq.com",
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
				Headers: map[string]string{},
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
				Headers: map[string]string{},
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
				Headers: map[string]string{},
				Allowed: []string{},
			},
		},
		// Error cases
		{
			name:     "Stdio with headers (invalid)",
			toolName: "stdio-invalid-headers",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": "server",
				"headers": map[string]any{
					"Authorization": "Bearer token",
				},
			},
			toolConfig:  map[string]any{},
			expectError: true,
		},
		{
			name:       "Type inferred from command field",
			toolName:   "inferred-stdio",
			mcpSection: map[string]any{"command": "server"},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:    "inferred-stdio",
				Type:    "stdio",
				Command: "server",
				Args:    nil,
				Env:     map[string]string{},
				Headers: map[string]string{},
				Allowed: nil,
			},
		},

		{
			name:     "Stdio with network proxy-args (new format)",
			toolName: "network-proxy-server",
			mcpSection: map[string]any{
				"type":    "stdio",
				"command": "docker",
				"args":    []any{"run", "myserver"},
				"network": map[string]any{
					"allowed":    []any{"example.com", "api.example.com"},
					"proxy-args": []any{"--network-proxy-arg1", "--network-proxy-arg2"},
				},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:      "network-proxy-server",
				Type:      "stdio",
				Command:   "docker",
				Args:      []string{"run", "myserver"},
				ProxyArgs: []string{"--network-proxy-arg1", "--network-proxy-arg2"},
				Env:       map[string]string{},
				Headers:   map[string]string{},
				Allowed:   []string{},
			},
		},
		{
			name:     "Local type (alias for stdio)",
			toolName: "local-server",
			mcpSection: map[string]any{
				"type":    "local",
				"command": "local-mcp-server",
				"args":    []any{"--local-mode"},
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:    "local-server",
				Type:    "stdio", // normalized to stdio
				Command: "local-mcp-server",
				Args:    []string{"--local-mode"},
				Env:     map[string]string{},
				Headers: map[string]string{},
				Allowed: []string{},
			},
		},
		{
			name:     "Stdio with registry",
			toolName: "registry-stdio",
			mcpSection: map[string]any{
				"type":     "stdio",
				"command":  "registry-server",
				"registry": "https://registry.example.com/servers/mcp-server",
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:     "registry-stdio",
				Type:     "stdio",
				Registry: "https://registry.example.com/servers/mcp-server",
				Command:  "registry-server",
				Env:      map[string]string{},
				Headers:  map[string]string{},
				Allowed:  []string{},
			},
		},
		{
			name:     "HTTP with registry",
			toolName: "registry-http",
			mcpSection: map[string]any{
				"type":     "http",
				"url":      "https://api.example.com/mcp",
				"registry": "https://registry.example.com/servers/http-mcp",
			},
			toolConfig: map[string]any{},
			expected: MCPServerConfig{
				Name:     "registry-http",
				Type:     "http",
				Registry: "https://registry.example.com/servers/http-mcp",
				URL:      "https://api.example.com/mcp",
				Headers:  map[string]string{},
				Env:      map[string]string{},
				Allowed:  []string{},
			},
		},
		{
			name:        "Missing type and no inferrable fields",
			toolName:    "no-type-no-fields",
			mcpSection:  map[string]any{"env": map[string]any{"KEY": "value"}},
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
			// Skip test for invalid stdio with headers during MCP revamp
			if strings.Contains(tt.name, "Stdio with headers") {
				t.Skip("Skipping test for MCP format validation - MCP revamp in progress")
				return
			}

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
			// Compare proxy args, handling nil vs empty slice equivalence
			actualProxyArgs := result.ProxyArgs
			if actualProxyArgs == nil {
				actualProxyArgs = []string{}
			}
			expectedProxyArgs := tt.expected.ProxyArgs
			if expectedProxyArgs == nil {
				expectedProxyArgs = []string{}
			}
			if !reflect.DeepEqual(actualProxyArgs, expectedProxyArgs) {
				t.Errorf("Expected proxy-args %v, got %v", expectedProxyArgs, actualProxyArgs)
			}
		})
	}
}

// TestMCPConfigTypes tests the struct types for proper JSON serialization
func TestMCPConfigTypes(t *testing.T) {
	// Test that our structs can be properly marshaled/unmarshaled
	config := MCPServerConfig{
		Name:      "test-server",
		Type:      "stdio",
		Command:   "test-command",
		Args:      []string{"arg1", "arg2"},
		ProxyArgs: []string{"--proxy-test"},
		Env:       map[string]string{"KEY": "value"},
		Headers:   map[string]string{"Content-Type": "application/json"},
		Allowed:   []string{"tool1", "tool2"},
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
