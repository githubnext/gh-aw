package workflow

import (
	"strings"
	"testing"
)

func TestGenerateSecretValidationStep(t *testing.T) {
	tests := []struct {
		name        string
		secretName  string
		engineName  string
		docsURL     string
		wantStrings []string
	}{
		{
			name:       "ANTHROPIC_API_KEY validation",
			secretName: "ANTHROPIC_API_KEY",
			engineName: "Claude Code",
			docsURL:    "https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code",
			wantStrings: []string{
				"Validate ANTHROPIC_API_KEY secret",
				"Error: ANTHROPIC_API_KEY secret is not set",
				"The Claude Code engine requires the ANTHROPIC_API_KEY secret to be configured",
				"Please configure this secret in your repository settings",
				"Documentation: https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code",
				"ANTHROPIC_API_KEY secret is configured",
				"ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}",
			},
		},
		{
			name:       "COPILOT_CLI_TOKEN validation",
			secretName: "COPILOT_CLI_TOKEN",
			engineName: "GitHub Copilot CLI",
			docsURL:    "https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default",
			wantStrings: []string{
				"Validate COPILOT_CLI_TOKEN secret",
				"Error: COPILOT_CLI_TOKEN secret is not set",
				"The GitHub Copilot CLI engine requires the COPILOT_CLI_TOKEN secret to be configured",
				"Please configure this secret in your repository settings",
				"Documentation: https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default",
				"COPILOT_CLI_TOKEN secret is configured",
				"COPILOT_CLI_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := GenerateSecretValidationStep(tt.secretName, tt.engineName, tt.docsURL)
			stepContent := strings.Join(step, "\n")

			for _, want := range tt.wantStrings {
				if !strings.Contains(stepContent, want) {
					t.Errorf("GenerateSecretValidationStep() missing expected string:\nwant: %s\ngot: %s", want, stepContent)
				}
			}

			// Verify it has a run block
			if !strings.Contains(stepContent, "run: |") {
				t.Error("Expected step to have 'run: |' block")
			}

			// Verify it has an env section
			if !strings.Contains(stepContent, "env:") {
				t.Error("Expected step to have 'env:' section")
			}

			// Verify it exits with code 1 on failure
			if !strings.Contains(stepContent, "exit 1") {
				t.Error("Expected step to exit with code 1 on validation failure")
			}
		})
	}
}

func TestGenerateMultiSecretValidationStep(t *testing.T) {
	tests := []struct {
		name        string
		secretNames []string
		engineName  string
		docsURL     string
		wantStrings []string
	}{
		{
			name:        "Codex dual secret validation",
			secretNames: []string{"CODEX_API_KEY", "OPENAI_API_KEY"},
			engineName:  "Codex",
			docsURL:     "https://githubnext.github.io/gh-aw/reference/engines/#openai-codex",
			wantStrings: []string{
				"Validate CODEX_API_KEY or OPENAI_API_KEY secret",
				"Neither CODEX_API_KEY nor OPENAI_API_KEY secret is set",
				"The Codex engine requires either CODEX_API_KEY or OPENAI_API_KEY secret to be configured",
				"Please configure one of these secrets in your repository settings",
				"Documentation: https://githubnext.github.io/gh-aw/reference/engines/#openai-codex",
				"CODEX_API_KEY secret is configured",
				"OPENAI_API_KEY secret is configured (using as fallback for CODEX_API_KEY)",
				"CODEX_API_KEY: ${{ secrets.CODEX_API_KEY }}",
				"OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := GenerateMultiSecretValidationStep(tt.secretNames, tt.engineName, tt.docsURL)
			stepContent := strings.Join(step, "\n")

			for _, want := range tt.wantStrings {
				if !strings.Contains(stepContent, want) {
					t.Errorf("GenerateMultiSecretValidationStep() missing expected string:\nwant: %s\ngot: %s", want, stepContent)
				}
			}

			// Verify it has a run block
			if !strings.Contains(stepContent, "run: |") {
				t.Error("Expected step to have 'run: |' block")
			}

			// Verify it has an env section
			if !strings.Contains(stepContent, "env:") {
				t.Error("Expected step to have 'env:' section")
			}

			// Verify it exits with code 1 on failure
			if !strings.Contains(stepContent, "exit 1") {
				t.Error("Expected step to exit with code 1 on validation failure")
			}
		})
	}
}

func TestClaudeEngineHasSecretValidation(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{}

	steps := engine.GetInstallationSteps(workflowData)
	if len(steps) < 1 {
		t.Fatal("Expected at least one installation step")
	}

	// First step should be secret validation (now supports both CLAUDE_CODE_OAUTH_TOKEN and ANTHROPIC_API_KEY)
	firstStep := strings.Join(steps[0], "\n")
	if !strings.Contains(firstStep, "Validate CLAUDE_CODE_OAUTH_TOKEN or ANTHROPIC_API_KEY secret") {
		t.Error("First installation step should validate CLAUDE_CODE_OAUTH_TOKEN or ANTHROPIC_API_KEY secret")
	}
	if !strings.Contains(firstStep, "CLAUDE_CODE_OAUTH_TOKEN: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}") {
		t.Error("Secret validation step should reference secrets.CLAUDE_CODE_OAUTH_TOKEN")
	}
	if !strings.Contains(firstStep, "ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}") {
		t.Error("Secret validation step should reference secrets.ANTHROPIC_API_KEY")
	}
}

func TestCopilotEngineHasSecretValidation(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{}

	steps := engine.GetInstallationSteps(workflowData)
	if len(steps) < 1 {
		t.Fatal("Expected at least one installation step")
	}

	// First step should be secret validation
	firstStep := strings.Join(steps[0], "\n")
	if !strings.Contains(firstStep, "Validate COPILOT_CLI_TOKEN secret") {
		t.Error("First installation step should validate COPILOT_CLI_TOKEN secret")
	}
	if !strings.Contains(firstStep, "COPILOT_CLI_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN }}") {
		t.Error("Secret validation step should reference secrets.COPILOT_CLI_TOKEN")
	}
}

func TestCodexEngineHasSecretValidation(t *testing.T) {
	engine := NewCodexEngine()
	workflowData := &WorkflowData{}

	steps := engine.GetInstallationSteps(workflowData)
	if len(steps) < 1 {
		t.Fatal("Expected at least one installation step")
	}

	// First step should be secret validation
	firstStep := strings.Join(steps[0], "\n")
	if !strings.Contains(firstStep, "Validate CODEX_API_KEY or OPENAI_API_KEY secret") {
		t.Error("First installation step should validate CODEX_API_KEY or OPENAI_API_KEY secret")
	}

	// Should check for both secrets
	if !strings.Contains(firstStep, "CODEX_API_KEY: ${{ secrets.CODEX_API_KEY }}") {
		t.Error("Secret validation step should reference secrets.CODEX_API_KEY")
	}
	if !strings.Contains(firstStep, "OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}") {
		t.Error("Secret validation step should reference secrets.OPENAI_API_KEY")
	}

	// Should have fallback logic
	if !strings.Contains(firstStep, "if [ -z \"$CODEX_API_KEY\" ] && [ -z \"$OPENAI_API_KEY\" ]") {
		t.Error("Should validate that at least one of CODEX_API_KEY or OPENAI_API_KEY is set")
	}
}

func TestCustomEngineDoesNotHaveSecretValidation(t *testing.T) {
	engine := NewCustomEngine()
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			ID: "custom",
		},
	}

	steps := engine.GetInstallationSteps(workflowData)

	// Custom engine should not have any installation steps
	if len(steps) != 0 {
		t.Errorf("Custom engine should not have installation steps, got %d", len(steps))
	}
}
