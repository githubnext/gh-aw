package workflow

import (
	"strings"
	"testing"
)

func requireCopilotPreflightAndExecutionSteps(t *testing.T, steps []GitHubActionStep) (string, string) {
	t.Helper()

	if len(steps) != 2 {
		t.Fatalf("Expected 2 execution steps (preflight + execution), got %d", len(steps))
	}

	preflightContent := strings.Join(steps[0], "\n")
	if !strings.Contains(preflightContent, "Copilot pre-flight diagnostic") {
		t.Fatalf("Expected first Copilot step to be the pre-flight diagnostic, got:\n%s", preflightContent)
	}
	if !strings.Contains(preflightContent, "id: copilot-preflight") {
		t.Fatalf("Expected pre-flight step to have id 'copilot-preflight', got:\n%s", preflightContent)
	}
	if !strings.Contains(preflightContent, "copilot_preflight_diagnostic.sh") {
		t.Fatalf("Expected pre-flight step to run the diagnostic script, got:\n%s", preflightContent)
	}

	executionContent := strings.Join(steps[1], "\n")
	if !strings.Contains(executionContent, "Execute GitHub Copilot CLI") {
		t.Fatalf("Expected second Copilot step to execute the CLI, got:\n%s", executionContent)
	}
	if !strings.Contains(executionContent, "id: agentic_execution") {
		t.Fatalf("Expected execution step to have id 'agentic_execution', got:\n%s", executionContent)
	}

	return preflightContent, executionContent
}

// TestEngineAWFEnableApiProxy tests that engines with LLM gateway support
// include --enable-api-proxy flag in AWF commands.
func TestEngineAWFEnableApiProxy(t *testing.T) {
	t.Run("Claude AWF command includes enable-api-proxy flag", func(t *testing.T) {
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

		if !strings.Contains(stepContent, "--enable-api-proxy") {
			t.Error("Expected Claude AWF command to contain '--enable-api-proxy' flag")
		}
	})

	t.Run("Copilot AWF command includes enable-api-proxy flag (supports LLM gateway)", func(t *testing.T) {
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

		_, stepContent := requireCopilotPreflightAndExecutionSteps(t, steps)

		if !strings.Contains(stepContent, "--enable-api-proxy") {
			t.Error("Expected Copilot AWF command to contain '--enable-api-proxy' flag")
		}
	})

	t.Run("Codex AWF command includes enable-api-proxy flag (supports LLM gateway)", func(t *testing.T) {
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

		if !strings.Contains(stepContent, "--enable-api-proxy") {
			t.Error("Expected Codex AWF command to contain '--enable-api-proxy' flag")
		}
	})

	t.Run("Gemini AWF command includes enable-api-proxy flag (supports LLM gateway)", func(t *testing.T) {
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

		// steps[0] = Write Gemini settings, steps[1] = Execute Gemini CLI
		stepContent := strings.Join(steps[1], "\n")

		if !strings.Contains(stepContent, "--enable-api-proxy") {
			t.Error("Expected Gemini AWF command to contain '--enable-api-proxy' flag")
		}
	})
}
