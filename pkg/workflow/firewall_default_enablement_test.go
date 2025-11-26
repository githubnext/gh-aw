package workflow

import (
	"strings"
	"testing"
)

// TestEnableFirewallByDefault tests the automatic firewall enablement for engines that support it
func TestEnableFirewallByDefault(t *testing.T) {
	t.Run("copilot engine with network restrictions enables firewall by default", func(t *testing.T) {
		engine := NewCopilotEngine()
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com", "api.github.com"},
		}

		enableFirewallByDefault(engine, networkPerms)

		if networkPerms.Firewall == nil {
			t.Error("Expected firewall to be enabled by default for copilot engine with network restrictions")
		}

		if !networkPerms.Firewall.Enabled {
			t.Error("Expected firewall.Enabled to be true")
		}
	})

	t.Run("claude engine with network restrictions enables firewall by default", func(t *testing.T) {
		engine := NewClaudeEngine()
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com", "api.anthropic.com"},
		}

		enableFirewallByDefault(engine, networkPerms)

		if networkPerms.Firewall == nil {
			t.Error("Expected firewall to be enabled by default for claude engine with network restrictions")
		}

		if !networkPerms.Firewall.Enabled {
			t.Error("Expected firewall.Enabled to be true")
		}
	})

	t.Run("copilot engine without network restrictions does not enable firewall", func(t *testing.T) {
		engine := NewCopilotEngine()
		networkPerms := &NetworkPermissions{
			Mode: "defaults",
		}

		enableFirewallByDefault(engine, networkPerms)

		if networkPerms.Firewall != nil {
			t.Error("Expected firewall to remain nil when no network restrictions are present")
		}
	})

	t.Run("copilot engine with explicit firewall config is not overridden", func(t *testing.T) {
		engine := NewCopilotEngine()
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com"},
			Firewall: &FirewallConfig{
				Enabled: false,
			},
		}

		enableFirewallByDefault(engine, networkPerms)

		if networkPerms.Firewall.Enabled {
			t.Error("Expected explicit firewall.Enabled=false to be preserved")
		}
	})

	t.Run("engine without firewall support does not enable firewall", func(t *testing.T) {
		engine := NewCodexEngine()
		networkPerms := &NetworkPermissions{
			Allowed: []string{"example.com"},
		}

		enableFirewallByDefault(engine, networkPerms)

		if networkPerms.Firewall != nil {
			t.Error("Expected firewall to remain nil for engine without firewall support")
		}
	})

	t.Run("nil network permissions does not cause error", func(t *testing.T) {
		engine := NewCopilotEngine()
		// Should not panic
		enableFirewallByDefault(engine, nil)
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
		engine := NewCopilotEngine()
		enableFirewallByDefault(engine, networkPerms)

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
		engine := NewCopilotEngine()
		enableFirewallByDefault(engine, networkPerms)

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
		copilotEngine := NewCopilotEngine()
		steps := copilotEngine.GetInstallationSteps(workflowData)

		// Verify AWF installation step is NOT present
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				t.Error("Expected AWF installation steps to NOT be included when firewall is explicitly disabled")
			}
		}
	})

	t.Run("claude engine with network restrictions enables firewall", func(t *testing.T) {
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

		// Enable firewall by default (should now work for claude engine too)
		engine := NewClaudeEngine()
		enableFirewallByDefault(engine, networkPerms)

		// Verify firewall IS enabled for claude now
		if networkPerms.Firewall == nil {
			t.Error("Expected firewall to be enabled for claude engine with network restrictions")
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
			t.Error("Expected AWF installation steps to be included for claude engine with firewall")
		}
	})
}
