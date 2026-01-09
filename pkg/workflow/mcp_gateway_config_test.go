package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayMCPServerConfig_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   *GatewayMCPServerConfig
		expected string
	}{
		{
			name: "stdio server with container",
			config: &GatewayMCPServerConfig{
				Container: "ghcr.io/example/server:latest",
				Type:      "stdio",
				Env: map[string]string{
					"API_KEY": "test-key",
				},
			},
			expected: `{"container":"ghcr.io/example/server:latest","env":{"API_KEY":"test-key"},"type":"stdio"}`,
		},
		{
			name: "http server",
			config: &GatewayMCPServerConfig{
				Type: "http",
				URL:  "https://api.example.com/mcp",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			expected: `{"type":"http","url":"https://api.example.com/mcp","headers":{"Authorization":"Bearer token"}}`,
		},
		{
			name: "stdio server with entrypoint and args",
			config: &GatewayMCPServerConfig{
				Container:      "ghcr.io/example/server:latest",
				Entrypoint:     "/custom/entrypoint.sh",
				EntrypointArgs: []string{"--config", "/app/config.json"},
				Type:           "stdio",
			},
			expected: `{"container":"ghcr.io/example/server:latest","entrypoint":"/custom/entrypoint.sh","entrypointArgs":["--config","/app/config.json"],"type":"stdio"}`,
		},
		{
			name: "stdio server with mounts",
			config: &GatewayMCPServerConfig{
				Container: "ghcr.io/example/server:latest",
				Mounts: []string{
					"/host/data:/data:ro",
					"/host/config:/config:rw",
				},
				Type: "stdio",
			},
			expected: `{"container":"ghcr.io/example/server:latest","mounts":["/host/data:/data:ro","/host/config:/config:rw"],"type":"stdio"}`,
		},
		{
			name: "copilot server with tools field",
			config: &GatewayMCPServerConfig{
				Container: "ghcr.io/github/github-mcp-server:latest",
				Type:      "stdio",
				Tools:     []string{"*"},
				Env: map[string]string{
					"GITHUB_PERSONAL_ACCESS_TOKEN": "token",
				},
			},
			expected: `{"container":"ghcr.io/github/github-mcp-server:latest","env":{"GITHUB_PERSONAL_ACCESS_TOKEN":"token"},"type":"stdio","tools":["*"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			require.NoError(t, err, "Failed to marshal config")

			// Unmarshal both to compare as objects (handles ordering differences)
			var actual, expected map[string]any
			require.NoError(t, json.Unmarshal(data, &actual), "Failed to unmarshal actual")
			require.NoError(t, json.Unmarshal([]byte(tt.expected), &expected), "Failed to unmarshal expected")

			assert.Equal(t, expected, actual, "Marshaled JSON should match expected")
		})
	}
}

func TestGatewayMCPServerConfig_JSONUnmarshaling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *GatewayMCPServerConfig
	}{
		{
			name:  "stdio server",
			input: `{"container":"ghcr.io/example/server:latest","type":"stdio"}`,
			expected: &GatewayMCPServerConfig{
				Container: "ghcr.io/example/server:latest",
				Type:      "stdio",
			},
		},
		{
			name:  "http server",
			input: `{"type":"http","url":"https://api.example.com/mcp"}`,
			expected: &GatewayMCPServerConfig{
				Type: "http",
				URL:  "https://api.example.com/mcp",
			},
		},
		{
			name:  "server with all fields",
			input: `{"container":"ghcr.io/example/server:latest","entrypoint":"/bin/sh","entrypointArgs":["--verbose"],"mounts":["/data:/data:ro"],"env":{"KEY":"value"},"type":"stdio"}`,
			expected: &GatewayMCPServerConfig{
				Container:      "ghcr.io/example/server:latest",
				Entrypoint:     "/bin/sh",
				EntrypointArgs: []string{"--verbose"},
				Mounts:         []string{"/data:/data:ro"},
				Env:            map[string]string{"KEY": "value"},
				Type:           "stdio",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual GatewayMCPServerConfig
			err := json.Unmarshal([]byte(tt.input), &actual)
			require.NoError(t, err, "Failed to unmarshal config")
			assert.Equal(t, tt.expected, &actual, "Unmarshaled config should match expected")
		})
	}
}

func TestGatewayConfig_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   *GatewayConfig
		expected string
	}{
		{
			name: "full gateway config",
			config: &GatewayConfig{
				Port:           8080,
				APIKey:         "test-key",
				Domain:         "localhost",
				StartupTimeout: 30,
				ToolTimeout:    60,
			},
			expected: `{"port":8080,"apiKey":"test-key","domain":"localhost","startupTimeout":30,"toolTimeout":60}`,
		},
		{
			name: "minimal gateway config",
			config: &GatewayConfig{
				Port: 8080,
			},
			expected: `{"port":8080}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			require.NoError(t, err, "Failed to marshal config")

			var actual, expected map[string]any
			require.NoError(t, json.Unmarshal(data, &actual), "Failed to unmarshal actual")
			require.NoError(t, json.Unmarshal([]byte(tt.expected), &expected), "Failed to unmarshal expected")

			assert.Equal(t, expected, actual, "Marshaled JSON should match expected")
		})
	}
}

func TestGatewayMCPRootConfig_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		config   *GatewayMCPRootConfig
		validate func(t *testing.T, data []byte)
	}{
		{
			name: "root config with servers and gateway",
			config: &GatewayMCPRootConfig{
				MCPServers: map[string]*GatewayMCPServerConfig{
					"github": {
						Container: "ghcr.io/github/github-mcp-server:latest",
						Type:      "stdio",
						Env: map[string]string{
							"GITHUB_PERSONAL_ACCESS_TOKEN": "token",
						},
					},
					"remote-server": {
						Type: "http",
						URL:  "https://api.example.com/mcp",
					},
				},
				Gateway: &GatewayConfig{
					Port:   8080,
					APIKey: "gateway-key",
					Domain: "localhost",
				},
			},
			validate: func(t *testing.T, data []byte) {
				var result map[string]any
				require.NoError(t, json.Unmarshal(data, &result), "Failed to unmarshal result")

				// Check mcpServers exists
				assert.Contains(t, result, "mcpServers", "Should have mcpServers field")
				servers := result["mcpServers"].(map[string]any)
				assert.Len(t, servers, 2, "Should have 2 servers")
				assert.Contains(t, servers, "github", "Should have github server")
				assert.Contains(t, servers, "remote-server", "Should have remote-server")

				// Check gateway exists
				assert.Contains(t, result, "gateway", "Should have gateway field")
				gateway := result["gateway"].(map[string]any)
				assert.Equal(t, float64(8080), gateway["port"], "Gateway port should match")
				assert.Equal(t, "gateway-key", gateway["apiKey"], "Gateway API key should match")
			},
		},
		{
			name: "root config without gateway",
			config: &GatewayMCPRootConfig{
				MCPServers: map[string]*GatewayMCPServerConfig{
					"test": {
						Container: "test:latest",
						Type:      "stdio",
					},
				},
			},
			validate: func(t *testing.T, data []byte) {
				var result map[string]any
				require.NoError(t, json.Unmarshal(data, &result), "Failed to unmarshal result")

				assert.Contains(t, result, "mcpServers", "Should have mcpServers field")
				assert.NotContains(t, result, "gateway", "Should not have gateway field when nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			require.NoError(t, err, "Failed to marshal config")
			tt.validate(t, data)
		})
	}
}

func TestGatewayMCPRootConfig_Roundtrip(t *testing.T) {
	// Test that we can marshal and unmarshal without data loss
	original := &GatewayMCPRootConfig{
		MCPServers: map[string]*GatewayMCPServerConfig{
			"github": {
				Container: "ghcr.io/github/github-mcp-server:v0.27.0",
				Type:      "stdio",
				Env: map[string]string{
					"GITHUB_PERSONAL_ACCESS_TOKEN": "test-token",
					"GITHUB_READ_ONLY":             "1",
				},
				Tools: []string{"*"},
			},
			"playwright": {
				Container:      "mcr.microsoft.com/playwright/mcp",
				EntrypointArgs: []string{"--output-dir", "/tmp/logs"},
				Mounts:         []string{"/tmp/logs:/tmp/logs:rw"},
				Type:           "stdio",
			},
		},
		Gateway: &GatewayConfig{
			Port:   8080,
			APIKey: "test-key",
			Domain: "localhost",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err, "Failed to marshal config")

	// Unmarshal back
	var restored GatewayMCPRootConfig
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err, "Failed to unmarshal config")

	// Compare
	assert.Equal(t, len(original.MCPServers), len(restored.MCPServers), "Server count should match")
	assert.NotNil(t, restored.Gateway, "Gateway should be restored")
	assert.Equal(t, original.Gateway.Port, restored.Gateway.Port, "Gateway port should match")
	assert.Equal(t, original.Gateway.APIKey, restored.Gateway.APIKey, "Gateway API key should match")
}

func TestBuildGatewayMCPServerConfig(t *testing.T) {
	tests := []struct {
		name     string
		options  GitHubMCPDockerOptions
		validate func(t *testing.T, config *GatewayMCPServerConfig)
	}{
		{
			name: "basic stdio server",
			options: GitHubMCPDockerOptions{
				DockerImageVersion: "v0.27.0",
				Toolsets:           "default",
				IncludeTypeField:   false,
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, "ghcr.io/github/github-mcp-server:v0.27.0", config.Container)
				assert.Equal(t, "stdio", config.Type)
				assert.Contains(t, config.Env, "GITHUB_PERSONAL_ACCESS_TOKEN")
				assert.Contains(t, config.Env, "GITHUB_TOOLSETS")
				assert.Equal(t, "default", config.Env["GITHUB_TOOLSETS"])
			},
		},
		{
			name: "copilot server with tools field",
			options: GitHubMCPDockerOptions{
				DockerImageVersion: "v0.27.0",
				Toolsets:           "repos,issues",
				IncludeTypeField:   true,
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, []string{"*"}, config.Tools)
				assert.Contains(t, config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"], "\\${")
			},
		},
		{
			name: "read-only server",
			options: GitHubMCPDockerOptions{
				DockerImageVersion: "v0.27.0",
				ReadOnly:           true,
				Toolsets:           "default",
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, "1", config.Env["GITHUB_READ_ONLY"])
			},
		},
		{
			name: "lockdown mode server",
			options: GitHubMCPDockerOptions{
				DockerImageVersion: "v0.27.0",
				Lockdown:           true,
				Toolsets:           "default",
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, "1", config.Env["GITHUB_LOCKDOWN_MODE"])
			},
		},
		{
			name: "lockdown from step",
			options: GitHubMCPDockerOptions{
				DockerImageVersion: "v0.27.0",
				LockdownFromStep:   true,
				Toolsets:           "default",
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, "$GITHUB_MCP_LOCKDOWN", config.Env["GITHUB_LOCKDOWN_MODE"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BuildGatewayMCPServerConfig(tt.options)
			require.NotNil(t, config)
			tt.validate(t, config)
		})
	}
}

func TestBuildGatewayMCPServerConfigFromRemote(t *testing.T) {
	tests := []struct {
		name     string
		options  GitHubMCPRemoteOptions
		validate func(t *testing.T, config *GatewayMCPServerConfig)
	}{
		{
			name: "basic http server",
			options: GitHubMCPRemoteOptions{
				AuthorizationValue: "Bearer test-token",
				Toolsets:           "default",
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, "http", config.Type)
				assert.Equal(t, "https://api.githubcopilot.com/mcp/", config.URL)
				assert.Equal(t, "Bearer test-token", config.Headers["Authorization"])
			},
		},
		{
			name: "read-only http server",
			options: GitHubMCPRemoteOptions{
				AuthorizationValue: "Bearer test-token",
				ReadOnly:           true,
				Toolsets:           "default",
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, "true", config.Headers["X-MCP-Readonly"])
			},
		},
		{
			name: "copilot http server with tools",
			options: GitHubMCPRemoteOptions{
				AuthorizationValue: "Bearer test-token",
				IncludeToolsField:  true,
				Toolsets:           "default",
			},
			validate: func(t *testing.T, config *GatewayMCPServerConfig) {
				assert.Equal(t, []string{"*"}, config.Tools)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BuildGatewayMCPServerConfigFromRemote(tt.options)
			require.NotNil(t, config)
			tt.validate(t, config)
		})
	}
}

func TestRenderGatewayMCPConfigJSON(t *testing.T) {
	config := &GatewayMCPRootConfig{
		MCPServers: map[string]*GatewayMCPServerConfig{
			"github": {
				Container: "ghcr.io/github/github-mcp-server:v0.27.0",
				Type:      "stdio",
				Env: map[string]string{
					"GITHUB_PERSONAL_ACCESS_TOKEN": "test-token",
				},
			},
		},
		Gateway: &GatewayConfig{
			Port:   8080,
			APIKey: "test-key",
			Domain: "localhost",
		},
	}

	result, err := RenderGatewayMCPConfigJSON(config)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify it's valid JSON by unmarshaling
	var decoded GatewayMCPRootConfig
	err = json.Unmarshal([]byte(result), &decoded)
	require.NoError(t, err)

	// Verify structure
	assert.Len(t, decoded.MCPServers, 1)
	assert.NotNil(t, decoded.Gateway)
	assert.Equal(t, 8080, decoded.Gateway.Port)
}
