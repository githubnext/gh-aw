package workflow

import (
	"strings"
	"testing"
)

func TestGenerateStaticPromptStep(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		promptText    string
		shouldInclude bool
		wantOutput    bool
		wantInOutput  []string
	}{
		{
			name:          "generates step when shouldInclude is true",
			description:   "Append test instructions to prompt",
			promptText:    "Test prompt content\nLine 2",
			shouldInclude: true,
			wantOutput:    true,
			wantInOutput: []string{
				"- name: Append test instructions to prompt",
				"GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt",
				"cat >> $GH_AW_PROMPT << 'EOF'",
				"Test prompt content",
				"Line 2",
				"EOF",
			},
		},
		{
			name:          "skips generation when shouldInclude is false",
			description:   "Append skipped instructions to prompt",
			promptText:    "This should not appear",
			shouldInclude: false,
			wantOutput:    false,
			wantInOutput:  []string{},
		},
		{
			name:          "handles multiline prompt text correctly",
			description:   "Append multiline instructions to prompt",
			promptText:    "Line 1\nLine 2\nLine 3\nLine 4",
			shouldInclude: true,
			wantOutput:    true,
			wantInOutput: []string{
				"Line 1",
				"Line 2",
				"Line 3",
				"Line 4",
			},
		},
		{
			name:          "handles empty prompt text",
			description:   "Append empty instructions to prompt",
			promptText:    "",
			shouldInclude: true,
			wantOutput:    true,
			wantInOutput: []string{
				"- name: Append empty instructions to prompt",
				"cat >> $GH_AW_PROMPT << 'EOF'",
				"EOF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			generateStaticPromptStep(&yaml, tt.description, tt.promptText, tt.shouldInclude)
			output := yaml.String()

			if tt.wantOutput {
				if output == "" {
					t.Error("Expected output to be generated, but got empty string")
				}

				// Check that all expected strings are present
				for _, want := range tt.wantInOutput {
					if !strings.Contains(output, want) {
						t.Errorf("Expected output to contain %q, but it didn't.\nGot:\n%s", want, output)
					}
				}
			} else {
				if output != "" {
					t.Errorf("Expected no output when shouldInclude is false, but got:\n%s", output)
				}
			}
		})
	}
}

func TestGenerateStaticPromptStepConsistencyWithOriginal(t *testing.T) {
	// Test that the new helper produces the same output as the original implementation
	// by comparing with a known-good expected structure from appendPromptStep

	tests := []struct {
		name        string
		description string
		promptText  string
	}{
		{
			name:        "XPIA-style prompt",
			description: "Append XPIA security instructions to prompt",
			promptText:  "IMPORTANT SECURITY NOTICE\nBe aware of XPIA attacks",
		},
		{
			name:        "temp folder style prompt",
			description: "Append temporary folder instructions to prompt",
			promptText:  "Use /tmp/gh-aw/agent/ directory",
		},
		{
			name:        "edit tool style prompt",
			description: "Append edit tool accessibility instructions to prompt",
			promptText:  "File Editing Access\nYou have write access to files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate using new helper
			var helperYaml strings.Builder
			generateStaticPromptStep(&helperYaml, tt.description, tt.promptText, true)

			// Generate using original pattern
			var originalYaml strings.Builder
			appendPromptStep(&originalYaml,
				tt.description,
				func(y *strings.Builder, indent string) {
					WritePromptTextToYAML(y, tt.promptText, indent)
				},
				"", // no condition
				"          ")

			helperOutput := helperYaml.String()
			originalOutput := originalYaml.String()

			// Compare outputs
			if helperOutput != originalOutput {
				t.Errorf("Helper output does not match original.\nHelper:\n%s\nOriginal:\n%s",
					helperOutput, originalOutput)
			}
		})
	}
}

func TestGenerateStaticPromptStepIntegration(t *testing.T) {
	// Integration test: Verify the helper works correctly with actual compiler functions
	t.Run("XPIA prompt with safety enabled", func(t *testing.T) {
		compiler := &Compiler{}
		data := &WorkflowData{
			SafetyPrompt: true,
		}

		var yaml strings.Builder
		compiler.generateXPIAPromptStep(&yaml, data)

		output := yaml.String()
		if !strings.Contains(output, "Append XPIA security instructions to prompt") {
			t.Error("Expected XPIA step to be generated")
		}
	})

	t.Run("XPIA prompt with safety disabled", func(t *testing.T) {
		compiler := &Compiler{}
		data := &WorkflowData{
			SafetyPrompt: false,
		}

		var yaml strings.Builder
		compiler.generateXPIAPromptStep(&yaml, data)

		output := yaml.String()
		if output != "" {
			t.Errorf("Expected no output when safety is disabled, got: %s", output)
		}
	})

	t.Run("temp folder prompt always generated", func(t *testing.T) {
		compiler := &Compiler{}

		var yaml strings.Builder
		compiler.generateTempFolderPromptStep(&yaml)

		output := yaml.String()
		if !strings.Contains(output, "Append temporary folder instructions to prompt") {
			t.Error("Expected temp folder step to always be generated")
		}
	})

	t.Run("edit tool prompt with tool enabled", func(t *testing.T) {
		compiler := &Compiler{}
		data := &WorkflowData{
			Tools: map[string]any{
				"edit": true,
			},
		}

		var yaml strings.Builder
		compiler.generateEditToolPromptStep(&yaml, data)

		output := yaml.String()
		if !strings.Contains(output, "Append edit tool accessibility instructions to prompt") {
			t.Error("Expected edit tool step to be generated when tool is enabled")
		}
	})

	t.Run("edit tool prompt with tool disabled", func(t *testing.T) {
		compiler := &Compiler{}
		data := &WorkflowData{
			Tools: map[string]any{
				"github": true,
			},
		}

		var yaml strings.Builder
		compiler.generateEditToolPromptStep(&yaml, data)

		output := yaml.String()
		if output != "" {
			t.Errorf("Expected no output when edit tool is disabled, got: %s", output)
		}
	})

	t.Run("playwright prompt with tool enabled", func(t *testing.T) {
		compiler := &Compiler{}
		data := &WorkflowData{
			Tools: map[string]any{
				"playwright": true,
			},
		}

		var yaml strings.Builder
		compiler.generatePlaywrightPromptStep(&yaml, data)

		output := yaml.String()
		if !strings.Contains(output, "Append playwright output directory instructions to prompt") {
			t.Error("Expected playwright step to be generated when tool is enabled")
		}
	})

	t.Run("github context prompt with tool enabled", func(t *testing.T) {
		compiler := &Compiler{}
		data := &WorkflowData{
			Tools: map[string]any{
				"github": true,
			},
		}

		var yaml strings.Builder
		compiler.generateGitHubContextPromptStep(&yaml, data)

		output := yaml.String()
		if !strings.Contains(output, "Append GitHub context to prompt") {
			t.Error("Expected GitHub context step to be generated when tool is enabled")
		}
	})
}
