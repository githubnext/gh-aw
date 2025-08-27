package workflow

import (
	"strings"
	"testing"
)

func TestClaudeEngineNetworkPermissions(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("InstallationSteps without network permissions", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		}

		steps := engine.GetInstallationSteps(config)
		if len(steps) != 0 {
			t.Errorf("Expected 0 installation steps without network permissions, got %d", len(steps))
		}
	})

	t.Run("InstallationSteps with network permissions", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"example.com", "*.trusted.com"},
				},
			},
		}

		steps := engine.GetInstallationSteps(config)
		if len(steps) != 2 {
			t.Errorf("Expected 2 installation steps with network permissions, got %d", len(steps))
		}

		// Check first step (hook generation)
		hookStepStr := strings.Join(steps[0], "\n")
		if !strings.Contains(hookStepStr, "Generate Network Permissions Hook") {
			t.Error("First step should generate network permissions hook")
		}
		if !strings.Contains(hookStepStr, ".claude/hooks/network_permissions.py") {
			t.Error("First step should create hook file")
		}
		if !strings.Contains(hookStepStr, "example.com") {
			t.Error("Hook should contain allowed domain example.com")
		}
		if !strings.Contains(hookStepStr, "*.trusted.com") {
			t.Error("Hook should contain allowed domain *.trusted.com")
		}

		// Check second step (settings generation)
		settingsStepStr := strings.Join(steps[1], "\n")
		if !strings.Contains(settingsStepStr, "Generate Claude Settings") {
			t.Error("Second step should generate Claude settings")
		}
		if !strings.Contains(settingsStepStr, ".claude/settings.json") {
			t.Error("Second step should create settings file")
		}
		if !strings.Contains(settingsStepStr, "WebFetch|WebSearch") {
			t.Error("Settings should match WebFetch and WebSearch tools")
		}
	})

	t.Run("ExecutionConfig without network permissions", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		}

		execConfig := engine.GetExecutionConfig("test-workflow", "test-log", config)

		// Verify settings parameter is not present
		if settings, exists := execConfig.Inputs["settings"]; exists {
			t.Errorf("Settings parameter should not be present without network permissions, got '%s'", settings)
		}

		// Verify other inputs are still correct
		if execConfig.Inputs["model"] != "claude-3-5-sonnet-20241022" {
			t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", execConfig.Inputs["model"])
		}
	})

	t.Run("ExecutionConfig with network permissions", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"example.com"},
				},
			},
		}

		execConfig := engine.GetExecutionConfig("test-workflow", "test-log", config)

		// Verify settings parameter is present
		if settings, exists := execConfig.Inputs["settings"]; !exists {
			t.Error("Settings parameter should be present with network permissions")
		} else if settings != ".claude/settings.json" {
			t.Errorf("Expected settings '.claude/settings.json', got '%s'", settings)
		}

		// Verify other inputs are still correct
		if execConfig.Inputs["model"] != "claude-3-5-sonnet-20241022" {
			t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", execConfig.Inputs["model"])
		}

		// Verify other expected inputs are present
		expectedInputs := []string{"prompt_file", "anthropic_api_key", "mcp_config", "claude_env", "allowed_tools", "timeout_minutes", "max_turns"}
		for _, input := range expectedInputs {
			if _, exists := execConfig.Inputs[input]; !exists {
				t.Errorf("Expected input '%s' should be present", input)
			}
		}
	})

	t.Run("ExecutionConfig with empty network permissions", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{}, // Empty allowed list means deny-all policy
				},
			},
		}

		execConfig := engine.GetExecutionConfig("test-workflow", "test-log", config)

		// With empty allowed list, we should enforce deny-all policy via settings
		if settings, exists := execConfig.Inputs["settings"]; !exists {
			t.Error("Settings parameter should be present with empty network permissions (deny-all policy)")
		} else if settings != ".claude/settings.json" {
			t.Errorf("Expected settings '.claude/settings.json', got '%s'", settings)
		}
	})

	t.Run("ExecutionConfig version handling with network permissions", func(t *testing.T) {
		config := &EngineConfig{
			ID:      "claude",
			Version: "v1.2.3",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"example.com"},
				},
			},
		}

		execConfig := engine.GetExecutionConfig("test-workflow", "test-log", config)

		// Verify action version uses config version
		expectedAction := "anthropics/claude-code-base-action@v1.2.3"
		if execConfig.Action != expectedAction {
			t.Errorf("Expected action '%s', got '%s'", expectedAction, execConfig.Action)
		}

		// Verify settings parameter is still present
		if settings, exists := execConfig.Inputs["settings"]; !exists {
			t.Error("Settings parameter should be present with network permissions")
		} else if settings != ".claude/settings.json" {
			t.Errorf("Expected settings '.claude/settings.json', got '%s'", settings)
		}
	})
}

func TestNetworkPermissionsIntegration(t *testing.T) {
	t.Run("Full workflow generation", func(t *testing.T) {
		engine := NewClaudeEngine()
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"api.github.com", "*.example.com", "trusted.org"},
				},
			},
		}

		// Get installation steps
		steps := engine.GetInstallationSteps(config)
		if len(steps) != 2 {
			t.Fatalf("Expected 2 installation steps, got %d", len(steps))
		}

		// Verify hook generation step
		hookStep := strings.Join(steps[0], "\n")
		expectedDomains := []string{"api.github.com", "*.example.com", "trusted.org"}
		for _, domain := range expectedDomains {
			if !strings.Contains(hookStep, domain) {
				t.Errorf("Hook step should contain domain '%s'", domain)
			}
		}

		// Verify settings generation step
		settingsStep := strings.Join(steps[1], "\n")
		if !strings.Contains(settingsStep, "PreToolUse") {
			t.Error("Settings step should configure PreToolUse hooks")
		}

		// Get execution config
		execConfig := engine.GetExecutionConfig("test-workflow", "test-log", config)
		if execConfig.Inputs["settings"] != ".claude/settings.json" {
			t.Error("Execution config should reference generated settings file")
		}

		// Verify all pieces work together
		if !HasNetworkPermissions(config) {
			t.Error("Config should have network permissions")
		}
		domains := GetAllowedDomains(config)
		if len(domains) != 3 {
			t.Errorf("Expected 3 allowed domains, got %d", len(domains))
		}
	})

	t.Run("Multiple engine instances consistency", func(t *testing.T) {
		engine1 := NewClaudeEngine()
		engine2 := NewClaudeEngine()

		config := &EngineConfig{
			ID: "claude",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"example.com"},
				},
			},
		}

		steps1 := engine1.GetInstallationSteps(config)
		steps2 := engine2.GetInstallationSteps(config)

		if len(steps1) != len(steps2) {
			t.Error("Different engine instances should generate same number of steps")
		}

		execConfig1 := engine1.GetExecutionConfig("test", "log", config)
		execConfig2 := engine2.GetExecutionConfig("test", "log", config)

		if execConfig1.Inputs["settings"] != execConfig2.Inputs["settings"] {
			t.Error("Different engine instances should generate consistent execution configs")
		}
	})
}
