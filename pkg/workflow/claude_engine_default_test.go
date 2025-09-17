package workflow

import (
	"testing"
)

// TestClaudeEngineNetworkHooksGeneration tests that network hooks are generated
// for Claude engine even when EngineConfig is nil (default engine case)
func TestClaudeEngineNetworkHooksGeneration(t *testing.T) {
	t.Run("Generates network hooks for default Claude engine", func(t *testing.T) {
		// Simulate workflow data similar to ci-doctor (no explicit engine config)
		workflowData := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Mode: "defaults",
			},
			Tools: map[string]any{
				"web-fetch":  true,
				"web-search": true,
			},
			// EngineConfig is nil to simulate default engine usage
			EngineConfig: nil,
		}

		engine := NewClaudeEngine()

		// Test that installation steps are generated
		installSteps := engine.GetInstallationSteps(workflowData)

		if len(installSteps) != 2 {
			t.Errorf("Expected 2 installation steps (Claude settings + network hook), got %d", len(installSteps))
		}

		// Verify the first step is Claude settings generation
		if len(installSteps) > 0 {
			settingsStep := installSteps[0]
			found := false
			for _, line := range settingsStep {
				if stringContains(line, "Generate Claude Settings") {
					found = true
					break
				}
			}
			if !found {
				t.Error("First installation step should be 'Generate Claude Settings'")
			}
		}

		// Verify the second step is network hook generation
		if len(installSteps) > 1 {
			hookStep := installSteps[1]
			found := false
			for _, line := range hookStep {
				if stringContains(line, "Generate Network Permissions Hook") {
					found = true
					break
				}
			}
			if !found {
				t.Error("Second installation step should be 'Generate Network Permissions Hook'")
			}
		}
	})

	t.Run("Uses network settings in execution steps for default engine", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Mode: "defaults",
			},
			Tools: map[string]any{
				"web-fetch":  true,
				"web-search": true,
			},
			// EngineConfig is nil to simulate default engine usage
			EngineConfig: nil,
		}

		engine := NewClaudeEngine()

		// Test that execution steps include settings flag
		execSteps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		// Find the execution step and verify it includes --settings flag
		found := false
		for _, step := range execSteps {
			for _, line := range step {
				if stringContains(line, "--settings /tmp/.claude/settings.json") {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			t.Error("Execution steps should include --settings /tmp/.claude/settings.json when network permissions are configured")
		}
	})

	t.Run("No network hooks when network permissions disabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: nil, // No network permissions
			Tools: map[string]any{
				"web-fetch":  true,
				"web-search": true,
			},
			EngineConfig: nil,
		}

		engine := NewClaudeEngine()

		// Test that no installation steps are generated
		installSteps := engine.GetInstallationSteps(workflowData)

		if len(installSteps) != 0 {
			t.Errorf("Expected 0 installation steps when network permissions disabled, got %d", len(installSteps))
		}
	})
}

// Helper function to check if a string contains a substring
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
