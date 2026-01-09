package workflow

import (
	"strings"
	"testing"
)

// TestEnableHostAccessWithMCPServers tests that --enable-host-access is added to AWF args when MCP servers are configured
func TestEnableHostAccessWithMCPServers(t *testing.T) {
	t.Run("copilot engine adds --enable-host-access when github tool configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			Tools: map[string]any{
				"github": true,
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is present when MCP servers are configured
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to contain '--enable-host-access' when MCP servers are configured")
		}
	})

	t.Run("copilot engine does not add --enable-host-access when no MCP servers", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			Tools: map[string]any{}, // No MCP tools
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is NOT present when no MCP servers
		if strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to NOT contain '--enable-host-access' when no MCP servers are configured")
		}
	})

	t.Run("claude engine adds --enable-host-access when playwright tool configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			Tools: map[string]any{
				"playwright": true,
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is present when MCP servers are configured
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to contain '--enable-host-access' when MCP servers are configured")
		}
	})

	t.Run("codex engine adds --enable-host-access when cache-memory tool configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			Tools: map[string]any{
				"cache-memory": true,
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is present when MCP servers are configured
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to contain '--enable-host-access' when MCP servers are configured")
		}
	})

	t.Run("copilot engine adds --enable-host-access when safe-outputs configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			Tools: map[string]any{}, // No explicit MCP tools
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is present when safe-outputs adds MCP server
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to contain '--enable-host-access' when safe-outputs is configured (adds MCP server)")
		}
	})
}
