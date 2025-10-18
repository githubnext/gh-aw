package workflow

import (
	"testing"
)

func TestParseLabelsFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []string
	}{
		{
			name: "valid labels array",
			input: map[string]any{
				"labels": []any{"bug", "enhancement", "documentation"},
			},
			expected: []string{"bug", "enhancement", "documentation"},
		},
		{
			name: "empty labels array",
			input: map[string]any{
				"labels": []any{},
			},
			expected: []string{},
		},
		{
			name:     "missing labels field",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "labels with mixed types (filters non-strings)",
			input: map[string]any{
				"labels": []any{"bug", 123, "enhancement", true, "documentation"},
			},
			expected: []string{"bug", "enhancement", "documentation"},
		},
		{
			name: "labels as non-array type",
			input: map[string]any{
				"labels": "not-an-array",
			},
			expected: nil,
		},
		{
			name: "labels with only non-string types",
			input: map[string]any{
				"labels": []any{123, true, 456},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLabelsFromConfig(tt.input)

			// Handle nil vs empty slice comparison
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at index %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestParseTitlePrefixFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			name: "valid title-prefix",
			input: map[string]any{
				"title-prefix": "[bot] ",
			},
			expected: "[bot] ",
		},
		{
			name: "empty title-prefix",
			input: map[string]any{
				"title-prefix": "",
			},
			expected: "",
		},
		{
			name:     "missing title-prefix field",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "title-prefix as non-string type",
			input: map[string]any{
				"title-prefix": 123,
			},
			expected: "",
		},
		{
			name: "title-prefix with special characters",
			input: map[string]any{
				"title-prefix": "[AI-Generated] 🤖 ",
			},
			expected: "[AI-Generated] 🤖 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTitlePrefixFromConfig(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseTargetRepoFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			name: "valid target-repo",
			input: map[string]any{
				"target-repo": "owner/repo",
			},
			expected: "owner/repo",
		},
		{
			name: "wildcard target-repo (returns * for caller to validate)",
			input: map[string]any{
				"target-repo": "*",
			},
			expected: "*",
		},
		{
			name:     "missing target-repo field",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "target-repo as non-string type",
			input: map[string]any{
				"target-repo": 123,
			},
			expected: "",
		},
		{
			name: "target-repo with organization and repo",
			input: map[string]any{
				"target-repo": "github/docs",
			},
			expected: "github/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTargetRepoFromConfig(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Integration tests to verify the helpers work correctly in the parser functions

func TestParseIssuesConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-issue": map[string]any{
			"title-prefix": "[bot] ",
			"labels":       []any{"automation", "ai-generated"},
			"target-repo":  "owner/repo",
		},
	}

	result := compiler.parseIssuesConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TitlePrefix != "[bot] " {
		t.Errorf("expected title-prefix '[bot] ', got %q", result.TitlePrefix)
	}

	if len(result.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(result.Labels))
	}

	if result.TargetRepoSlug != "owner/repo" {
		t.Errorf("expected target-repo 'owner/repo', got %q", result.TargetRepoSlug)
	}
}

func TestParsePullRequestsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request": map[string]any{
			"title-prefix": "[auto] ",
			"labels":       []any{"automated", "needs-review"},
			"target-repo":  "org/project",
		},
	}

	result := compiler.parsePullRequestsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TitlePrefix != "[auto] " {
		t.Errorf("expected title-prefix '[auto] ', got %q", result.TitlePrefix)
	}

	if len(result.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(result.Labels))
	}

	if result.TargetRepoSlug != "org/project" {
		t.Errorf("expected target-repo 'org/project', got %q", result.TargetRepoSlug)
	}
}

func TestParseDiscussionsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-discussion": map[string]any{
			"title-prefix": "[analysis] ",
			"target-repo":  "team/discussions",
		},
	}

	result := compiler.parseDiscussionsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TitlePrefix != "[analysis] " {
		t.Errorf("expected title-prefix '[analysis] ', got %q", result.TitlePrefix)
	}

	if result.TargetRepoSlug != "team/discussions" {
		t.Errorf("expected target-repo 'team/discussions', got %q", result.TargetRepoSlug)
	}
}

func TestParseCommentsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"add-comment": map[string]any{
			"target-repo": "upstream/project",
		},
	}

	result := compiler.parseCommentsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TargetRepoSlug != "upstream/project" {
		t.Errorf("expected target-repo 'upstream/project', got %q", result.TargetRepoSlug)
	}
}

func TestParsePRReviewCommentsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request-review-comment": map[string]any{
			"target-repo": "company/codebase",
		},
	}

	result := compiler.parsePullRequestReviewCommentsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TargetRepoSlug != "company/codebase" {
		t.Errorf("expected target-repo 'company/codebase', got %q", result.TargetRepoSlug)
	}
}

// Test wildcard validation (should return nil for invalid config)

func TestParseIssuesConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-issue": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parseIssuesConfig(outputMap)
	if result != nil {
		t.Errorf("expected nil for wildcard target-repo, got %+v", result)
	}
}

func TestParsePullRequestsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parsePullRequestsConfig(outputMap)
	if result != nil {
		t.Errorf("expected nil for wildcard target-repo, got %+v", result)
	}
}

func TestParseDiscussionsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-discussion": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parseDiscussionsConfig(outputMap)
	if result != nil {
		t.Errorf("expected nil for wildcard target-repo, got %+v", result)
	}
}

func TestParseCommentsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"add-comment": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parseCommentsConfig(outputMap)
	if result != nil {
		t.Errorf("expected nil for wildcard target-repo, got %+v", result)
	}
}

func TestParsePRReviewCommentsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request-review-comment": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parsePullRequestReviewCommentsConfig(outputMap)
	if result != nil {
		t.Errorf("expected nil for wildcard target-repo, got %+v", result)
	}
}
