package workflow

import (
	"strings"
	"testing"
)

func TestGenerateGitHubContextPromptStep(t *testing.T) {
	tests := []struct {
		name          string
		tools         map[string]any
		expectContext bool
	}{
		{
			name: "GitHub tool enabled",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []string{"get_issue"},
				},
			},
			expectContext: true,
		},
		{
			name: "GitHub tool enabled with empty config",
			tools: map[string]any{
				"github": true,
			},
			expectContext: true,
		},
		{
			name:          "No GitHub tool",
			tools:         map[string]any{},
			expectContext: false,
		},
		{
			name: "Other tools only",
			tools: map[string]any{
				"edit":       true,
				"web-fetch":  true,
				"web-search": true,
			},
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			var yaml strings.Builder
			data := &WorkflowData{
				Tools:       tt.tools,
				ParsedTools: NewTools(tt.tools), // Populate ParsedTools from Tools map
			}

			compiler.generateGitHubContextPromptStep(&yaml, data)
			output := yaml.String()

			if tt.expectContext {
				if !strings.Contains(output, "Append GitHub context to prompt") {
					t.Error("Expected GitHub context step to be added")
				}
				if !strings.Contains(output, "## GitHub Context") {
					t.Error("Expected GitHub Context header in output")
				}
				if !strings.Contains(output, "github.repository") {
					t.Error("Expected repository context in output")
				}
				if !strings.Contains(output, "github.event.issue.number") {
					t.Error("Expected issue number context in output")
				}
				if !strings.Contains(output, "github.event.discussion.number") {
					t.Error("Expected discussion number context in output")
				}
				if !strings.Contains(output, "github.event.pull_request.number") {
					t.Error("Expected pull request number context in output")
				}
				if !strings.Contains(output, "github.event.comment.id") {
					t.Error("Expected comment ID context in output")
				}
				if !strings.Contains(output, "github.run_id") {
					t.Error("Expected run ID context in output")
				}
			} else {
				if strings.Contains(output, "Append GitHub context to prompt") {
					t.Error("Expected NO GitHub context step when GitHub tool is not enabled")
				}
			}
		})
	}
}

func TestGenerateTemplateRenderingWithGitHubContext(t *testing.T) {
	tests := []struct {
		name                    string
		markdownContent         string
		hasGitHubTool           bool
		expectTemplateRendering bool
	}{
		{
			name:                    "Template in markdown only",
			markdownContent:         "{{#if ${{ github.actor }} }}Hello{{/if}}",
			hasGitHubTool:           false,
			expectTemplateRendering: true,
		},
		{
			name:                    "GitHub tool enabled",
			markdownContent:         "Regular content",
			hasGitHubTool:           true,
			expectTemplateRendering: true,
		},
		{
			name:                    "Both template and GitHub tool",
			markdownContent:         "{{#if ${{ github.actor }} }}Hello{{/if}}",
			hasGitHubTool:           true,
			expectTemplateRendering: true,
		},
		{
			name:                    "Neither template nor GitHub tool",
			markdownContent:         "Regular content",
			hasGitHubTool:           false,
			expectTemplateRendering: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			var yaml strings.Builder

			tools := map[string]any{}
			if tt.hasGitHubTool {
				tools["github"] = true
			}

			data := &WorkflowData{
				MarkdownContent: tt.markdownContent,
				Tools:           tools,
				ParsedTools:     NewTools(tools), // Populate ParsedTools from Tools map
			}

			compiler.generateTemplateRenderingStep(&yaml, data)
			output := yaml.String()

			if tt.expectTemplateRendering {
				if !strings.Contains(output, "Render template conditionals") {
					t.Error("Expected template rendering step to be added")
				}
				if !strings.Contains(output, "renderMarkdownTemplate") {
					t.Error("Expected template rendering script in output")
				}
			} else {
				if strings.Contains(output, "Render template conditionals") {
					t.Error("Expected NO template rendering step")
				}
			}
		})
	}
}

func TestGitHubContextTemplateConditionals(t *testing.T) {
	// Test that the GitHub context prompt contains proper template conditionals
	contextText := githubContextPromptText

	// Check for all expected conditional blocks
	expectedConditionals := []string{
		"{{#if ${{ github.repository }} }}",
		"{{#if ${{ github.event.issue.number }} }}",
		"{{#if ${{ github.event.discussion.number }} }}",
		"{{#if ${{ github.event.pull_request.number }} }}",
		"{{#if ${{ github.event.comment.id }} }}",
		"{{#if ${{ github.run_id }} }}",
	}

	for _, conditional := range expectedConditionals {
		if !strings.Contains(contextText, conditional) {
			t.Errorf("Expected conditional '%s' in GitHub context prompt", conditional)
		}
	}

	// Check that all conditionals have proper closing tags
	openCount := strings.Count(contextText, "{{#if")
	closeCount := strings.Count(contextText, "{{/if}}")
	if openCount != closeCount {
		t.Errorf("Mismatched conditional tags: %d open, %d close", openCount, closeCount)
	}
}
