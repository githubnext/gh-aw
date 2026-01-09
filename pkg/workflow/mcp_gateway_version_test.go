package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPGatewayDefaultVersion tests that the default version is used when no version is specified
func TestMCPGatewayDefaultVersion(t *testing.T) {
	tests := []struct {
		name            string
		container       string
		version         string
		expectedVersion string
	}{
		{
			name:            "uses default version when not specified",
			container:       "ghcr.io/githubnext/gh-aw-mcpg",
			version:         "",
			expectedVersion: string(constants.DefaultMCPGatewayVersion),
		},
		{
			name:            "uses custom version when specified",
			container:       "ghcr.io/githubnext/gh-aw-mcpg",
			version:         "v0.0.5",
			expectedVersion: "v0.0.5",
		},
		{
			name:            "uses latest tag when specified",
			container:       "ghcr.io/githubnext/gh-aw-mcpg",
			version:         "latest",
			expectedVersion: "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow data structure
			workflowData := &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
					MCP: &MCPGatewayRuntimeConfig{
						Container: tt.container,
						Version:   tt.version,
					},
				},
				// MCP gateway is now mandatory, no feature flag needed
			}

			// Create a simple copilot engine (we just need something that implements the interface)
			engine := &CopilotEngine{}

			// Generate the MCP gateway step
			var yaml strings.Builder
			generateMCPGatewayStepInline(&yaml, engine, workflowData)

			// Verify the output contains the expected container image with version
			output := yaml.String()
			expectedImage := tt.container + ":" + tt.expectedVersion
			assert.Contains(t, output, expectedImage,
				"Generated YAML should contain container image with version: %s", expectedImage)
		})
	}
}

// TestMCPGatewayVersionConstantValue ensures the constant has the expected value
func TestMCPGatewayVersionConstantValue(t *testing.T) {
	// This test documents the expected version and will fail if it changes
	expectedVersion := "v0.0.10"
	actualVersion := string(constants.DefaultMCPGatewayVersion)

	assert.Equal(t, expectedVersion, actualVersion,
		"DefaultMCPGatewayVersion constant should be %q", expectedVersion)

	// Also verify the constant is valid (non-empty)
	require.NotEmpty(t, actualVersion, "DefaultMCPGatewayVersion should not be empty")
	require.True(t, constants.DefaultMCPGatewayVersion.IsValid(),
		"DefaultMCPGatewayVersion should be valid")
}

// TestMCPGatewayDebugEnvironmentVariable tests that DEBUG env var is passed to gateway container
func TestMCPGatewayDebugEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		shouldFind  []string
		description string
	}{
		{
			name: "DEBUG environment variable is passed through",
			envVars: map[string]string{
				"DEBUG": "*",
			},
			shouldFind: []string{
				"export DEBUG=*",
				"-e DEBUG",
			},
			description: "DEBUG env var should be exported and added to docker command",
		},
		{
			name: "Multiple environment variables including DEBUG",
			envVars: map[string]string{
				"DEBUG":       "gateway:*",
				"CUSTOM_VAR":  "value",
				"ANOTHER_VAR": "test",
			},
			shouldFind: []string{
				"export DEBUG=gateway:*",
				"export CUSTOM_VAR=value",
				"export ANOTHER_VAR=test",
				"-e DEBUG",
				"-e CUSTOM_VAR",
				"-e ANOTHER_VAR",
			},
			description: "All env vars including DEBUG should be exported and passed to container",
		},
		{
			name:    "No DEBUG variable",
			envVars: map[string]string{},
			shouldFind: []string{
				"export MCP_GATEWAY_PORT",
				"export MCP_GATEWAY_DOMAIN",
			},
			description: "Gateway should work without DEBUG env var",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create workflow data with DEBUG env var in sandbox.mcp.env
			workflowData := &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
					MCP: &MCPGatewayRuntimeConfig{
						Container: "ghcr.io/githubnext/gh-aw-mcpg",
						Version:   "v0.0.10",
						Env:       tt.envVars,
					},
				},
			}

			// Create copilot engine
			engine := &CopilotEngine{}

			// Generate the MCP gateway step
			var yaml strings.Builder
			generateMCPGatewayStepInline(&yaml, engine, workflowData)

			// Verify the output contains expected strings
			output := yaml.String()
			for _, expectedString := range tt.shouldFind {
				assert.Contains(t, output, expectedString,
					"%s: Output should contain: %s", tt.description, expectedString)
			}
		})
	}
}
