package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayDomainConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		expectedDomain string
	}{
		{
			name: "firewall enabled - should use host.docker.internal",
			workflowData: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: false,
					},
					MCP: &MCPGatewayRuntimeConfig{
						Port: 8080,
					},
				},
			},
			expectedDomain: "host.docker.internal",
		},
		{
			name: "firewall disabled - should use localhost",
			workflowData: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: true,
					},
					MCP: &MCPGatewayRuntimeConfig{
						Port: 8080,
					},
				},
			},
			expectedDomain: "localhost",
		},
		{
			name: "no agent config - should use host.docker.internal (default enabled)",
			workflowData: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayRuntimeConfig{
						Port: 8080,
					},
				},
			},
			expectedDomain: "host.docker.internal",
		},
		{
			name: "explicit domain overrides auto-detection",
			workflowData: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: false,
					},
					MCP: &MCPGatewayRuntimeConfig{
						Port:   8080,
						Domain: "custom.domain.com",
					},
				},
			},
			expectedDomain: "custom.domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copilot engine
			engine := &CopilotEngine{}

			// Prepare test data
			tools := map[string]any{}
			mcpTools := []string{}

			// Render MCP config
			var yaml strings.Builder
			engine.RenderMCPConfig(&yaml, tools, mcpTools, tt.workflowData)

			// Check that the domain is in the rendered config
			output := yaml.String()
			if tt.expectedDomain != "" {
				assert.Contains(t, output, "\"domain\": \""+tt.expectedDomain+"\"",
					"Expected domain %s to be in rendered config", tt.expectedDomain)
			}
		})
	}
}

func TestGatewayDomainInRenderedJSON(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Disabled: false, // Firewall enabled
			},
			MCP: &MCPGatewayRuntimeConfig{
				Port:   8080,
				APIKey: "test-key",
			},
		},
	}

	engine := &CopilotEngine{}
	tools := map[string]any{}
	mcpTools := []string{}

	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

	output := yaml.String()

	// Verify the gateway section is present
	require.Contains(t, output, "\"gateway\":", "Gateway section should be present")
	require.Contains(t, output, "\"port\": 8080", "Port should be present")
	require.Contains(t, output, "\"apiKey\": \"test-key\"", "API key should be present")
	require.Contains(t, output, "\"domain\": \"host.docker.internal\"",
		"Domain should be set to host.docker.internal when firewall is enabled")
}

func TestGatewayDomainLocalhostWhenFirewallDisabled(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Disabled: true, // Firewall disabled
			},
			MCP: &MCPGatewayRuntimeConfig{
				Port: 8080,
			},
		},
	}

	engine := &CopilotEngine{}
	tools := map[string]any{}
	mcpTools := []string{}

	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

	output := yaml.String()

	// Verify the domain is set to localhost
	require.Contains(t, output, "\"gateway\":", "Gateway section should be present")
	require.Contains(t, output, "\"domain\": \"localhost\"",
		"Domain should be set to localhost when firewall is disabled")
}
