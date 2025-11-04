package workflow

import (
	"strings"
	"testing"
)

// TestEnableFirewallByDefaultForCopilot tests the automatic firewall enablement for copilot engine
func TestEnableFirewallByDefaultForCopilot(t *testing.T) {
	t.Run("copilot engine with network restrictions enables firewall by default", func(t *testing.T) {
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com", "api.github.com"},
		}

		enableFirewallByDefaultForCopilot("copilot", networkPerms)

		if networkPerms.Firewall == nil {
			t.Error("Expected firewall to be enabled by default for copilot engine with network restrictions")
		}

		if !networkPerms.Firewall.Enabled {
			t.Error("Expected firewall.Enabled to be true")
		}
	})

	t.Run("copilot engine without network restrictions does not enable firewall", func(t *testing.T) {
		networkPerms := &NetworkPermissions{
			Mode: "defaults",
		}

		enableFirewallByDefaultForCopilot("copilot", networkPerms)

		if networkPerms.Firewall != nil {
			t.Error("Expected firewall to remain nil when no network restrictions are present")
		}
	})

	t.Run("copilot engine with explicit firewall config is not overridden", func(t *testing.T) {
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com"},
			Firewall: &FirewallConfig{
				Enabled: false,
			},
		}

		enableFirewallByDefaultForCopilot("copilot", networkPerms)

		if networkPerms.Firewall.Enabled {
			t.Error("Expected explicit firewall.Enabled=false to be preserved")
		}
	})

	t.Run("non-copilot engine does not enable firewall", func(t *testing.T) {
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com"},
		}

		enableFirewallByDefaultForCopilot("claude", networkPerms)

		if networkPerms.Firewall != nil {
			t.Error("Expected firewall to remain nil for non-copilot engine")
		}
	})

	t.Run("nil network permissions does not cause error", func(t *testing.T) {
		// Should not panic
		enableFirewallByDefaultForCopilot("copilot", nil)
	})
}

// TestCopilotFirewallDefaultIntegration tests the integration with workflow compilation
func TestCopilotFirewallDefaultIntegration(t *testing.T) {
	t.Run("copilot workflow with network restrictions includes AWF installation", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"permissions": map[string]any{
				"contents": "read",
			},
			"engine": "copilot",
			"network": map[string]any{
				"allowed": []any{"example.com", "api.github.com"},
			},
		}

		// Create compiler
		c := NewCompiler(false, "", "test")
		c.SetSkipValidation(true)

		// Extract engine config
		engineSetting, engineConfig := c.ExtractEngineConfig(frontmatter)
		if engineSetting != "copilot" {
			t.Fatalf("Expected engine 'copilot', got '%s'", engineSetting)
		}

		// Extract network permissions
		networkPerms := c.extractNetworkPermissions(frontmatter)
		if networkPerms == nil {
			t.Fatal("Expected network permissions to be extracted")
		}

		// Enable firewall by default
		enableFirewallByDefaultForCopilot(engineConfig.ID, networkPerms)

		// Verify firewall is enabled
		if networkPerms.Firewall == nil {
			t.Error("Expected firewall to be automatically enabled")
		}

		if !networkPerms.Firewall.Enabled {
			t.Error("Expected firewall.Enabled to be true")
		}

		// Create workflow data
		workflowData := &WorkflowData{
			Name:               "test-workflow",
			EngineConfig:       engineConfig,
			NetworkPermissions: networkPerms,
		}

		// Get installation steps
		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Verify AWF installation step is present
		found := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") || strings.Contains(stepStr, "awf --version") {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected AWF installation steps to be included")
		}
	})

	t.Run("copilot workflow with explicit firewall:false does not include AWF", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"permissions": map[string]any{
				"contents": "read",
			},
			"engine": "copilot",
			"network": map[string]any{
				"allowed":  []any{"example.com"},
				"firewall": false,
			},
		}

		// Create compiler
		c := NewCompiler(false, "", "test")
		c.SetSkipValidation(true)

		// Extract engine config
		engineSetting, engineConfig := c.ExtractEngineConfig(frontmatter)
		if engineSetting != "copilot" {
			t.Fatalf("Expected engine 'copilot', got '%s'", engineSetting)
		}

		// Extract network permissions
		networkPerms := c.extractNetworkPermissions(frontmatter)
		if networkPerms == nil {
			t.Fatal("Expected network permissions to be extracted")
		}

		// Verify firewall is explicitly disabled
		if networkPerms.Firewall == nil {
			t.Error("Expected firewall config to be present")
		}

		if networkPerms.Firewall.Enabled {
			t.Error("Expected firewall.Enabled to be false")
		}

		// Enable firewall by default (should not override explicit config)
		enableFirewallByDefaultForCopilot(engineConfig.ID, networkPerms)

		// Verify firewall is still disabled
		if networkPerms.Firewall.Enabled {
			t.Error("Expected firewall to remain disabled when explicitly set to false")
		}

		// Create workflow data
		workflowData := &WorkflowData{
			Name:               "test-workflow",
			EngineConfig:       engineConfig,
			NetworkPermissions: networkPerms,
		}

		// Get installation steps
		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Verify AWF installation step is NOT present
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				t.Error("Expected AWF installation steps to NOT be included when firewall is explicitly disabled")
			}
		}
	})

	t.Run("claude engine with network restrictions does not enable firewall", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"permissions": map[string]any{
				"contents": "read",
			},
			"engine": "claude",
			"network": map[string]any{
				"allowed": []any{"example.com"},
			},
		}

		// Create compiler
		c := NewCompiler(false, "", "test")
		c.SetSkipValidation(true)

		// Extract engine config
		engineSetting, engineConfig := c.ExtractEngineConfig(frontmatter)
		if engineSetting != "claude" {
			t.Fatalf("Expected engine 'claude', got '%s'", engineSetting)
		}

		// Extract network permissions
		networkPerms := c.extractNetworkPermissions(frontmatter)
		if networkPerms == nil {
			t.Fatal("Expected network permissions to be extracted")
		}

		// Enable firewall by default (should not affect non-copilot engines)
		enableFirewallByDefaultForCopilot(engineConfig.ID, networkPerms)

		// Verify firewall is NOT enabled for claude
		if networkPerms.Firewall != nil {
			t.Error("Expected firewall to remain nil for claude engine")
		}
	})
}
