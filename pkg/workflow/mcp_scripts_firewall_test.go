//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestMCPScriptsWithFirewallIncludesHostDockerInternal tests that host.docker.internal
// is added to allowed domains when mcp-scripts is enabled with firewall
func TestMCPScriptsWithFirewallIncludesHostDockerInternal(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
			Allowed: []string{"github.com"},
		},
		MCPScripts: &MCPScriptsConfig{
			Tools: map[string]*MCPScriptToolConfig{
				"test-tool": {
					Name:        "test-tool",
					Description: "A test tool",
					Script:      "return 'test';",
				},
			},
		},
	}

	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(workflowData, "test.log")

	stepContent := requireCopilotExecutionStep(t, steps)

	// Verify that host.docker.internal is in the allowed domains
	if !strings.Contains(stepContent, "host.docker.internal") {
		t.Error("Expected firewall command to include 'host.docker.internal' when mcp-scripts is enabled")
	}

	// Verify the firewall command structure uses config file
	if !strings.Contains(stepContent, "allowDomains") {
		t.Error("Expected command to contain 'allowDomains' in AWF config JSON")
	}
}

// TestGetCopilotAllowedDomainsWithMCPScripts tests the domain calculation function
func TestGetCopilotAllowedDomainsWithMCPScripts(t *testing.T) {
	t.Run("always includes host.docker.internal in default domains", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"github.com"},
		}

		result := GetAllowedDomainsForEngine(constants.CopilotEngine, network, nil, nil)

		if !strings.Contains(result, "host.docker.internal") {
			t.Errorf("Expected result to contain 'host.docker.internal', got: %s", result)
		}

		if !strings.Contains(result, "github.com") {
			t.Errorf("Expected result to contain 'github.com', got: %s", result)
		}
	})

	t.Run("includes host.docker.internal even when mcp-scripts disabled", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"github.com"},
		}

		result := GetAllowedDomainsForEngine(constants.CopilotEngine, network, nil, nil)

		// host.docker.internal is now in default domains, so it's always included
		if !strings.Contains(result, "host.docker.internal") {
			t.Errorf("Expected result to contain 'host.docker.internal' (now in defaults), got: %s", result)
		}

		if !strings.Contains(result, "github.com") {
			t.Errorf("Expected result to contain 'github.com', got: %s", result)
		}
	})

	t.Run("backward compatibility with GetCopilotAllowedDomains", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"github.com"},
		}

		result := GetAllowedDomainsForEngine(constants.CopilotEngine, network, nil, nil)

		// host.docker.internal is now in default domains
		if !strings.Contains(result, "host.docker.internal") {
			t.Errorf("Expected result to contain 'host.docker.internal' (now in defaults), got: %s", result)
		}
	})
}

// TestMCPScripts_T_MCP_050_GoSandboxNetworkIsolation validates that domain configuration
// does not grant unrestricted outbound access by default.
func TestMCPScripts_T_MCP_050_GoSandboxNetworkIsolation(t *testing.T) {
	network := &NetworkPermissions{
		Allowed: []string{"api.github.com"},
	}

	result := GetAllowedDomainsForEngine(constants.CopilotEngine, network, nil, nil)

	if strings.Contains(result, "*") || strings.Contains(result, "0.0.0.0/0") {
		t.Fatalf("T-MCP-050: expected restricted domains, got unrestricted allow-list: %s", result)
	}

	if !strings.Contains(result, "api.github.com") {
		t.Fatalf("T-MCP-050: expected explicit allowed domain to be present, got: %s", result)
	}
}
