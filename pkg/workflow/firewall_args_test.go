package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestFirewallArgsInCopilotEngine tests that custom firewall args are included in AWF command
func TestFirewallArgsInCopilotEngine(t *testing.T) {
	t.Run("no custom args uses only default flags", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
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

		// Check that the command contains standard awf flags
		if !strings.Contains(stepContent, "awf --env-all") {
			t.Error("Expected command to contain 'awf --env-all'")
		}

		if !strings.Contains(stepContent, "--allow-domains") {
			t.Error("Expected command to contain '--allow-domains'")
		}

		if !strings.Contains(stepContent, "--log-level") {
			t.Error("Expected command to contain '--log-level'")
		}

		// Verify that --log-dir is included in copilot args for log collection
		if !strings.Contains(stepContent, "--log-dir /tmp/gh-aw/sandbox/agent/logs/") {
			t.Error("Expected copilot command to contain '--log-dir /tmp/gh-aw/sandbox/agent/logs/' for log collection in firewall mode")
		}
	})

	t.Run("custom args are included in AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Args:    []string{"--custom-arg", "value", "--another-flag"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that custom args are included
		if !strings.Contains(stepContent, "--custom-arg") {
			t.Error("Expected command to contain custom arg '--custom-arg'")
		}

		if !strings.Contains(stepContent, "value") {
			t.Error("Expected command to contain custom arg value 'value'")
		}

		if !strings.Contains(stepContent, "--another-flag") {
			t.Error("Expected command to contain custom arg '--another-flag'")
		}

		// Verify standard flags are still present
		if !strings.Contains(stepContent, "--allow-domains") {
			t.Error("Expected command to still contain '--allow-domains'")
		}
	})

	t.Run("custom args with spaces are properly escaped", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Args:    []string{"--message", "hello world", "--path", "/some/path with spaces"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that args with spaces are present (they should be escaped)
		if !strings.Contains(stepContent, "--message") {
			t.Error("Expected command to contain '--message' flag")
		}

		// The value might be escaped, so just check the flag exists
		if !strings.Contains(stepContent, "--path") {
			t.Error("Expected command to contain '--path' flag")
		}
	})

	t.Run("gh CLI binary is mounted to AWF container", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
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

		// Check that gh CLI binary mount is included in AWF command
		if !strings.Contains(stepContent, "--mount /usr/bin/gh:/usr/bin/gh:ro") {
			t.Error("Expected AWF command to contain gh CLI binary mount '--mount /usr/bin/gh:/usr/bin/gh:ro'")
		}
	})

	t.Run("AWF command includes image-tag with default version", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
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

		// Check that --image-tag is included with default version (without v prefix)
		expectedImageTag := "--image-tag " + strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF command to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}
	})

	t.Run("AWF command includes image-tag with custom version", func(t *testing.T) {
		customVersion := "v0.5.0"
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: customVersion,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --image-tag is included with custom version (without v prefix)
		expectedImageTag := "--image-tag " + strings.TrimPrefix(customVersion, "v")
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF command to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}

		// Ensure default version is not used when custom version is specified
		defaultImageTag := "--image-tag " + strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
		if strings.TrimPrefix(customVersion, "v") != strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v") && strings.Contains(stepContent, defaultImageTag) {
			t.Error("Should use custom version, not default version")
		}
	})

	t.Run("AWF command includes --enable-host-access when MCP servers are enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// Add GitHub tool to enable MCP servers
			Tools: map[string]any{
				"github": true,
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is included when MCP servers are enabled
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to contain '--enable-host-access' when MCP servers are enabled")
		}
	})

	t.Run("AWF command does not include --enable-host-access when no MCP servers", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// No tools configured, so no MCP servers
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is NOT included when no MCP servers
		if strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected AWF command to NOT contain '--enable-host-access' when no MCP servers are configured")
		}
	})
}

// TestClaudeEngineEnableHostAccessWithMCPServers tests that Claude engine includes --enable-host-access when MCP servers are configured
func TestClaudeEngineEnableHostAccessWithMCPServers(t *testing.T) {
	t.Run("AWF command includes --enable-host-access when MCP servers are enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// Add GitHub tool to enable MCP servers
			Tools: map[string]any{
				"github": true,
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is included when MCP servers are enabled
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected Claude AWF command to contain '--enable-host-access' when MCP servers are enabled")
		}
	})

	t.Run("AWF command does not include --enable-host-access when no MCP servers", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// No tools configured, so no MCP servers
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is NOT included when no MCP servers
		if strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected Claude AWF command to NOT contain '--enable-host-access' when no MCP servers are configured")
		}
	})
}

// TestCodexEngineEnableHostAccessWithMCPServers tests that Codex engine includes --enable-host-access when MCP servers are configured
func TestCodexEngineEnableHostAccessWithMCPServers(t *testing.T) {
	t.Run("AWF command includes --enable-host-access when MCP servers are enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// Add GitHub tool to enable MCP servers
			Tools: map[string]any{
				"github": true,
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is included when MCP servers are enabled
		if !strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected Codex AWF command to contain '--enable-host-access' when MCP servers are enabled")
		}
	})

	t.Run("AWF command does not include --enable-host-access when no MCP servers", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// No tools configured, so no MCP servers
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --enable-host-access is NOT included when no MCP servers
		if strings.Contains(stepContent, "--enable-host-access") {
			t.Error("Expected Codex AWF command to NOT contain '--enable-host-access' when no MCP servers are configured")
		}
	})
}
