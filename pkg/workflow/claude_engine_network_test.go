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
		if len(steps) != 3 {
			t.Errorf("Expected 3 installation steps without network permissions (secret validation + Node.js setup + install), got %d", len(steps))
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
		if len(steps) != 5 {
			t.Errorf("Expected 5 installation steps with network permissions (secret validation + Node.js setup + install + settings + hook), got %d", len(steps))
		}

		// Check settings step (4th step, index 3)
		settingsStepStr := strings.Join(steps[3], "\n")
		if !strings.Contains(settingsStepStr, "Generate Claude Settings") {
			t.Error("Fourth step should generate Claude settings")
		}
		if !strings.Contains(settingsStepStr, "/tmp/gh-aw/.claude/settings.json") {
			t.Error("Fourth step should create settings file")
		}

		// Check hook step (5th step, index 4)
		hookStepStr := strings.Join(steps[4], "\n")
		if !strings.Contains(hookStepStr, "Generate Network Permissions Hook") {
			t.Error("Fifth step should generate network permissions hook")
		}
		if !strings.Contains(hookStepStr, ".claude/hooks/network_permissions.py") {
			t.Error("Fifth step should create hook file")
		}
		if !strings.Contains(hookStepStr, "example.com") {
			t.Error("Hook should contain allowed domain example.com")
		}
		if !strings.Contains(hookStepStr, "*.trusted.com") {
			t.Error("Hook should contain allowed domain *.trusted.com")
		}

	})

	t.Run("ExecutionSteps without network permissions", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "test-log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		// Convert steps to string for analysis
		stepYAML := strings.Join(steps[0], "\n")

		// Verify settings parameter is not present
		if strings.Contains(stepYAML, "--settings") {
			t.Error("Settings parameter should not be present without network permissions")
		}

		// Verify model parameter is present in claude_args
		if !strings.Contains(stepYAML, "--model claude-3-5-sonnet-20241022") {
			t.Error("Expected model 'claude-3-5-sonnet-20241022' in step YAML")
		}
	})

	t.Run("ExecutionSteps with network permissions", func(t *testing.T) {
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

		steps := engine.GetExecutionSteps(workflowData, "test-log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		// Convert steps to string for analysis
		stepYAML := strings.Join(steps[0], "\n")

		// Verify settings parameter is present
		if !strings.Contains(stepYAML, "--settings /tmp/gh-aw/.claude/settings.json") {
			t.Error("Settings parameter should be present with network permissions")
		}

		// Verify model parameter is present in claude_args
		if !strings.Contains(stepYAML, "--model claude-3-5-sonnet-20241022") {
			t.Error("Expected model 'claude-3-5-sonnet-20241022' in step YAML")
		}
	})

	t.Run("ExecutionSteps with empty allowed domains (deny all)", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		}

		networkPermissions := &NetworkPermissions{
			Allowed: []string{}, // Empty list means deny all
		}

		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow", EngineConfig: config, NetworkPermissions: networkPermissions}, "test-log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		// Convert steps to string for analysis
		stepYAML := strings.Join(steps[0], "\n")

		// Verify settings parameter is present even with deny-all policy
		if !strings.Contains(stepYAML, "--settings /tmp/gh-aw/.claude/settings.json") {
			t.Error("Settings parameter should be present with deny-all network permissions")
		}
	})

	t.Run("ExecutionSteps with non-Claude engine", func(t *testing.T) {
		config := &EngineConfig{
			ID:    "codex", // Non-Claude engine
			Model: "gpt-4",
		}

		networkPermissions := &NetworkPermissions{
			Allowed: []string{"example.com"},
		}

		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow", EngineConfig: config, NetworkPermissions: networkPermissions}, "test-log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		// Convert steps to string for analysis
		stepYAML := strings.Join(steps[0], "\n")

		// Verify settings parameter is not present for non-Claude engines
		if strings.Contains(stepYAML, "settings:") {
			t.Error("Settings parameter should not be present for non-Claude engine")
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
		if len(steps) != 5 {
			t.Fatalf("Expected 5 installation steps (secret validation + Node.js setup + install + settings + hook), got %d", len(steps))
		}

		// Verify hook generation step (fifth step, index 4)
		hookStep := strings.Join(steps[4], "\n")
		expectedDomains := []string{"api.github.com", "*.example.com", "trusted.org"}
		for _, domain := range expectedDomains {
			if !strings.Contains(hookStep, domain) {
				t.Errorf("Hook step should contain domain '%s'", domain)
			}
		}

		// Get execution steps
		execSteps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow", EngineConfig: config, NetworkPermissions: networkPermissions}, "test-log")
		if len(execSteps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		// Convert steps to string for analysis
		stepYAML := strings.Join(execSteps[0], "\n")

		// Verify settings is configured
		if !strings.Contains(stepYAML, "--settings /tmp/gh-aw/.claude/settings.json") {
			t.Error("Settings parameter should be present")
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

		execSteps1 := engine1.GetExecutionSteps(&WorkflowData{Name: "test", EngineConfig: config, NetworkPermissions: networkPermissions}, "log")
		execSteps2 := engine2.GetExecutionSteps(&WorkflowData{Name: "test", EngineConfig: config, NetworkPermissions: networkPermissions}, "log")

		if len(execSteps1) != len(execSteps2) {
			t.Errorf("Engine instances should produce same number of execution steps, got %d and %d", len(execSteps1), len(execSteps2))
		}

		// Compare the first execution step if they exist
		if len(execSteps1) > 0 && len(execSteps2) > 0 {
			step1YAML := strings.Join(execSteps1[0], "\n")
			step2YAML := strings.Join(execSteps2[0], "\n")
			if step1YAML != step2YAML {
				t.Error("Engine instances should produce identical execution steps")
			}
		}
	})
}
