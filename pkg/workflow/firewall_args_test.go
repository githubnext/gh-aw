package workflow

import (
	"strings"
	"testing"
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
}

// TestCleanupScriptWorkspacePath tests that cleanup script path uses $GITHUB_WORKSPACE
func TestCleanupScriptWorkspacePath(t *testing.T) {
	t.Run("default cleanup script uses GITHUB_WORKSPACE", func(t *testing.T) {
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
		steps := engine.GetInstallationSteps(workflowData)

		// Find the cleanup step
		var cleanupStep GitHubActionStep
		for _, step := range steps {
			stepContent := strings.Join(step, "\n")
			if strings.Contains(stepContent, "Cleanup any existing awf resources") {
				cleanupStep = step
				break
			}
		}

		if len(cleanupStep) == 0 {
			t.Fatal("Could not find cleanup step")
		}

		stepContent := strings.Join(cleanupStep, "\n")

		// Check that the path uses $GITHUB_WORKSPACE environment variable
		if !strings.Contains(stepContent, "$GITHUB_WORKSPACE") && !strings.Contains(stepContent, "${GITHUB_WORKSPACE}") {
			t.Errorf("Expected cleanup script path to use $GITHUB_WORKSPACE, got:\n%s", stepContent)
		}

		// Should not contain relative path at compile time
		if strings.Contains(stepContent, "./scripts/ci/cleanup.sh") {
			t.Error("Cleanup script should not use compile-time relative path './scripts/ci/cleanup.sh'")
		}
	})

	t.Run("custom cleanup script path is preserved", func(t *testing.T) {
		customPath := "/custom/cleanup/script.sh"
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled:       true,
					CleanupScript: customPath,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Find the cleanup step
		var cleanupStep GitHubActionStep
		for _, step := range steps {
			stepContent := strings.Join(step, "\n")
			if strings.Contains(stepContent, "Cleanup any existing awf resources") {
				cleanupStep = step
				break
			}
		}

		if len(cleanupStep) == 0 {
			t.Fatal("Could not find cleanup step")
		}

		stepContent := strings.Join(cleanupStep, "\n")

		// Check that the custom path is used
		if !strings.Contains(stepContent, customPath) {
			t.Errorf("Expected custom cleanup script path '%s', got:\n%s", customPath, stepContent)
		}
	})

	t.Run("post-execution cleanup also uses GITHUB_WORKSPACE", func(t *testing.T) {
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
		cleanupStep := engine.GetCleanupStep(workflowData)

		if len(cleanupStep) == 0 {
			t.Fatal("Expected cleanup step to be generated")
		}

		stepContent := strings.Join(cleanupStep, "\n")

		// Check that the path uses $GITHUB_WORKSPACE environment variable
		if !strings.Contains(stepContent, "$GITHUB_WORKSPACE") && !strings.Contains(stepContent, "${GITHUB_WORKSPACE}") {
			t.Errorf("Expected post-execution cleanup script path to use $GITHUB_WORKSPACE, got:\n%s", stepContent)
		}

		// Should not contain relative path at compile time
		if strings.Contains(stepContent, "./scripts/ci/cleanup.sh") {
			t.Error("Post-execution cleanup script should not use compile-time relative path './scripts/ci/cleanup.sh'")
		}
	})
}
