package workflow

import (
	"strings"
	"testing"
)

func TestLintMCPAllowedPattern_NoGitHubTool(t *testing.T) {
	tools := map[string]any{
		"playwright": true,
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when GitHub tool is not present, got %d", len(warnings))
	}
}

func TestLintMCPAllowedPattern_GitHubToolNotMap(t *testing.T) {
	tools := map[string]any{
		"github": true,
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when GitHub tool is boolean, got %d", len(warnings))
	}
}

func TestLintMCPAllowedPattern_NoAllowedField(t *testing.T) {
	tools := map[string]any{
		"github": map[string]any{
			"mode": "remote",
		},
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when 'allowed' field is not present, got %d", len(warnings))
	}
}

func TestLintMCPAllowedPattern_AlreadyUsingToolsets(t *testing.T) {
	tools := map[string]any{
		"github": map[string]any{
			"allowed":  []any{"get_repository", "list_issues"},
			"toolsets": []any{"repos", "issues"},
		},
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when 'toolsets' is already configured, got %d", len(warnings))
	}
}

func TestLintMCPAllowedPattern_DetectsLegacyPattern(t *testing.T) {
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"get_repository", "list_issues", "pull_request_read"},
		},
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}

	warning := warnings[0]

	// Check tool name
	if warning.ToolName != "github" {
		t.Errorf("Expected tool name 'github', got %s", warning.ToolName)
	}

	// Check allowed list was captured
	if len(warning.AllowedList) != 3 {
		t.Errorf("Expected 3 allowed tools, got %d", len(warning.AllowedList))
	}

	// Check toolsets suggestions were generated
	if len(warning.Toolsets) != 3 {
		t.Errorf("Expected 3 suggested toolsets, got %d: %v", len(warning.Toolsets), warning.Toolsets)
	}

	// Check message contains expected content
	if !strings.Contains(warning.Message, "Legacy 'allowed' pattern detected") {
		t.Errorf("Expected message to contain 'Legacy' text, got: %s", warning.Message)
	}

	if !strings.Contains(warning.Message, "toolsets:") {
		t.Errorf("Expected message to suggest toolsets configuration, got: %s", warning.Message)
	}

	if !strings.Contains(warning.Message, "https://githubnext.github.io/gh-aw/reference/mcp-servers/") {
		t.Errorf("Expected message to contain migration guide link, got: %s", warning.Message)
	}

	// Check suggestion is well-formed
	if !strings.HasPrefix(warning.Suggestion, "toolsets: [") {
		t.Errorf("Expected suggestion to start with 'toolsets: [', got: %s", warning.Suggestion)
	}
}

func TestLintMCPAllowedPattern_WithStringSlice(t *testing.T) {
	// Test with []string instead of []any
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []string{"get_repository", "list_commits"},
		},
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}

	warning := warnings[0]

	// Should suggest repos toolset for these tools
	foundRepos := false
	for _, toolset := range warning.Toolsets {
		if toolset == "repos" {
			foundRepos = true
			break
		}
	}

	if !foundRepos {
		t.Errorf("Expected 'repos' toolset in suggestions, got: %v", warning.Toolsets)
	}
}

func TestLintMCPAllowedPattern_EmptyAllowedList(t *testing.T) {
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{},
		},
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for empty allowed list, got %d", len(warnings))
	}
}

func TestSuggestToolsetsFromAllowedTools(t *testing.T) {
	tests := []struct {
		name          string
		allowedTools  []string
		wantToolsets  []string
		checkContains bool // if true, check contains instead of exact match
	}{
		{
			name:         "repos tools only",
			allowedTools: []string{"get_repository", "list_commits", "get_file_contents"},
			wantToolsets: []string{"repos"},
		},
		{
			name:         "issues tools only",
			allowedTools: []string{"list_issues", "create_issue", "update_issue"},
			wantToolsets: []string{"issues"},
		},
		{
			name:         "mixed toolsets",
			allowedTools: []string{"get_repository", "list_issues", "pull_request_read"},
			wantToolsets: []string{"issues", "pull_requests", "repos"},
		},
		{
			name:         "actions toolset",
			allowedTools: []string{"list_workflows", "get_workflow_run"},
			wantToolsets: []string{"actions"},
		},
		{
			name:          "unknown tools are ignored",
			allowedTools:  []string{"get_repository", "unknown_tool_xyz"},
			wantToolsets:  []string{"repos"},
			checkContains: true,
		},
		{
			name:         "discussions toolset",
			allowedTools: []string{"list_discussions", "create_discussion"},
			wantToolsets: []string{"discussions"},
		},
		{
			name:         "context toolset",
			allowedTools: []string{"get_me", "get_teams"},
			wantToolsets: []string{"context"},
		},
		{
			name:         "users and search",
			allowedTools: []string{"get_user", "search_repositories"},
			wantToolsets: []string{"search", "users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestToolsetsFromAllowedTools(tt.allowedTools)

			if tt.checkContains {
				// Check that all expected toolsets are present
				for _, want := range tt.wantToolsets {
					found := false
					for _, got := range got {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected toolset %s in result, got: %v", want, got)
					}
				}
			} else {
				// Exact match
				if len(got) != len(tt.wantToolsets) {
					t.Errorf("Expected %d toolsets, got %d: %v", len(tt.wantToolsets), len(got), got)
					return
				}

				for i, want := range tt.wantToolsets {
					if got[i] != want {
						t.Errorf("Expected toolset[%d] = %s, got %s", i, want, got[i])
					}
				}
			}
		})
	}
}

func TestBuildLintWarningMessage(t *testing.T) {
	allowedTools := []string{"get_repository", "list_issues"}
	suggestedToolsets := []string{"issues", "repos"}

	message := buildLintWarningMessage(allowedTools, suggestedToolsets)

	// Check key parts of the message
	expectedParts := []string{
		"Legacy 'allowed' pattern detected",
		"Consider migrating to 'toolsets'",
		"Current configuration uses:",
		"allowed: [get_repository, list_issues]",
		"Recommended migration:",
		"toolsets: [issues, repos]",
		"https://githubnext.github.io/gh-aw/reference/mcp-servers/",
	}

	for _, part := range expectedParts {
		if !strings.Contains(message, part) {
			t.Errorf("Expected message to contain %q, got:\n%s", part, message)
		}
	}
}

func TestBuildLintSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		toolsets []string
		want     string
	}{
		{
			name:     "empty toolsets defaults to default",
			toolsets: []string{},
			want:     "toolsets: [default]",
		},
		{
			name:     "single toolset",
			toolsets: []string{"repos"},
			want:     "toolsets: [repos]",
		},
		{
			name:     "multiple toolsets",
			toolsets: []string{"issues", "pull_requests", "repos"},
			want:     "toolsets: [issues, pull_requests, repos]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLintSuggestion(tt.toolsets)
			if got != tt.want {
				t.Errorf("buildLintSuggestion(%v) = %q, want %q", tt.toolsets, got, tt.want)
			}
		})
	}
}

func TestLintMCPAllowedPattern_InvalidAllowedType(t *testing.T) {
	tools := map[string]any{
		"github": map[string]any{
			"allowed": "single_tool", // string instead of array
		},
	}

	warnings := LintMCPAllowedPattern(tools)

	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for invalid 'allowed' type, got %d", len(warnings))
	}
}
