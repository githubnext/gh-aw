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

		stepContent := requireCopilotExecutionStep(t, steps)

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

		// steps[0] = Write Gemini Config, steps[1] = Execute Gemini CLI
		stepContent := strings.Join(steps[1], "\n")

		if !strings.Contains(stepContent, "--enable-api-proxy") {
			t.Error("Expected Gemini AWF command to contain '--enable-api-proxy' flag")
		}
	})
}
