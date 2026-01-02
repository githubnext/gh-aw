package workflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMCPGatewayTool(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected *MCPGatewayRuntimeConfig
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
			name: "minimal config with port only",
			input: map[string]any{
				"port": 8080,
			},
			expected: &MCPGatewayRuntimeConfig{
				Port: 8080,
			},
		},
		{
			name: "full config",
			input: map[string]any{
				"port":           8888,
				"api-key":        "${{ secrets.API_KEY }}",
				"args":           []any{"-v", "--debug"},
				"entrypointArgs": []any{"--config", "/config.json"},
				"env": map[string]any{
					"DEBUG": "true",
				},
			},
			expected: &MCPGatewayRuntimeConfig{
				Port:           8888,
				APIKey:         "${{ secrets.API_KEY }}",
				Args:           []string{"-v", "--debug"},
				EntrypointArgs: []string{"--config", "/config.json"},
				Env:            map[string]string{"DEBUG": "true"},
			},
		},
		{
			name:  "empty config",
			input: map[string]any{},
			expected: &MCPGatewayRuntimeConfig{
				Port: DefaultMCPGatewayPort,
			},
		},
		{
			name: "float port",
			input: map[string]any{
				"port": 8888.0,
			},
			expected: &MCPGatewayRuntimeConfig{
				Port: 8888,
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
			name: "sandbox.mcp configured",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayRuntimeConfig{
						Port: 8080,
					},
				},
			},
			expected: true,
		},
		{
			name: "sandbox.mcp with empty config",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayRuntimeConfig{},
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
					MCP: &MCPGatewayRuntimeConfig{
						Port: 9090,
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
				assert.Equal(t, 9090, result.Port)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestGenerateMCPGatewaySteps(t *testing.T) {
	tests := []struct {
		name             string
		data             *WorkflowData
		mcpServers       map[string]any
		gatewayedServers []string
		expectSteps      int
	}{
		{
			name:             "gateway disabled returns no steps",
			data:             &WorkflowData{},
			mcpServers:       map[string]any{},
			gatewayedServers: []string{},
			expectSteps:      0,
		},
		{
			name: "gateway enabled returns two steps",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayRuntimeConfig{
						Port: 8080,
					},
				},
				Features: map[string]any{
					"mcp-gateway": true,
				},
			},
			mcpServers: map[string]any{
				"github": map[string]any{},
			},
			gatewayedServers: []string{"github"},
			expectSteps:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := generateMCPGatewaySteps(tt.data, tt.mcpServers, map[string]string{})
			assert.Len(t, steps, tt.expectSteps)
		})
	}
}

func TestGenerateMCPGatewayStartStep(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}
	mcpEnvVars := map[string]string{}

	step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
	stepStr := strings.Join(step, "\n")

	assert.Contains(t, stepStr, "Start MCP Gateway")
	assert.Contains(t, stepStr, "awmg")
	assert.Contains(t, stepStr, "--config")
	assert.Contains(t, stepStr, "/home/runner/.copilot/mcp-config.json")
	assert.Contains(t, stepStr, "--port 8080")
	assert.Contains(t, stepStr, MCPGatewayLogsFolder)
}

func TestGenerateMCPGatewayStartStep_WithEnvVars(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}
	mcpEnvVars := map[string]string{
		"GITHUB_MCP_SERVER_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		"GH_AW_SAFE_OUTPUTS":      "${{ env.GH_AW_SAFE_OUTPUTS }}",
		"API_KEY":                 "${{ secrets.API_KEY }}",
	}

	step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
	stepStr := strings.Join(step, "\n")

	// Should have env block
	assert.Contains(t, stepStr, "env:")

	// Should contain all environment variables in sorted order
	assert.Contains(t, stepStr, "API_KEY: ${{ secrets.API_KEY }}")
	assert.Contains(t, stepStr, "GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}")
	assert.Contains(t, stepStr, "GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GITHUB_TOKEN }}")

	// Verify env block comes before run block
	envIdx := strings.Index(stepStr, "env:")
	runIdx := strings.Index(stepStr, "run:")
	assert.Less(t, envIdx, runIdx, "env block should come before run block")
}

func TestGenerateMCPGatewayStartStep_WithoutEnvVars(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}
	mcpEnvVars := map[string]string{} // Empty env vars

	step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
	stepStr := strings.Join(step, "\n")

	// Should NOT have env block when no env vars provided
	assert.NotContains(t, stepStr, "env:")

	// Should still have the step name and run block
	assert.Contains(t, stepStr, "Start MCP Gateway")
	assert.Contains(t, stepStr, "run:")
}

func TestGenerateMCPGatewayHealthCheckStep(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}

	step := generateMCPGatewayHealthCheckStep(config)
	stepStr := strings.Join(step, "\n")

	assert.Contains(t, stepStr, "Verify MCP Gateway Health")
	assert.Contains(t, stepStr, "bash /tmp/gh-aw/actions/verify_mcp_gateway_health.sh")
	assert.Contains(t, stepStr, "http://localhost:8080")
	assert.Contains(t, stepStr, "/home/runner/.copilot/mcp-config.json")
	assert.Contains(t, stepStr, MCPGatewayLogsFolder)
}

func TestGenerateMCPGatewayHealthCheckStep_ValidatesGatewayedServers(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}

	// Test with multiple gatewayed servers
	gatewayedServers := []string{"github", "playwright", "serena"}
	step := generateMCPGatewayHealthCheckStep(config)
	stepStr := strings.Join(step, "\n")

	// Should include gateway validation section
	assert.Contains(t, stepStr, "Validating gatewayed servers...")

	// Should validate each gatewayed server using the shell script
	for _, serverName := range gatewayedServers {
		// Verify the script is called with correct arguments
		assert.Contains(t, stepStr, fmt.Sprintf("# Validate %s server", serverName))
		assert.Contains(t, stepStr, fmt.Sprintf("/tmp/gh-aw/actions/validate_gatewayed_server.sh \"%s\"", serverName))
		assert.Contains(t, stepStr, "http://localhost:8080")
	}

	// Should have completion message
	assert.Contains(t, stepStr, "All gatewayed servers validated successfully")
}

func TestGenerateMCPGatewayHealthCheckStep_NoGatewayedServers(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}

	// Test with no gatewayed servers (only internal servers)
	step := generateMCPGatewayHealthCheckStep(config)
	stepStr := strings.Join(step, "\n")

	// Should NOT include gateway validation section
	assert.NotContains(t, stepStr, "Validating gatewayed servers...")
	assert.NotContains(t, stepStr, "All gatewayed servers validated successfully")

	// Should still have basic health check
	assert.Contains(t, stepStr, "Verify MCP Gateway Health")
	assert.Contains(t, stepStr, "Waiting for MCP Gateway to be ready...")
}

func TestGenerateMCPGatewayHealthCheckStep_SkipsInternalServers(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}

	// Test with internal servers that should be skipped
	step := generateMCPGatewayHealthCheckStep(config)
	stepStr := strings.Join(step, "\n")

	// Should NOT validate safe-inputs or safe-outputs as gatewayed
	assert.NotContains(t, stepStr, "# Validate safe-inputs server")
	assert.NotContains(t, stepStr, "# Validate safe-outputs server")

	// Should validate github as gatewayed
	assert.Contains(t, stepStr, "# Validate github server")
	assert.Contains(t, stepStr, "/tmp/gh-aw/actions/validate_gatewayed_server.sh \"github\"")
}

func TestGetMCPGatewayURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *MCPGatewayRuntimeConfig
		expected string
	}{
		{
			name:     "default port",
			config:   &MCPGatewayRuntimeConfig{},
			expected: "http://localhost:8080",
		},
		{
			name: "custom port",
			config: &MCPGatewayRuntimeConfig{
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
		config     *MCPGatewayRuntimeConfig
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
			config: &MCPGatewayRuntimeConfig{
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
			config: &MCPGatewayRuntimeConfig{
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
		MCP: &MCPGatewayRuntimeConfig{
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

func TestGenerateContainerStartCommands(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Container:      "ghcr.io/githubnext/gh-aw-mcpg:latest",
		Args:           []string{"--rm", "-i", "-v", "/var/run/docker.sock:/var/run/docker.sock", "-p", "8000:8000", "--entrypoint", "/app/flowguard-go"},
		EntrypointArgs: []string{"--routed", "--listen", "0.0.0.0:8000", "--config-stdin"},
		Port:           8000,
		Env: map[string]string{
			"DOCKER_API_VERSION": "1.44",
		},
	}

	mcpConfigPath := "/home/runner/.copilot/mcp-config.json"
	lines := generateContainerStartCommands(config, mcpConfigPath, 8000)
	output := strings.Join(lines, "\n")

	// Verify container mode is indicated
	assert.Contains(t, output, "Start MCP gateway using Docker container")
	assert.Contains(t, output, "ghcr.io/githubnext/gh-aw-mcpg:latest")

	// Verify docker run command is constructed correctly
	assert.Contains(t, output, "docker run")
	assert.Contains(t, output, "--rm")
	assert.Contains(t, output, "-i")
	assert.Contains(t, output, "-v")
	assert.Contains(t, output, "/var/run/docker.sock:/var/run/docker.sock")
	assert.Contains(t, output, "-p")
	assert.Contains(t, output, "8000:8000")
	assert.Contains(t, output, "--entrypoint")
	assert.Contains(t, output, "/app/flowguard-go")

	// Verify environment variables are set
	assert.Contains(t, output, "-e DOCKER_API_VERSION=\"1.44\"")

	// Verify entrypoint args
	assert.Contains(t, output, "--routed")
	assert.Contains(t, output, "--listen")
	assert.Contains(t, output, "0.0.0.0:8000")
	assert.Contains(t, output, "--config-stdin")

	// Verify config is piped via stdin
	assert.Contains(t, output, "cat /home/runner/.copilot/mcp-config.json |")
	assert.Contains(t, output, MCPGatewayLogsFolder)
}

func TestGenerateCommandStartCommands(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Command: "/usr/local/bin/mcp-gateway",
		Args:    []string{"--port", "8080", "--verbose"},
		Port:    8080,
		Env: map[string]string{
			"LOG_LEVEL": "debug",
			"API_KEY":   "test-key",
		},
	}

	mcpConfigPath := "/home/runner/.copilot/mcp-config.json"
	lines := generateCommandStartCommands(config, mcpConfigPath, 8080)
	output := strings.Join(lines, "\n")

	// Verify command mode is indicated
	assert.Contains(t, output, "Start MCP gateway using custom command")
	assert.Contains(t, output, "/usr/local/bin/mcp-gateway")

	// Verify command with args
	assert.Contains(t, output, "/usr/local/bin/mcp-gateway --port 8080 --verbose")

	// Verify environment variables are exported
	assert.Contains(t, output, "export LOG_LEVEL=\"debug\"")
	assert.Contains(t, output, "export API_KEY=\"test-key\"")

	// Verify config is piped via stdin
	assert.Contains(t, output, "cat /home/runner/.copilot/mcp-config.json |")
	assert.Contains(t, output, MCPGatewayLogsFolder)
}

func TestGenerateDefaultAWMGCommands(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}

	mcpConfigPath := "/home/runner/.copilot/mcp-config.json"
	lines := generateDefaultAWMGCommands(config, mcpConfigPath, 8080)
	output := strings.Join(lines, "\n")

	// Verify awmg binary handling
	assert.Contains(t, output, "awmg")
	assert.Contains(t, output, "AWMG_CMD")

	// Verify config file and port
	assert.Contains(t, output, "--config /home/runner/.copilot/mcp-config.json")
	assert.Contains(t, output, "--port 8080")
	assert.Contains(t, output, MCPGatewayLogsFolder)
}

func TestGenerateMCPGatewayStartStep_ContainerMode(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Container:      "ghcr.io/githubnext/gh-aw-mcpg:latest",
		Args:           []string{"--rm", "-i"},
		EntrypointArgs: []string{"--config-stdin"},
		Port:           8000,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}
	mcpEnvVars := map[string]string{}

	step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
	stepStr := strings.Join(step, "\n")

	// Should use container mode
	assert.Contains(t, stepStr, "Start MCP Gateway")
	assert.Contains(t, stepStr, "docker run")
	assert.Contains(t, stepStr, "ghcr.io/githubnext/gh-aw-mcpg:latest")
	assert.NotContains(t, stepStr, "awmg") // Should not use awmg
}

func TestGenerateMCPGatewayStartStep_CommandMode(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Command: "/usr/local/bin/custom-gateway",
		Args:    []string{"--debug"},
		Port:    9000,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}
	mcpEnvVars := map[string]string{}

	step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
	stepStr := strings.Join(step, "\n")

	// Should use command mode
	assert.Contains(t, stepStr, "Start MCP Gateway")
	assert.Contains(t, stepStr, "/usr/local/bin/custom-gateway --debug")
	assert.NotContains(t, stepStr, "docker run") // Should not use docker
	assert.NotContains(t, stepStr, "awmg")       // Should not use awmg
}

func TestGenerateMCPGatewayStartStep_DefaultMode(t *testing.T) {
	config := &MCPGatewayRuntimeConfig{
		Port: 8080,
	}
	mcpServers := map[string]any{
		"github": map[string]any{},
	}
	mcpEnvVars := map[string]string{}

	step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
	stepStr := strings.Join(step, "\n")

	// Should use default awmg mode
	assert.Contains(t, stepStr, "Start MCP Gateway")
	assert.Contains(t, stepStr, "awmg")
	assert.NotContains(t, stepStr, "docker run")                    // Should not use docker
	assert.NotContains(t, stepStr, "/usr/local/bin/custom-gateway") // Should not use custom command
}

func TestValidateAndNormalizePort(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		expected    int
		expectError bool
	}{
		{
			name:        "port 0 uses default",
			port:        0,
			expected:    DefaultMCPGatewayPort,
			expectError: false,
		},
		{
			name:        "valid port 1",
			port:        1,
			expected:    1,
			expectError: false,
		},
		{
			name:        "valid port 8080",
			port:        8080,
			expected:    8080,
			expectError: false,
		},
		{
			name:        "valid port 65535",
			port:        65535,
			expected:    65535,
			expectError: false,
		},
		{
			name:        "negative port returns error",
			port:        -1,
			expected:    0,
			expectError: true,
		},
		{
			name:        "port above 65535 returns error",
			port:        65536,
			expected:    0,
			expectError: true,
		},
		{
			name:        "large negative port returns error",
			port:        -9999,
			expected:    0,
			expectError: true,
		},
		{
			name:        "port well above max returns error",
			port:        100000,
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateAndNormalizePort(tt.port)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "port must be between 1 and 65535")
				assert.Contains(t, err.Error(), fmt.Sprintf("%d", tt.port))
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGenerateMCPGatewayStartStepWithInvalidPort(t *testing.T) {
	tests := []struct {
		name         string
		port         int
		expectsInLog bool
	}{
		{
			name:         "negative port falls back to default",
			port:         -1,
			expectsInLog: true,
		},
		{
			name:         "port above 65535 falls back to default",
			port:         70000,
			expectsInLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MCPGatewayRuntimeConfig{
				Port: tt.port,
			}
			mcpServers := map[string]any{
				"github": map[string]any{},
			}
			mcpEnvVars := map[string]string{}

			step := generateMCPGatewayStartStep(config, mcpServers, mcpEnvVars)
			stepStr := strings.Join(step, "\n")

			// Should still generate valid step with default port
			assert.Contains(t, stepStr, "Start MCP Gateway")
			assert.Contains(t, stepStr, fmt.Sprintf("--port %d", DefaultMCPGatewayPort))
		})
	}
}

func TestGenerateMCPGatewayHealthCheckStepWithInvalidPort(t *testing.T) {
	tests := []struct {
		name         string
		port         int
		expectsInLog bool
	}{
		{
			name:         "negative port falls back to default",
			port:         -1,
			expectsInLog: true,
		},
		{
			name:         "port above 65535 falls back to default",
			port:         70000,
			expectsInLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MCPGatewayRuntimeConfig{
				Port: tt.port,
			}

			step := generateMCPGatewayHealthCheckStep(config)
			stepStr := strings.Join(step, "\n")

			// Should still generate valid step with default port
			assert.Contains(t, stepStr, "Verify MCP Gateway Health")
			assert.Contains(t, stepStr, fmt.Sprintf("http://localhost:%d", DefaultMCPGatewayPort))
		})
	}
}

func TestGetMCPGatewayURLWithInvalidPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected string
	}{
		{
			name:     "negative port falls back to default",
			port:     -1,
			expected: fmt.Sprintf("http://localhost:%d", DefaultMCPGatewayPort),
		},
		{
			name:     "port above 65535 falls back to default",
			port:     70000,
			expected: fmt.Sprintf("http://localhost:%d", DefaultMCPGatewayPort),
		},
		{
			name:     "port 0 uses default",
			port:     0,
			expected: fmt.Sprintf("http://localhost:%d", DefaultMCPGatewayPort),
		},
		{
			name:     "valid port 9090",
			port:     9090,
			expected: "http://localhost:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MCPGatewayRuntimeConfig{
				Port: tt.port,
			}

			result := getMCPGatewayURL(config)
			assert.Equal(t, tt.expected, result)
		})
	}
}
