package workflow

import (
	"strings"
	"testing"
)

func TestAppendPromptStep(t *testing.T) {
	tests := []struct {
		name      string
		stepName  string
		condition string
		wantSteps []string
	}{
		{
			name:      "basic step without condition",
			stepName:  "Append test instructions to prompt",
			condition: "",
			wantSteps: []string{
				"- name: Append test instructions to prompt",
				"env:",
				"GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt",
				"run: |",
				`cat >> "$GH_AW_PROMPT" << 'PROMPT_EOF'`,
				"Test prompt content",
				"PROMPT_EOF",
			},
		},
		{
			name:      "step with condition",
			stepName:  "Append conditional instructions to prompt",
			condition: "github.event.issue != null",
			wantSteps: []string{
				"- name: Append conditional instructions to prompt",
				"if: github.event.issue != null",
				"env:",
				"GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt",
				"run: |",
				`cat >> "$GH_AW_PROMPT" << 'PROMPT_EOF'`,
				"Conditional prompt content",
				"PROMPT_EOF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			// Call the helper with a simple renderer
			var promptContent string
			if tt.condition == "" {
				promptContent = "Test prompt content"
			} else {
				promptContent = "Conditional prompt content"
			}

			appendPromptStep(&yaml, tt.stepName, func(y *strings.Builder, indent string) {
				WritePromptTextToYAML(y, promptContent, indent)
			}, tt.condition, "          ")

			result := yaml.String()

			// Check that all expected strings are present
			for _, want := range tt.wantSteps {
				if !strings.Contains(result, want) {
					t.Errorf("Expected output to contain %q, but it didn't.\nGot:\n%s", want, result)
				}
			}
		})
	}
}

func TestAppendPromptStepWithHeredoc(t *testing.T) {
	tests := []struct {
		name      string
		stepName  string
		content   string
		wantSteps []string
	}{
		{
			name:     "basic heredoc step",
			stepName: "Append structured data to prompt",
			content:  "Structured content line 1\nStructured content line 2",
			wantSteps: []string{
				"- name: Append structured data to prompt",
				"env:",
				"GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt",
				"run: |",
				`cat >> "$GH_AW_PROMPT" << 'PROMPT_EOF'`,
				"Structured content line 1",
				"Structured content line 2",
				"PROMPT_EOF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			appendPromptStepWithHeredoc(&yaml, tt.stepName, func(y *strings.Builder) {
				y.WriteString(tt.content)
			})

			result := yaml.String()

			// Check that all expected strings are present
			for _, want := range tt.wantSteps {
				if !strings.Contains(result, want) {
					t.Errorf("Expected output to contain %q, but it didn't.\nGot:\n%s", want, result)
				}
			}
		})
	}
}

func TestAppendPromptStepWithHeredoc_ShellSpecialCharacters(t *testing.T) {
	// Test that shell-unfriendly characters are properly protected by the quoted heredoc
	// This is critical for preventing shell injection vulnerabilities
	var yaml strings.Builder

	content := "Use `gh` command and $(command) or ${var} expansion"

	appendPromptStepWithHeredoc(&yaml, "Test shell protection", func(y *strings.Builder) {
		y.WriteString(content)
	})

	result := yaml.String()

	// Should use quoted heredoc to prevent shell expansion
	if !strings.Contains(result, `<< 'PROMPT_EOF'`) {
		t.Error("Expected quoted heredoc delimiter to prevent shell expansion")
	}

	// Original content with backticks should be preserved
	if !strings.Contains(result, "`gh`") {
		t.Error("Expected backticks to be preserved in output")
	}

	// Dollar sign expressions should be preserved, not expanded
	if !strings.Contains(result, "$(command)") {
		t.Error("Expected $(command) to be preserved in output")
	}
	if !strings.Contains(result, "${var}") {
		t.Error("Expected ${var} to be preserved in output")
	}
}

func TestPromptStepRefactoringConsistency(t *testing.T) {
	// Test that the refactored functions produce the same output as the original implementation
	// by comparing with a known-good expected structure

	t.Run("temp_folder generates expected structure", func(t *testing.T) {
		var yaml strings.Builder
		compiler := &Compiler{}
		compiler.generateTempFolderPromptStep(&yaml)

		result := yaml.String()

		// Verify key elements are present
		if !strings.Contains(result, "Append temporary folder instructions to prompt") {
			t.Error("Expected step name for temp folder not found")
		}
		if !strings.Contains(result, "GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt") {
			t.Error("Expected GH_AW_PROMPT env variable not found")
		}
		if !strings.Contains(result, `cat >> "$GH_AW_PROMPT" << 'PROMPT_EOF'`) {
			t.Error("Expected heredoc start not found")
		}
	})

	t.Run("xpia generates expected structure with safety enabled", func(t *testing.T) {
		var yaml strings.Builder
		compiler := &Compiler{}
		data := &WorkflowData{
			SafetyPrompt: true,
		}
		compiler.generateXPIAPromptStep(&yaml, data)

		result := yaml.String()

		// Verify key elements are present
		if !strings.Contains(result, "Append XPIA security instructions to prompt") {
			t.Error("Expected step name for XPIA not found")
		}
		if !strings.Contains(result, "GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt") {
			t.Error("Expected GH_AW_PROMPT env variable not found")
		}
	})

	t.Run("xpia skips generation with safety disabled", func(t *testing.T) {
		var yaml strings.Builder
		compiler := &Compiler{}
		data := &WorkflowData{
			SafetyPrompt: false,
		}
		compiler.generateXPIAPromptStep(&yaml, data)

		result := yaml.String()

		// Should be empty when safety is disabled
		if result != "" {
			t.Errorf("Expected no output when SafetyPrompt is false, got: %s", result)
		}
	})
}
