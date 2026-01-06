package stringutil

import "testing"

func TestNormalizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without extension",
			input:    "weekly-research",
			expected: "weekly-research",
		},
		{
			name:     "name with .md extension",
			input:    "weekly-research.md",
			expected: "weekly-research",
		},
		{
			name:     "name with .lock.yml extension",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research",
		},
		{
			name:     "name with dots in filename",
			input:    "my.workflow.md",
			expected: "my.workflow",
		},
		{
			name:     "name with dots and lock.yml",
			input:    "my.workflow.lock.yml",
			expected: "my.workflow",
		},
		{
			name:     "name with other extension",
			input:    "workflow.yaml",
			expected: "workflow.yaml",
		},
		{
			name:     "simple name",
			input:    "agent",
			expected: "agent",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just .md",
			input:    ".md",
			expected: "",
		},
		{
			name:     "just .lock.yml",
			input:    ".lock.yml",
			expected: "",
		},
		{
			name:     "multiple extensions priority",
			input:    "workflow.md.lock.yml",
			expected: "workflow.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWorkflowName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWorkflowName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSafeOutputIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "dash-separated to underscore",
			identifier: "create-issue",
			expected:   "create_issue",
		},
		{
			name:       "already underscore-separated",
			identifier: "create_issue",
			expected:   "create_issue",
		},
		{
			name:       "multiple dashes",
			identifier: "add-comment-to-issue",
			expected:   "add_comment_to_issue",
		},
		{
			name:       "mixed dashes and underscores",
			identifier: "update-pr_status",
			expected:   "update_pr_status",
		},
		{
			name:       "no dashes or underscores",
			identifier: "createissue",
			expected:   "createissue",
		},
		{
			name:       "single dash",
			identifier: "add-comment",
			expected:   "add_comment",
		},
		{
			name:       "trailing dash",
			identifier: "update-",
			expected:   "update_",
		},
		{
			name:       "leading dash",
			identifier: "-create",
			expected:   "_create",
		},
		{
			name:       "consecutive dashes",
			identifier: "create--issue",
			expected:   "create__issue",
		},
		{
			name:       "empty string",
			identifier: "",
			expected:   "",
		},
		{
			name:       "only dashes",
			identifier: "---",
			expected:   "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSafeOutputIdentifier(tt.identifier)
			if result != tt.expected {
				t.Errorf("NormalizeSafeOutputIdentifier(%q) = %q, want %q", tt.identifier, result, tt.expected)
			}
		})
	}
}

func BenchmarkNormalizeWorkflowName(b *testing.B) {
	name := "weekly-research-workflow.lock.yml"
	for i := 0; i < b.N; i++ {
		NormalizeWorkflowName(name)
	}
}

func BenchmarkNormalizeSafeOutputIdentifier(b *testing.B) {
	identifier := "create-pull-request-review-comment"
	for i := 0; i < b.N; i++ {
		NormalizeSafeOutputIdentifier(identifier)
	}
}
