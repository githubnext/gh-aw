package workflow

import (
	"strings"
	"testing"
)

func TestClaudeEngineNetworkPermissions(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("InstallationSteps without network permissions", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
		}

		steps := engine.GetInstallationSteps(workflowData)
		if len(steps) != 0 {
			t.Errorf("Expected 0 installation steps without network permissions, got %d", len(steps))
		}
	})

	t.Run("InstallationSteps with network permissions", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com", "*.trusted.com"},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)
		if len(steps) != 2 {
			t.Errorf("Expected 2 installation steps with network permissions, got %d", len(steps))
		}

		// Check first step (settings generation)
		settingsStepStr := strings.Join(steps[0], "\n")
		if !strings.Contains(settingsStepStr, "Generate Claude Settings") {
			t.Error("First step should generate Claude settings")
		}
		if !strings.Contains(settingsStepStr, ".claude/settings.json") {
			t.Error("First step should create settings file")
		}

		// Check second step (hook generation)
		hookStepStr := strings.Join(steps[1], "\n")
		if !strings.Contains(hookStepStr, "Generate Network Permissions Hook") {
			t.Error("Second step should generate network permissions hook")
		}
		if !strings.Contains(hookStepStr, ".claude/hooks/network_permissions.py") {
			t.Error("Second step should create hook file")
		}
		if !strings.Contains(hookStepStr, "example.com") {
			t.Error("Hook should contain allowed domain example.com")
		}
		if !strings.Contains(hookStepStr, "*.trusted.com") {
			t.Error("Hook should contain allowed domain *.trusted.com")
		}

	})

	t.Run("ExecutionConfig without network permissions", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
		}

		execConfig := engine.GetExecutionConfig(workflowData, "test-log")

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
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com"},
			},
		}

		execConfig := engine.GetExecutionConfig(workflowData, "test-log")

		// Verify settings parameter is present
		if settings, exists := execConfig.Inputs["settings"]; !exists {
			t.Error("Settings parameter should be present with network permissions")
		} else if settings != ".claude/settings.json" {
			t.Errorf("Expected settings parameter '.claude/settings.json', got '%s'", settings)
		}

		// Verify other inputs are still correct
		if execConfig.Inputs["model"] != "claude-3-5-sonnet-20241022" {
			t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", execConfig.Inputs["model"])
		}
	})

	t.Run("ExecutionConfig with empty allowed domains (deny all)", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		}

		networkPermissions := &NetworkPermissions{
			Allowed: []string{}, // Empty list means deny all
		}

		execConfig := engine.GetExecutionConfig(&WorkflowData{Name: "test-workflow", EngineConfig: config, NetworkPermissions: networkPermissions}, "test-log")

		// Verify settings parameter is present even with deny-all policy
		if settings, exists := execConfig.Inputs["settings"]; !exists {
			t.Error("Settings parameter should be present with deny-all network permissions")
		} else if settings != ".claude/settings.json" {
			t.Errorf("Expected settings parameter '.claude/settings.json', got '%s'", settings)
		}
	})

	t.Run("ExecutionConfig with non-Claude engine", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "codex", // Non-Claude engine
			Model: "gpt-4",
		}

		networkPermissions := &NetworkPermissions{
			Allowed: []string{"example.com"},
		}

		execConfig := engine.GetExecutionConfig(&WorkflowData{Name: "test-workflow", EngineConfig: config, NetworkPermissions: networkPermissions}, "test-log")

		// Verify settings parameter is not present for non-Claude engines
		if settings, exists := execConfig.Inputs["settings"]; exists {
			t.Errorf("Settings parameter should not be present for non-Claude engine, got '%s'", settings)
		}
	})
}

func TestNetworkPermissionsIntegration(t *testing.T) {
	t.Run("Full workflow generation", func(t *testing.T) {
		engine := NewClaudeEngine()
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		}

		networkPermissions := &NetworkPermissions{
			Allowed: []string{"api.github.com", "*.example.com", "trusted.org"},
		}

		// Get installation steps
		steps := engine.GetInstallationSteps(&WorkflowData{EngineConfig: config, NetworkPermissions: networkPermissions})
		if len(steps) != 2 {
			t.Fatalf("Expected 2 installation steps, got %d", len(steps))
		}

		// Verify hook generation step (second step)
		hookStep := strings.Join(steps[1], "\n")
		expectedDomains := []string{"api.github.com", "*.example.com", "trusted.org"}
		for _, domain := range expectedDomains {
			if !strings.Contains(hookStep, domain) {
				t.Errorf("Hook step should contain domain '%s'", domain)
			}
		}

		// Get execution config
		execConfig := engine.GetExecutionConfig(&WorkflowData{Name: "test-workflow", EngineConfig: config, NetworkPermissions: networkPermissions}, "test-log")

		// Verify settings is configured
		if settings, exists := execConfig.Inputs["settings"]; !exists {
			t.Error("Settings parameter should be present")
		} else if settings != ".claude/settings.json" {
			t.Errorf("Expected settings parameter '.claude/settings.json', got '%s'", settings)
		}

		// Test the GetAllowedDomains function
		domains := GetAllowedDomains(networkPermissions)
		if len(domains) != 3 {
			t.Fatalf("Expected 3 allowed domains, got %d", len(domains))
		}

		expectedDomainsList := []string{"api.github.com", "*.example.com", "trusted.org"}
		for i, expected := range expectedDomainsList {
			if domains[i] != expected {
				t.Errorf("Expected domain %d to be '%s', got '%s'", i, expected, domains[i])
			}
		}
	})

	t.Run("Engine consistency", func(t *testing.T) {
		engine1 := NewClaudeEngine()
		engine2 := NewClaudeEngine()

		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		}

		networkPermissions := &NetworkPermissions{
			Allowed: []string{"example.com"},
		}

		steps1 := engine1.GetInstallationSteps(&WorkflowData{EngineConfig: config, NetworkPermissions: networkPermissions})
		steps2 := engine2.GetInstallationSteps(&WorkflowData{EngineConfig: config, NetworkPermissions: networkPermissions})

		if len(steps1) != len(steps2) {
			t.Errorf("Engine instances should produce same number of steps, got %d and %d", len(steps1), len(steps2))
		}

		execConfig1 := engine1.GetExecutionConfig(&WorkflowData{Name: "test", EngineConfig: config, NetworkPermissions: networkPermissions}, "log")
		execConfig2 := engine2.GetExecutionConfig(&WorkflowData{Name: "test", EngineConfig: config, NetworkPermissions: networkPermissions}, "log")

		if execConfig1.Action != execConfig2.Action {
			t.Errorf("Engine instances should produce same action, got '%s' and '%s'", execConfig1.Action, execConfig2.Action)
		}
	})
}
