//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func requireCopilotExecutionStep(t *testing.T, steps []GitHubActionStep) string {
	t.Helper()

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	executionContent := strings.Join(steps[0], "\n")
	if !strings.Contains(executionContent, "Execute GitHub Copilot CLI") {
		t.Fatalf("Expected Copilot step to execute the CLI, got:\n%s", executionContent)
	}
	if !strings.Contains(executionContent, "id: agentic_execution") {
		t.Fatalf("Expected execution step to have id 'agentic_execution', got:\n%s", executionContent)
	}

	return executionContent
}

// TestEngineAWFEnableApiProxy tests that engines with LLM gateway support
// emit API proxy enablement in the AWF config file (new) or as --enable-api-proxy
// CLI flag (legacy: only when the workflow pins an old AWF version).
func TestEngineAWFEnableApiProxy(t *testing.T) {
	// The current default AWF version supports --config, so apiProxy.enabled is expressed
	// in the JSON config file written by the run: step via printf.  Tests verify that
	// "enable" appears inside the JSON blob that is embedded in the step content.
	// For legacy (old version) workflows, --enable-api-proxy is still emitted as a CLI flag.

	t.Run("Claude AWF command includes apiProxy enabled in config file", func(t *testing.T) {
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
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// New behavior: apiProxy.enabled is expressed in the JSON config file.
		// The step content contains the printf command that writes the config JSON.
		if !strings.Contains(stepContent, `"enabled":true`) {
			t.Error("Expected Claude AWF command to contain apiProxy enabled in config JSON")
		}
	})

	t.Run("Copilot AWF command includes apiProxy enabled in config file", func(t *testing.T) {
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

		if !strings.Contains(stepContent, `"enabled":true`) {
			t.Error("Expected Copilot AWF command to contain apiProxy enabled in config JSON")
		}
	})

	t.Run("Codex AWF command includes apiProxy enabled in config file", func(t *testing.T) {
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
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		if !strings.Contains(stepContent, `"enabled":true`) {
			t.Error("Expected Codex AWF command to contain apiProxy enabled in config JSON")
		}
	})

	t.Run("Gemini AWF command includes apiProxy enabled in config file", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "gemini",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewGeminiEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) < 2 {
			t.Fatal("Expected at least two execution steps (settings + execution)")
		}

		// steps[0] = Write Gemini Config, steps[1] = Execute Gemini CLI
		stepContent := strings.Join(steps[1], "\n")

		if !strings.Contains(stepContent, `"enabled":true`) {
			t.Error("Expected Gemini AWF command to contain apiProxy enabled in config JSON")
		}
	})

	t.Run("Pi AWF command includes apiProxy enabled in config file", func(t *testing.T) {
		toolsRaw := map[string]any{
			"github":    map[string]any{"mode": "gh-proxy"},
			"cli-proxy": true,
		}
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "pi",
				Model: "copilot/claude-sonnet-4-20250514",
			},
			Tools:       toolsRaw,
			ParsedTools: NewTools(toolsRaw),
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewPiEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// AWF config JSON embedded in the step must have apiProxy.enabled = true.
		if !strings.Contains(stepContent, `"enabled":true`) {
			t.Error("Expected Pi AWF command to contain apiProxy enabled in config JSON")
		}
	})
}
