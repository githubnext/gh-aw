//go:build !integration

package workflow

import (
	"fmt"
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
			docsURL:    "https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code",
			wantStrings: []string{
				"Validate ANTHROPIC_API_KEY secret",
				"Error: ANTHROPIC_API_KEY secret is not set",
				"The Claude Code engine requires the ANTHROPIC_API_KEY secret to be configured",
				"Please configure this secret in your repository settings",
				"Documentation: https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code",
				"<details>",
				"<summary>Agent Environment Validation</summary>",
				"✅ ANTHROPIC_API_KEY: Configured",
				"</details>",
				"ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}",
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
			docsURL:     "https://github.github.com/gh-aw/reference/engines/#openai-codex",
			wantStrings: []string{
				"Validate CODEX_API_KEY or OPENAI_API_KEY secret",
				"run: /opt/gh-aw/actions/validate_multi_secret.sh CODEX_API_KEY OPENAI_API_KEY Codex https://github.github.com/gh-aw/reference/engines/#openai-codex",
				"CODEX_API_KEY: ${{ secrets.CODEX_API_KEY }}",
				"OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}",
			},
		},
		{
			name:        "GitHub Copilot CLI with multi-word engine name",
			secretNames: []string{"COPILOT_GITHUB_TOKEN"},
			engineName:  "GitHub Copilot CLI",
			docsURL:     "https://github.github.com/gh-aw/reference/engines/#github-copilot-default",
			wantStrings: []string{
				"Validate COPILOT_GITHUB_TOKEN secret",
				"run: /opt/gh-aw/actions/validate_multi_secret.sh COPILOT_GITHUB_TOKEN 'GitHub Copilot CLI' https://github.github.com/gh-aw/reference/engines/#github-copilot-default",
				"COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}",
			},
		},
		{
			name:        "Claude Code with single secret",
			secretNames: []string{"ANTHROPIC_API_KEY"},
			engineName:  "Claude Code",
			docsURL:     "https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code",
			wantStrings: []string{
				"Validate ANTHROPIC_API_KEY secret",
				"run: /opt/gh-aw/actions/validate_multi_secret.sh ANTHROPIC_API_KEY 'Claude Code' https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code",
				"ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := GenerateMultiSecretValidationStep(tt.secretNames, tt.engineName, tt.docsURL, nil)
			stepContent := strings.Join(step, "\n")

			for _, want := range tt.wantStrings {
				if !strings.Contains(stepContent, want) {
					t.Errorf("GenerateMultiSecretValidationStep() missing expected string:\nwant: %s\ngot: %s", want, stepContent)
				}
			}

			// Verify it calls the validate_multi_secret.sh script
			if !strings.Contains(stepContent, "/opt/gh-aw/actions/validate_multi_secret.sh") {
				t.Error("Expected step to call validate_multi_secret.sh script")
			}

			// Verify it has an env section
			if !strings.Contains(stepContent, "env:") {
				t.Error("Expected step to have 'env:' section")
			}

			// Verify all secrets are passed as environment variables
			for _, secretName := range tt.secretNames {
				expectedEnvVar := fmt.Sprintf("%s: ${{ secrets.%s }}", secretName, secretName)
				if !strings.Contains(stepContent, expectedEnvVar) {
					t.Errorf("Expected step to have environment variable: %s", expectedEnvVar)
				}
			}

			// Verify step has id field
			if !strings.Contains(stepContent, "id: validate-secret") {
				t.Error("Expected step to have 'id: validate-secret' field")
			}
		})
	}
}

func TestClaudeEngineHasSecretValidation(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{}

	// Secret validation is now returned by GetSecretValidationStep (not GetInstallationSteps)
	step := engine.GetSecretValidationStep(workflowData)
	if len(step) == 0 {
		t.Fatal("Expected a non-empty secret validation step")
	}

	stepContent := strings.Join(step, "\n")
	if !strings.Contains(stepContent, "Validate ANTHROPIC_API_KEY secret") {
		t.Error("Secret validation step should validate ANTHROPIC_API_KEY secret")
	}
	if !strings.Contains(stepContent, "ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}") {
		t.Error("Secret validation step should reference secrets.ANTHROPIC_API_KEY")
	}
}

func TestCopilotEngineHasSecretValidation(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{}

	// Secret validation is now returned by GetSecretValidationStep (not GetInstallationSteps)
	step := engine.GetSecretValidationStep(workflowData)
	if len(step) == 0 {
		t.Fatal("Expected a non-empty secret validation step")
	}

	stepContent := strings.Join(step, "\n")
	if !strings.Contains(stepContent, "Validate COPILOT_GITHUB_TOKEN secret") {
		t.Error("Secret validation step should validate COPILOT_GITHUB_TOKEN secret")
	}
	if !strings.Contains(stepContent, "COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}") {
		t.Error("Secret validation step should reference secrets.COPILOT_GITHUB_TOKEN")
	}
}

func TestCodexEngineHasSecretValidation(t *testing.T) {
	engine := NewCodexEngine()
	workflowData := &WorkflowData{}

	// Secret validation is now returned by GetSecretValidationStep (not GetInstallationSteps)
	step := engine.GetSecretValidationStep(workflowData)
	if len(step) == 0 {
		t.Fatal("Expected a non-empty secret validation step")
	}

	stepContent := strings.Join(step, "\n")
	if !strings.Contains(stepContent, "Validate CODEX_API_KEY or OPENAI_API_KEY secret") {
		t.Error("Secret validation step should validate CODEX_API_KEY or OPENAI_API_KEY secret")
	}

	// Should check for both secrets
	if !strings.Contains(stepContent, "CODEX_API_KEY: ${{ secrets.CODEX_API_KEY }}") {
		t.Error("Secret validation step should reference secrets.CODEX_API_KEY")
	}
	if !strings.Contains(stepContent, "OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}") {
		t.Error("Secret validation step should reference secrets.OPENAI_API_KEY")
	}

	// Should call the validate_multi_secret.sh script with both secret names
	if !strings.Contains(stepContent, "/opt/gh-aw/actions/validate_multi_secret.sh") {
		t.Error("Should call validate_multi_secret.sh script")
	}
	if !strings.Contains(stepContent, "CODEX_API_KEY OPENAI_API_KEY") {
		t.Error("Should pass both CODEX_API_KEY and OPENAI_API_KEY to the script")
	}
}

func TestGenerateMultiSecretValidationStepWithEnvOverrides(t *testing.T) {
	t.Run("override replaces default secret expression", func(t *testing.T) {
		overrides := map[string]string{
			"COPILOT_GITHUB_TOKEN": "${{ secrets.MY_ORG_COPILOT_TOKEN }}",
		}
		step := GenerateMultiSecretValidationStep(
			[]string{"COPILOT_GITHUB_TOKEN"},
			"GitHub Copilot CLI",
			"https://docs.example.com",
			overrides,
		)
		stepContent := strings.Join(step, "\n")

		if !strings.Contains(stepContent, "COPILOT_GITHUB_TOKEN: ${{ secrets.MY_ORG_COPILOT_TOKEN }}") {
			t.Errorf("Expected overridden expression in validation step env, got:\n%s", stepContent)
		}
		if strings.Contains(stepContent, "COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}") {
			t.Errorf("Default expression should be replaced by override, got:\n%s", stepContent)
		}
	})

	t.Run("nil overrides uses default secret expressions", func(t *testing.T) {
		step := GenerateMultiSecretValidationStep(
			[]string{"COPILOT_GITHUB_TOKEN"},
			"GitHub Copilot CLI",
			"https://docs.example.com",
			nil,
		)
		stepContent := strings.Join(step, "\n")

		if !strings.Contains(stepContent, "COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}") {
			t.Errorf("Expected default expression when overrides is nil, got:\n%s", stepContent)
		}
	})

	t.Run("partial override only replaces matching keys", func(t *testing.T) {
		overrides := map[string]string{
			"CODEX_API_KEY": "${{ secrets.MY_ORG_CODEX_KEY }}",
		}
		step := GenerateMultiSecretValidationStep(
			[]string{"CODEX_API_KEY", "OPENAI_API_KEY"},
			"Codex",
			"https://docs.example.com",
			overrides,
		)
		stepContent := strings.Join(step, "\n")

		if !strings.Contains(stepContent, "CODEX_API_KEY: ${{ secrets.MY_ORG_CODEX_KEY }}") {
			t.Errorf("Expected overridden CODEX_API_KEY expression, got:\n%s", stepContent)
		}
		if !strings.Contains(stepContent, "OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}") {
			t.Errorf("Expected default OPENAI_API_KEY expression (not overridden), got:\n%s", stepContent)
		}
	})
}

func TestValidationStepUsesEngineEnvOverride(t *testing.T) {
	tests := []struct {
		name           string
		engine         CodingAgentEngine
		tokenKey       string
		overrideSecret string
	}{
		{
			name:           "Copilot engine validation uses engine.env override",
			engine:         NewCopilotEngine(),
			tokenKey:       "COPILOT_GITHUB_TOKEN",
			overrideSecret: "MY_ORG_COPILOT_TOKEN",
		},
		{
			name:           "Claude engine validation uses engine.env override",
			engine:         NewClaudeEngine(),
			tokenKey:       "ANTHROPIC_API_KEY",
			overrideSecret: "MY_ORG_ANTHROPIC_KEY",
		},
		{
			name:           "Codex engine validation uses engine.env override",
			engine:         NewCodexEngine(),
			tokenKey:       "CODEX_API_KEY",
			overrideSecret: "MY_ORG_CODEX_KEY",
		},
		{
			name:           "Gemini engine validation uses engine.env override",
			engine:         NewGeminiEngine(),
			tokenKey:       "GEMINI_API_KEY",
			overrideSecret: "MY_ORG_GEMINI_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						tt.tokenKey: fmt.Sprintf("${{ secrets.%s }}", tt.overrideSecret),
					},
				},
			}

			// Secret validation is now returned by GetSecretValidationStep (not GetInstallationSteps)
			step := tt.engine.GetSecretValidationStep(workflowData)
			if len(step) == 0 {
				t.Fatal("Expected a non-empty secret validation step")
			}

			validationStep := strings.Join(step, "\n")

			// The validation step should use the overridden secret expression
			expectedExpr := fmt.Sprintf("%s: ${{ secrets.%s }}", tt.tokenKey, tt.overrideSecret)
			if !strings.Contains(validationStep, expectedExpr) {
				t.Errorf("Validation step should use overridden secret expression %q, got:\n%s", expectedExpr, validationStep)
			}
			// The default expression should NOT be present
			defaultExpr := fmt.Sprintf("%s: ${{ secrets.%s }}", tt.tokenKey, tt.tokenKey)
			if strings.Contains(validationStep, defaultExpr) {
				t.Errorf("Validation step should NOT use default expression %q when engine.env overrides it, got:\n%s", defaultExpr, validationStep)
			}
		})
	}
}
