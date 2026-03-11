package workflow

import (
	"strings"
	"testing"
)

// TestExtractHostname tests the extractHostname helper function.
func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "full URL", input: "https://llm-router.internal.example.com/v1", expected: "llm-router.internal.example.com"},
		{name: "URL with port", input: "https://llm-router.internal.example.com:8443/v1", expected: "llm-router.internal.example.com"},
		{name: "plain hostname", input: "api.openai.com", expected: "api.openai.com"},
		{name: "empty string", input: "", expected: ""},
		{name: "URL without path", input: "https://example.com", expected: "example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractHostname(tc.input)
			if result != tc.expected {
				t.Errorf("extractHostname(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestAWFOpenAIApiTarget tests that --openai-api-target is emitted when OPENAI_BASE_URL is in engine.env.
func TestAWFOpenAIApiTarget(t *testing.T) {
	t.Run("emits --openai-api-target when OPENAI_BASE_URL is set in engine.env", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				Env: map[string]string{
					"OPENAI_BASE_URL": "https://llm-router.internal.example.com/v1",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")
		if !strings.Contains(stepContent, "--openai-api-target llm-router.internal.example.com") {
			t.Errorf("Expected AWF command to contain '--openai-api-target llm-router.internal.example.com', got:\n%s", stepContent)
		}
	})

	t.Run("does not emit --openai-api-target when OPENAI_BASE_URL is absent", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")
		if strings.Contains(stepContent, "--openai-api-target") {
			t.Errorf("Expected AWF command NOT to contain '--openai-api-target', got:\n%s", stepContent)
		}
	})

	t.Run("does not emit --openai-api-target when OPENAI_BASE_URL is invalid", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				Env: map[string]string{
					"OPENAI_BASE_URL": "://invalid-url",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")
		if strings.Contains(stepContent, "--openai-api-target") {
			t.Errorf("Expected AWF command NOT to contain '--openai-api-target' for invalid URL, got:\n%s", stepContent)
		}
	})
}

// TestAWFAnthropicApiTarget tests that --anthropic-api-target is emitted when ANTHROPIC_BASE_URL is in engine.env.
func TestAWFAnthropicApiTarget(t *testing.T) {
	t.Run("emits --anthropic-api-target when ANTHROPIC_BASE_URL is set in engine.env", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
				Env: map[string]string{
					"ANTHROPIC_BASE_URL": "https://llm-router.internal.example.com/v1",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")
		if !strings.Contains(stepContent, "--anthropic-api-target llm-router.internal.example.com") {
			t.Errorf("Expected AWF command to contain '--anthropic-api-target llm-router.internal.example.com', got:\n%s", stepContent)
		}
	})

	t.Run("does not emit --anthropic-api-target when ANTHROPIC_BASE_URL is absent", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")
		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")
		if strings.Contains(stepContent, "--anthropic-api-target") {
			t.Errorf("Expected AWF command NOT to contain '--anthropic-api-target', got:\n%s", stepContent)
		}
	})
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

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

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
