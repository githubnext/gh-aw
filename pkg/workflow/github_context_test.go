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
					"allowed": []string{"issue_read"},
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
				if !strings.Contains(output, "<github-context>") {
					t.Error("Expected <github-context> XML tag in output")
				}
				// Verify expressions are in the env section (secure pattern)
				if !strings.Contains(output, "${{ github.repository }}") {
					t.Error("Expected repository context in env section")
				}
				if !strings.Contains(output, "${{ github.workspace }}") {
					t.Error("Expected workspace context in env section")
				}
				if !strings.Contains(output, "${{ github.event.issue.number }}") {
					t.Error("Expected issue number context in env section")
				}
				if !strings.Contains(output, "${{ github.event.discussion.number }}") {
					t.Error("Expected discussion number context in env section")
				}
				if !strings.Contains(output, "${{ github.event.pull_request.number }}") {
					t.Error("Expected pull request number context in env section")
				}
				if !strings.Contains(output, "${{ github.event.comment.id }}") {
					t.Error("Expected comment ID context in env section")
				}
				if !strings.Contains(output, "${{ github.run_id }}") {
					t.Error("Expected run ID context in env section")
				}
			} else {
				if strings.Contains(output, "Append GitHub context to prompt") {
					t.Error("Expected NO GitHub context step when GitHub tool is not enabled")
				}
			}
		})
	}
}

// TestGenerateGitHubContextSecurePattern verifies that the GitHub context step uses
// the secure pattern: expressions are extracted into env vars and the heredoc uses
// shell variable references instead of direct template expressions.
func TestGenerateGitHubContextSecurePattern(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	var yaml strings.Builder
	data := &WorkflowData{
		Tools: map[string]any{
			"github": true,
		},
		ParsedTools: NewTools(map[string]any{"github": true}),
	}

	compiler.generateGitHubContextPromptStep(&yaml, data)
	output := yaml.String()

	// Split output into env section and heredoc section
	// Find the heredoc content (between "run: |" and "PROMPT_EOF")
	runIndex := strings.Index(output, "run: |")
	if runIndex == -1 {
		t.Fatal("Expected 'run: |' in output")
	}

	envSection := output[:runIndex]
	heredocSection := output[runIndex:]

	// Verify that expressions appear in the env section with pretty names
	expectedEnvVars := map[string]string{
		"GH_AW_GITHUB_REPOSITORY":                "${{ github.repository }}",
		"GH_AW_GITHUB_WORKSPACE":                 "${{ github.workspace }}",
		"GH_AW_GITHUB_EVENT_ISSUE_NUMBER":        "${{ github.event.issue.number }}",
		"GH_AW_GITHUB_EVENT_DISCUSSION_NUMBER":   "${{ github.event.discussion.number }}",
		"GH_AW_GITHUB_EVENT_PULL_REQUEST_NUMBER": "${{ github.event.pull_request.number }}",
		"GH_AW_GITHUB_EVENT_COMMENT_ID":          "${{ github.event.comment.id }}",
		"GH_AW_GITHUB_RUN_ID":                    "${{ github.run_id }}",
	}

	for envVar, expr := range expectedEnvVars {
		// Check that the expression appears in the env section
		if !strings.Contains(envSection, expr) {
			t.Errorf("Expected expression '%s' in env section, but not found", expr)
		}
		// Check that the pretty env var name is used
		if !strings.Contains(envSection, envVar) {
			t.Errorf("Expected env var name '%s' in env section, but not found", envVar)
		}
	}

	// Verify that the heredoc does NOT contain direct ${{ }} expressions
	// It should only contain ${GH_AW_*} references
	heredocStart := strings.Index(heredocSection, "cat << 'PROMPT_EOF'")
	heredocEnd := strings.Index(heredocSection, "PROMPT_EOF\n")
	if heredocStart == -1 || heredocEnd == -1 {
		t.Fatal("Expected heredoc markers in output")
	}

	heredocContent := heredocSection[heredocStart:heredocEnd]

	// Check that no ${{ }} expressions appear in the heredoc
	if strings.Contains(heredocContent, "${{ ") {
		t.Error("Expected NO ${{ }} expressions in heredoc content (should use ${GH_AW_*} instead)")
	}

	// Verify that shell variable references are used in heredoc
	if !strings.Contains(heredocContent, "${GH_AW_GITHUB_") {
		t.Error("Expected ${GH_AW_GITHUB_*} shell variable references in heredoc content")
	}

	// Verify the envsubst command is used (for shell variable substitution)
	if !strings.Contains(heredocContent, "envsubst") {
		t.Error("Expected envsubst command for shell variable substitution")
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

			compiler.generateInterpolationAndTemplateStep(&yaml, nil, data)
			output := yaml.String()

			if tt.expectTemplateRendering {
				if !strings.Contains(output, "Interpolate variables and render templates") {
					t.Error("Expected interpolation and template step to be added")
				}
				if !strings.Contains(output, "renderMarkdownTemplate") {
					t.Error("Expected template rendering script in output")
				}
			} else {
				if strings.Contains(output, "Interpolate variables and render templates") {
					t.Error("Expected NO interpolation and template step")
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
		"{{#if ${{ github.workspace }} }}",
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
