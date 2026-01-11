package workflow

import (
	"strings"
	"testing"
)

// TestCodexMCPGatewayConversion verifies that Codex workflows properly configure
// the MCP gateway conversion script to transform gateway JSON output to TOML format.
// This test validates the integration between gateway output and Codex config.toml.
func TestCodexMCPGatewayConversion(t *testing.T) {
	tests := []struct {
		name             string
		tools            map[string]any
		expectedInOutput []string
	}{
		{
			name: "GitHub MCP server with gateway",
			tools: map[string]any{
				"github": map[string]any{
					"mode": "remote",
				},
			},
			expectedInOutput: []string{
				// Gateway JSON configuration passed to start_mcp_gateway.sh
				"cat << MCPCONFIG_EOF | bash /opt/gh-aw/actions/start_mcp_gateway.sh",
				"\"mcpServers\": {",
				"\"gateway\": {",
				// Gateway will detect engine type and call converter
				// The start_mcp_gateway.sh script handles conversion
			},
		},
		{
			name: "Multiple MCP servers with gateway",
			tools: map[string]any{
				"github": map[string]any{
					"mode": "remote",
				},
				"playwright": map[string]any{},
			},
			expectedInOutput: []string{
				"cat << MCPCONFIG_EOF | bash /opt/gh-aw/actions/start_mcp_gateway.sh",
				"\"mcpServers\": {",
				"\"github\": {",
				"\"playwright\": {",
				"\"gateway\": {",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test workflow data
			workflowData := &WorkflowData{
				Tools: tt.tools,
				SandboxConfig: &SandboxConfig{
					MCP: &MCPGatewayRuntimeConfig{
						Port:   8080,
						Domain: "localhost",
						APIKey: "test-key",
					},
				},
			}

			// Compile workflow with Codex engine
			engine := NewCodexEngine()
			var yamlBuilder strings.Builder

			// Get MCP tool list
			mcpTools := []string{}
			if _, ok := tt.tools["github"]; ok {
				mcpTools = append(mcpTools, "github")
			}
			if _, ok := tt.tools["playwright"]; ok {
				mcpTools = append(mcpTools, "playwright")
			}

			// Render MCP config
			engine.RenderMCPConfig(&yamlBuilder, tt.tools, mcpTools, workflowData)
			result := yamlBuilder.String()

			// Verify expected output
			for _, expected := range tt.expectedInOutput {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it was not found\nOutput:\n%s", expected, result)
				}
			}

			// Verify TOML config is written
			if !strings.Contains(result, "cat > /tmp/gh-aw/mcp-config/config.toml << EOF") {
				t.Error("Expected Codex TOML config to be written")
			}

			// Verify history section in TOML
			if !strings.Contains(result, "[history]") || !strings.Contains(result, "persistence = \"none\"") {
				t.Error("Expected TOML config to contain history section")
			}

			// Verify JSON config is also generated for gateway as part of the input
			if !strings.Contains(result, "cat << MCPCONFIG_EOF | bash /opt/gh-aw/actions/start_mcp_gateway.sh") {
				t.Error("Expected JSON config to be piped to MCP gateway startup script")
			}
		})
	}
}

// TestCodexMCPGatewayConfigFormat validates the structure of the MCP gateway
// configuration passed to the conversion script for Codex engine.
func TestCodexMCPGatewayConfigFormat(t *testing.T) {
	// Create test workflow data with gateway configuration
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"mode": "remote",
			},
		},
		SandboxConfig: &SandboxConfig{
			MCP: &MCPGatewayRuntimeConfig{
				Port:   8080,
				Domain: "${MCP_GATEWAY_DOMAIN}",
				APIKey: "${MCP_GATEWAY_API_KEY}",
			},
		},
	}

	// Build gateway config
	gatewayConfig := buildMCPGatewayConfig(workflowData)

	// Verify gateway config structure
	if gatewayConfig == nil {
		t.Fatal("Expected gateway config to be non-nil")
	}

	if gatewayConfig.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", gatewayConfig.Port)
	}

	if gatewayConfig.Domain != "${MCP_GATEWAY_DOMAIN}" {
		t.Errorf("Expected domain ${MCP_GATEWAY_DOMAIN}, got %s", gatewayConfig.Domain)
	}

	if gatewayConfig.APIKey != "${MCP_GATEWAY_API_KEY}" {
		t.Errorf("Expected API key ${MCP_GATEWAY_API_KEY}, got %s", gatewayConfig.APIKey)
	}
}

// TestCodexTOMLFormatGeneration validates that the Codex TOML configuration
// is generated in the correct format before gateway conversion.
func TestCodexTOMLFormatGeneration(t *testing.T) {
	engine := NewCodexEngine()
	var yamlBuilder strings.Builder

	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"mode": "local",
			},
		},
	}

	tools := map[string]any{
		"github": map[string]any{
			"mode": "local",
		},
	}
	mcpTools := []string{"github"}

	// Render MCP config
	engine.RenderMCPConfig(&yamlBuilder, tools, mcpTools, workflowData)
	result := yamlBuilder.String()

	// Verify TOML format sections
	expectedTOMLSections := []string{
		"cat > /tmp/gh-aw/mcp-config/config.toml << EOF",
		"[history]",
		"persistence = \"none\"",
		"[mcp_servers.github]",
		"EOF",
	}

	for _, section := range expectedTOMLSections {
		if !strings.Contains(result, section) {
			t.Errorf("Expected TOML config to contain %q, but it was not found", section)
		}
	}

	// Verify proper TOML format (not JSON)
	// TOML uses = not : for key-value pairs
	if strings.Contains(result, "\"persistence\": \"none\"") {
		t.Error("TOML config should not use JSON colon syntax")
	}

	// Local mode servers don't go through gateway, so no separate JSON file needed
	// The config.toml is used directly by Codex
}

// TestGatewayConfigConversionIntegration validates the end-to-end flow of:
// 1. Generating JSON config for gateway
// 2. Gateway processing and transforming the config
// 3. Conversion script transforming gateway output to Codex TOML
func TestGatewayConfigConversionIntegration(t *testing.T) {
	engine := NewCodexEngine()
	var yamlBuilder strings.Builder

	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"mode": "remote",
			},
			"playwright": map[string]any{},
		},
		SandboxConfig: &SandboxConfig{
			MCP: &MCPGatewayRuntimeConfig{
				Port:   8080,
				Domain: "${MCP_GATEWAY_DOMAIN}",
				APIKey: "${MCP_GATEWAY_API_KEY}",
			},
		},
	}

	tools := map[string]any{
		"github": map[string]any{
			"mode": "remote",
		},
		"playwright": map[string]any{},
	}
	mcpTools := []string{"github", "playwright"}

	// Render complete MCP config
	engine.RenderMCPConfig(&yamlBuilder, tools, mcpTools, workflowData)
	result := yamlBuilder.String()

	// Verify the complete flow is set up:
	
	// 1. JSON config is piped to gateway
	if !strings.Contains(result, "cat << MCPCONFIG_EOF | bash /opt/gh-aw/actions/start_mcp_gateway.sh") {
		t.Error("Expected JSON config to be piped to gateway startup script")
	}

	// 2. Gateway config includes mcpServers section
	if !strings.Contains(result, "\"mcpServers\": {") {
		t.Error("Expected mcpServers section in gateway input")
	}

	// 3. Gateway config includes gateway section with port, domain, apiKey
	if !strings.Contains(result, "\"gateway\": {") {
		t.Error("Expected gateway section in configuration")
	}

	// 4. Gateway outputs to gateway-output.json (handled by start_mcp_gateway.sh)
	// 5. Converter script is called by start_mcp_gateway.sh (engine detection)
	
	// 6. Final TOML config is written to config.toml
	if !strings.Contains(result, "/tmp/gh-aw/mcp-config/config.toml") {
		t.Error("Expected final TOML config path")
	}

	// Verify both formats are present:
	// - TOML for Codex direct config (for non-gateway servers)
	// - JSON for gateway input (piped to start_mcp_gateway.sh)
	hasTOML := strings.Contains(result, "cat > /tmp/gh-aw/mcp-config/config.toml << EOF")
	hasGatewayInput := strings.Contains(result, "cat << MCPCONFIG_EOF | bash /opt/gh-aw/actions/start_mcp_gateway.sh")
	
	if !hasTOML {
		t.Error("Expected TOML configuration to be generated for direct Codex config")
	}
	
	if !hasGatewayInput {
		t.Error("Expected JSON configuration to be piped to gateway startup script")
	}
}
