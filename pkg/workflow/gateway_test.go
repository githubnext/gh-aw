package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMCPGatewayTool(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected *MCPGatewayConfig
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "non-map input returns nil",
			input:    "not a map",
			expected: nil,
		},
		{
			name: "minimal config with container only",
			input: map[string]any{
				"container": "ghcr.io/githubnext/mcp-gateway",
			},
			expected: &MCPGatewayConfig{
				Container: "ghcr.io/githubnext/mcp-gateway",
				Port:      DefaultMCPGatewayPort,
			},
		},
		{
			name: "full config",
			input: map[string]any{
				"container":      "ghcr.io/githubnext/mcp-gateway",
				"version":        "v1.0.0",
				"port":           8888,
				"api-key":        "${{ secrets.API_KEY }}",
				"args":           []any{"-v", "--debug"},
				"entrypointArgs": []any{"--config", "/config.json"},
				"env": map[string]any{
					"DEBUG": "true",
				},
			},
			expected: &MCPGatewayConfig{
				Container:      "ghcr.io/githubnext/mcp-gateway",
				Version:        "v1.0.0",
				Port:           8888,
				APIKey:         "${{ secrets.API_KEY }}",
				Args:           []string{"-v", "--debug"},
				EntrypointArgs: []string{"--config", "/config.json"},
				Env:            map[string]string{"DEBUG": "true"},
			},
		},
		{
			name: "numeric version",
			input: map[string]any{
				"container": "ghcr.io/githubnext/mcp-gateway",
				"version":   1.0,
			},
			expected: &MCPGatewayConfig{
				Container: "ghcr.io/githubnext/mcp-gateway",
				Version:   "1",
				Port:      DefaultMCPGatewayPort,
			},
		},
		{
			name: "float port",
			input: map[string]any{
				"container": "ghcr.io/githubnext/mcp-gateway",
				"port":      8888.0,
			},
			expected: &MCPGatewayConfig{
				Container: "ghcr.io/githubnext/mcp-gateway",
				Port:      8888,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMCPGatewayTool(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Container, result.Container)
				assert.Equal(t, tt.expected.Version, result.Version)
				assert.Equal(t, tt.expected.Port, result.Port)
				assert.Equal(t, tt.expected.APIKey, result.APIKey)
				assert.Equal(t, tt.expected.Args, result.Args)
				assert.Equal(t, tt.expected.EntrypointArgs, result.EntrypointArgs)
				assert.Equal(t, tt.expected.Env, result.Env)
			}
		})
	}
}

func TestIsMCPGatewayEnabled(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected bool
	}{
		{
			name:     "nil workflow data",
			data:     nil,
			expected: false,
		},
		{
			name: "nil sandbox config",
			data: &WorkflowData{
				SandboxConfig: nil,
			},
			expected: false,
		},
		{
			name: "no mcp in sandbox",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{Type: SandboxTypeAWF},
				},
			},
			expected: false,
		},
		{
			name: "sandbox.mcp without feature flag",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayConfig{
						Container: "test",
					},
				},
			},
			expected: false,
		},
		{
			name: "sandbox.mcp with feature flag",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayConfig{
						Container: "test",
					},
				},
				Features: map[string]bool{
					"mcp-gateway": true,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMCPGatewayEnabled(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMCPGatewayConfig(t *testing.T) {
	tests := []struct {
		name      string
		data      *WorkflowData
		hasConfig bool
	}{
		{
			name:      "nil workflow data",
			data:      nil,
			hasConfig: false,
		},
		{
			name: "no mcp in sandbox",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{Type: SandboxTypeAWF},
				},
			},
			hasConfig: false,
		},
		{
			name: "valid sandbox.mcp config",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayConfig{
						Container: "test-image",
						Port:      9090,
					},
				},
			},
			hasConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMCPGatewayConfig(tt.data)
			if tt.hasConfig {
				require.NotNil(t, result)
				assert.Equal(t, "test-image", result.Container)
				assert.Equal(t, 9090, result.Port)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestGenerateMCPGatewaySteps(t *testing.T) {
	tests := []struct {
		name        string
		data        *WorkflowData
		mcpServers  map[string]any
		expectSteps int
	}{
		{
			name:        "gateway disabled returns no steps",
			data:        &WorkflowData{},
			mcpServers:  map[string]any{},
			expectSteps: 0,
		},
		{
			name: "gateway enabled returns two steps",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayConfig{
						Container: "test-gateway",
						Port:      8080,
					},
				},
				Features: map[string]bool{
					"mcp-gateway": true,
				},
			},
			mcpServers: map[string]any{
				"github": map[string]any{},
			},
			expectSteps: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := generateMCPGatewaySteps(tt.data, tt.mcpServers)
			assert.Len(t, steps, tt.expectSteps)
		})
	}
}

func TestGenerateMCPGatewayStartStep(t *testing.T) {
	config := &MCPGatewayConfig{
		Container: "ghcr.io/githubnext/mcp-gateway",
		Port:      8080,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}

	step := generateMCPGatewayStartStep(config, mcpServers)
	stepStr := strings.Join(step, "\n")

	assert.Contains(t, stepStr, "Start MCP Gateway")
	assert.Contains(t, stepStr, "docker")
	assert.Contains(t, stepStr, "ghcr.io/githubnext/mcp-gateway")
	assert.Contains(t, stepStr, "8080:8080")
}

func TestGenerateMCPGatewayHealthCheckStep(t *testing.T) {
	config := &MCPGatewayConfig{
		Port: 8080,
	}

	step := generateMCPGatewayHealthCheckStep(config)
	stepStr := strings.Join(step, "\n")

	assert.Contains(t, stepStr, "Verify MCP Gateway Health")
	assert.Contains(t, stepStr, "http://localhost:8080")
	assert.Contains(t, stepStr, "/health")
	assert.Contains(t, stepStr, "max_retries")
}

func TestGetMCPGatewayURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *MCPGatewayConfig
		expected string
	}{
		{
			name:     "default port",
			config:   &MCPGatewayConfig{},
			expected: "http://localhost:8080",
		},
		{
			name: "custom port",
			config: &MCPGatewayConfig{
				Port: 9090,
			},
			expected: "http://localhost:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMCPGatewayURL(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformMCPConfigForGateway(t *testing.T) {
	tests := []struct {
		name       string
		mcpServers map[string]any
		config     *MCPGatewayConfig
		expected   map[string]any
	}{
		{
			name: "nil config returns original",
			mcpServers: map[string]any{
				"github": map[string]any{"type": "local"},
			},
			config: nil,
			expected: map[string]any{
				"github": map[string]any{"type": "local"},
			},
		},
		{
			name: "transforms servers to gateway URLs",
			mcpServers: map[string]any{
				"github":     map[string]any{},
				"playwright": map[string]any{},
			},
			config: &MCPGatewayConfig{
				Port: 8080,
			},
			expected: map[string]any{
				"github": map[string]any{
					"type": "http",
					"url":  "http://localhost:8080/mcp/github",
				},
				"playwright": map[string]any{
					"type": "http",
					"url":  "http://localhost:8080/mcp/playwright",
				},
			},
		},
		{
			name: "adds auth header when api-key present",
			mcpServers: map[string]any{
				"github": map[string]any{},
			},
			config: &MCPGatewayConfig{
				Port:   8080,
				APIKey: "secret",
			},
			expected: map[string]any{
				"github": map[string]any{
					"type": "http",
					"url":  "http://localhost:8080/mcp/github",
					"headers": map[string]any{
						"Authorization": "Bearer ${MCP_GATEWAY_API_KEY}",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformMCPConfigForGateway(tt.mcpServers, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSandboxConfigWithMCP(t *testing.T) {
	sandboxConfig := &SandboxConfig{
		Agent: &AgentSandboxConfig{
			Type: SandboxTypeAWF,
		},
		MCP: &MCPGatewayConfig{
			Container: "test-image",
			Port:      9000,
		},
	}

	require.NotNil(t, sandboxConfig.MCP)
	assert.Equal(t, "test-image", sandboxConfig.MCP.Container)
	assert.Equal(t, 9000, sandboxConfig.MCP.Port)

	require.NotNil(t, sandboxConfig.Agent)
	assert.Equal(t, SandboxTypeAWF, sandboxConfig.Agent.Type)
}
