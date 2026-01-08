package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPGatewayMountsConfiguration tests that volume mounts are properly handled in MCP gateway configuration
func TestMCPGatewayMountsConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		sandboxConfig  *SandboxConfig
		expectMounts   []string
		expectError    bool
		expectInDocker bool
	}{
		{
			name: "valid mounts configuration",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: "ghcr.io/example/gateway:latest",
					Mounts: []string{
						"/host/data:/container/data:ro",
						"/host/config:/container/config:rw",
					},
				},
			},
			expectMounts:   []string{"/host/data:/container/data:ro", "/host/config:/container/config:rw"},
			expectError:    false,
			expectInDocker: true,
		},
		{
			name: "no mounts configured",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: "ghcr.io/example/gateway:latest",
					Mounts:    []string{},
				},
			},
			expectMounts:   []string{},
			expectError:    false,
			expectInDocker: false,
		},
		{
			name: "invalid mount syntax - missing mode",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: "ghcr.io/example/gateway:latest",
					Mounts: []string{
						"/host/data:/container/data",
					},
				},
			},
			expectMounts:   nil,
			expectError:    true,
			expectInDocker: false,
		},
		{
			name: "invalid mount syntax - invalid mode",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: "ghcr.io/example/gateway:latest",
					Mounts: []string{
						"/host/data:/container/data:xyz",
					},
				},
			},
			expectMounts:   nil,
			expectError:    true,
			expectInDocker: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				SandboxConfig: tt.sandboxConfig,
			}

			// Validate the configuration
			err := validateSandboxConfig(workflowData)
			if tt.expectError {
				assert.Error(t, err, "Expected validation error")
				return
			}
			require.NoError(t, err, "Unexpected validation error")

			// If mounts are expected, verify they're present
			if len(tt.expectMounts) > 0 {
				assert.ElementsMatch(t, tt.expectMounts, workflowData.SandboxConfig.MCP.Mounts,
					"Mounts should match expected values")
			}
		})
	}
}

// TestMCPGatewayDockerCommandGeneration tests that docker command includes mounts and other options
func TestMCPGatewayDockerCommandGeneration(t *testing.T) {
	tests := []struct {
		name            string
		gatewayConfig   *MCPGatewayRuntimeConfig
		expectInCommand []string
		expectNotInCmd  []string
	}{
		{
			name: "mounts included in docker command",
			gatewayConfig: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/example/gateway:latest",
				Mounts: []string{
					"/host/data:/container/data:ro",
					"/host/config:/container/config:rw",
				},
			},
			expectInCommand: []string{
				"-v /host/config:/container/config:rw",
				"-v /host/data:/container/data:ro",
			},
		},
		{
			name: "network mode included in docker command",
			gatewayConfig: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/example/gateway:latest",
				Network:   "bridge",
			},
			expectInCommand: []string{
				"--network bridge",
			},
			expectNotInCmd: []string{
				"--network host",
			},
		},
		{
			name: "port mappings included in docker command",
			gatewayConfig: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/example/gateway:latest",
				Ports: []string{
					"8080:8080",
					"9090:9090",
				},
			},
			expectInCommand: []string{
				"-p 8080:8080",
				"-p 9090:9090",
			},
		},
		{
			name: "default network mode is host",
			gatewayConfig: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/example/gateway:latest",
			},
			expectInCommand: []string{
				"--network host",
			},
		},
		{
			name: "all options combined",
			gatewayConfig: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/example/gateway:latest",
				Network:   "bridge",
				Mounts: []string{
					"/data:/data:ro",
				},
				Ports: []string{
					"8080:8080",
				},
			},
			expectInCommand: []string{
				"--network bridge",
				"-v /data:/data:ro",
				"-p 8080:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow data with MCP gateway enabled
			workflowData := &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: tt.gatewayConfig,
				},
				Features: map[string]any{
					"mcp-gateway": true,
				},
			}

			// Generate the docker command by calling the generation function
			var yamlBuilder strings.Builder
			engine := &CopilotEngine{}
			generateMCPGatewayStepInline(&yamlBuilder, engine, workflowData)

			dockerCmd := yamlBuilder.String()

			// Verify expected strings are present
			for _, expected := range tt.expectInCommand {
				assert.Contains(t, dockerCmd, expected,
					"Docker command should contain '%s'", expected)
			}

			// Verify strings that should not be present
			for _, notExpected := range tt.expectNotInCmd {
				assert.NotContains(t, dockerCmd, notExpected,
					"Docker command should not contain '%s'", notExpected)
			}
		})
	}
}

// TestMCPGatewayExtraction tests that the extraction function properly parses mounts and other fields
func TestMCPGatewayExtraction(t *testing.T) {
	tests := []struct {
		name          string
		mcpConfig     map[string]any
		expectMounts  []string
		expectNetwork string
		expectPorts   []string
	}{
		{
			name: "extract mounts",
			mcpConfig: map[string]any{
				"container": "ghcr.io/example/gateway:latest",
				"mounts": []any{
					"/host/data:/container/data:ro",
					"/host/config:/container/config:rw",
				},
			},
			expectMounts: []string{
				"/host/data:/container/data:ro",
				"/host/config:/container/config:rw",
			},
		},
		{
			name: "extract network",
			mcpConfig: map[string]any{
				"container": "ghcr.io/example/gateway:latest",
				"network":   "bridge",
			},
			expectNetwork: "bridge",
		},
		{
			name: "extract ports",
			mcpConfig: map[string]any{
				"container": "ghcr.io/example/gateway:latest",
				"ports": []any{
					"8080:8080",
					"9090:9090",
				},
			},
			expectPorts: []string{
				"8080:8080",
				"9090:9090",
			},
		},
		{
			name: "extract all fields",
			mcpConfig: map[string]any{
				"container": "ghcr.io/example/gateway:latest",
				"mounts": []any{
					"/data:/data:ro",
				},
				"network": "bridge",
				"ports": []any{
					"8080:8080",
				},
			},
			expectMounts: []string{
				"/data:/data:ro",
			},
			expectNetwork: "bridge",
			expectPorts: []string{
				"8080:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			extracted := compiler.extractMCPGatewayConfig(tt.mcpConfig)

			require.NotNil(t, extracted, "Extraction should not return nil")

			if len(tt.expectMounts) > 0 {
				assert.ElementsMatch(t, tt.expectMounts, extracted.Mounts,
					"Mounts should match expected values")
			}

			if tt.expectNetwork != "" {
				assert.Equal(t, tt.expectNetwork, extracted.Network,
					"Network should match expected value")
			}

			if len(tt.expectPorts) > 0 {
				assert.ElementsMatch(t, tt.expectPorts, extracted.Ports,
					"Ports should match expected values")
			}
		})
	}
}
