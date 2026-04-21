//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestFirewallLogLevelParsing tests handling of deprecated network.firewall parsing.
func TestFirewallLogLevelParsing(t *testing.T) {
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)

	t.Run("network.firewall object is ignored during extraction", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"firewall": map[string]any{
					"log-level": "debug",
				},
			},
		}

		networkPerms := compiler.extractNetworkPermissions(frontmatter)
		if networkPerms == nil {
			t.Fatal("Network permissions should not be nil")
		}

		if networkPerms.Firewall != nil {
			t.Fatalf("Expected network.firewall to be ignored, got: %+v", networkPerms.Firewall)
		}
	})
}

// TestFirewallLogLevelInCopilotEngine tests that the log-level is used in the copilot engine
func TestFirewallLogLevelInCopilotEngine(t *testing.T) {
	t.Run("default log-level is 'info' when not specified", func(t *testing.T) {
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

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that the command contains --log-level info (default)
		if !strings.Contains(stepContent, "--log-level info") {
			t.Errorf("Expected command to contain '--log-level info' (default), got:\n%s", stepContent)
		}
	})

	t.Run("custom log-level is used when specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled:  true,
					LogLevel: "debug",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that the command contains --log-level debug
		if !strings.Contains(stepContent, "--log-level debug") {
			t.Errorf("Expected command to contain '--log-level debug', got:\n%s", stepContent)
		}
	})

	t.Run("log-level can be set to different values", func(t *testing.T) {
		logLevels := []string{"debug", "info", "warn", "error"}

		for _, level := range logLevels {
			workflowData := &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled:  true,
						LogLevel: level,
					},
				},
			}

			engine := NewCopilotEngine()
			steps := engine.GetExecutionSteps(workflowData, "test.log")

			stepContent := requireCopilotExecutionStep(t, steps)

			expectedFlag := "--log-level " + level
			if !strings.Contains(stepContent, expectedFlag) {
				t.Errorf("Expected command to contain '%s', got:\n%s", expectedFlag, stepContent)
			}
		}
	})
}
