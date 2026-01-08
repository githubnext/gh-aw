package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeMCPWithoutGateway tests that Claude generates standard MCP configuration when gateway is disabled
func TestClaudeMCPWithoutGateway(t *testing.T) {
	// Create workflow data without MCP gateway
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{
				"toolsets": []string{"repos"},
			},
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID: "awf",
			},
			// No MCP gateway configured
		},
	}

	engine := NewClaudeEngine()
	var yaml strings.Builder

	// Render MCP config
	mcpTools := []string{"github"}
	engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)

	output := yaml.String()

	// Verify config is written to Claude's standard location
	assert.Contains(t, output, "/tmp/gh-aw/mcp-config/mcp-servers.json",
		"Claude should generate config at its standard location")

	// Verify it's in JSON format without Copilot-specific fields
	assert.Contains(t, output, `"mcpServers"`, "Should contain mcpServers key")

	// Claude doesn't use "type" field like Copilot does
	// The config should be cleaner JSON format
	assert.NotContains(t, output, `"type": "http"`, 
		"Claude config should not include type field at top level")
}

// TestClaudeMCPWithGatewayEnabled tests that Claude generates config that gateway can read
func TestClaudeMCPWithGatewayEnabled(t *testing.T) {
	// Create workflow data with MCP gateway enabled
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{
				"toolsets": []string{"repos"},
			},
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID: "awf",
			},
			MCP: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/githubnext/gh-aw-mcpg",
				Port:      8080,
			},
		},
		Features: map[string]any{
			string(constants.MCPGatewayFeatureFlag): true,
		},
	}

	engine := NewClaudeEngine()
	var yaml strings.Builder

	// Render MCP config
	mcpTools := []string{"github"}
	engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)

	output := yaml.String()

	// Verify config is still written to Claude's standard location
	// The gateway script will read from this location
	assert.Contains(t, output, "/tmp/gh-aw/mcp-config/mcp-servers.json",
		"Claude should generate config at its standard location for gateway to read")

	// Verify it contains mcpServers structure
	assert.Contains(t, output, `"mcpServers"`, "Should contain mcpServers key")
	assert.Contains(t, output, `"github"`, "Should contain github server config")

	// Now test that the gateway step generation uses the correct file
	var gatewayYaml strings.Builder
	generateMCPGatewayStepInline(&gatewayYaml, engine, workflowData)

	gatewayOutput := gatewayYaml.String()

	// Verify gateway environment variables are set
	assert.Contains(t, gatewayOutput, "export MCP_GATEWAY_PORT=", 
		"Gateway should set port environment variable")
	assert.Contains(t, gatewayOutput, "export GH_AW_ENGINE=\"claude\"",
		"Gateway should export engine type for converter selection")
}

// TestGatewayScriptEngineDetection tests that the gateway startup script can detect engine type
func TestGatewayScriptEngineDetection(t *testing.T) {
	tests := []struct {
		name           string
		engine         CodingAgentEngine
		expectedEngine string
		configPath     string
	}{
		{
			name:           "detects claude engine",
			engine:         NewClaudeEngine(),
			expectedEngine: "claude",
			configPath:     "/tmp/gh-aw/mcp-config/mcp-servers.json",
		},
		{
			name:           "detects copilot engine",
			engine:         NewCopilotEngine(),
			expectedEngine: "copilot",
			configPath:     "/home/runner/.copilot/mcp-config.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name: "test-workflow",
				Tools: map[string]any{
					"github": map[string]any{},
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
					MCP: &MCPGatewayRuntimeConfig{
						Container: "ghcr.io/githubnext/gh-aw-mcpg",
					},
				},
				Features: map[string]any{
					string(constants.MCPGatewayFeatureFlag): true,
				},
			}

			var yaml strings.Builder
			generateMCPGatewayStepInline(&yaml, tt.engine, workflowData)
			output := yaml.String()

			// Verify the engine type is exported
			require.Contains(t, output, "export GH_AW_ENGINE=\""+tt.expectedEngine+"\"",
				"Gateway should export correct engine type")

			// Verify the gateway startup script is called
			assert.Contains(t, output, "bash /opt/gh-aw/actions/start_mcp_gateway.sh",
				"Should call gateway startup script")
		})
	}
}

// TestClaudeConfigPathForGateway tests that the gateway uses the correct config file path for Claude
func TestClaudeConfigPathForGateway(t *testing.T) {
	// This test documents the expected behavior:
	// 1. Claude generates config to /tmp/gh-aw/mcp-config/mcp-servers.json
	// 2. Gateway script reads from that location when engine is claude
	// 3. Gateway converts and writes back to the same location

	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{},
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID: "awf",
			},
			MCP: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/githubnext/gh-aw-mcpg",
			},
		},
		Features: map[string]any{
			string(constants.MCPGatewayFeatureFlag): true,
		},
	}

	engine := NewClaudeEngine()

	// Generate MCP config
	var configYaml strings.Builder
	mcpTools := []string{"github"}
	engine.RenderMCPConfig(&configYaml, workflowData.Tools, mcpTools, workflowData)
	configOutput := configYaml.String()

	// Generate gateway startup
	var gatewayYaml strings.Builder
	generateMCPGatewayStepInline(&gatewayYaml, engine, workflowData)
	gatewayOutput := gatewayYaml.String()

	// Verify Claude writes to its standard location
	assert.Contains(t, configOutput, "/tmp/gh-aw/mcp-config/mcp-servers.json",
		"Claude should write config to its standard location")

	// Verify gateway exports claude engine type
	assert.Contains(t, gatewayOutput, "export GH_AW_ENGINE=\"claude\"",
		"Gateway should know it's running with Claude engine")

	// The start_mcp_gateway.sh script will use GH_AW_ENGINE to determine which config file to read
	// This is tested by the shell script itself and integration tests
}

// TestClaudeGatewayAgentDisabled tests that when agent is disabled, gateway uses localhost
func TestClaudeGatewayAgentDisabled(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{},
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Disabled: true, // Agent explicitly disabled
			},
			MCP: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/githubnext/gh-aw-mcpg",
			},
		},
		Features: map[string]any{
			string(constants.MCPGatewayFeatureFlag): true,
		},
	}

	engine := NewClaudeEngine()
	var yaml strings.Builder
	generateMCPGatewayStepInline(&yaml, engine, workflowData)
	output := yaml.String()

	// When agent is disabled, gateway should use localhost instead of host.docker.internal
	assert.Contains(t, output, "export MCP_GATEWAY_DOMAIN=\"localhost\"",
		"Gateway should use localhost when agent is disabled")
}

// TestClaudeGatewayAgentEnabled tests that when agent is enabled, gateway uses host.docker.internal
func TestClaudeGatewayAgentEnabled(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{},
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID: "awf", // Agent enabled
			},
			MCP: &MCPGatewayRuntimeConfig{
				Container: "ghcr.io/githubnext/gh-aw-mcpg",
			},
		},
		Features: map[string]any{
			string(constants.MCPGatewayFeatureFlag): true,
		},
	}

	engine := NewClaudeEngine()
	var yaml strings.Builder
	generateMCPGatewayStepInline(&yaml, engine, workflowData)
	output := yaml.String()

	// When agent is enabled, gateway should use host.docker.internal for Docker networking
	assert.Contains(t, output, "export MCP_GATEWAY_DOMAIN=\"host.docker.internal\"",
		"Gateway should use host.docker.internal when agent is enabled")
}
