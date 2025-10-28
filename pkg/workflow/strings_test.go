package workflow

import "testing"

func TestSanitizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "MyWorkflow",
			expected: "myworkflow",
		},
		{
			name:     "spaces to dashes",
			input:    "My Workflow Name",
			expected: "my-workflow-name",
		},
		{
			name:     "colons to dashes",
			input:    "workflow:test",
			expected: "workflow-test",
		},
		{
			name:     "slashes to dashes",
			input:    "workflow/test",
			expected: "workflow-test",
		},
		{
			name:     "backslashes to dashes",
			input:    "workflow\\test",
			expected: "workflow-test",
		},
		{
			name:     "special characters to dashes",
			input:    "workflow@#$test",
			expected: "workflow-test",
		},
		{
			name:     "preserve dots and underscores",
			input:    "workflow.test_name",
			expected: "workflow.test_name",
		},
		{
			name:     "complex name",
			input:    "My Workflow: Test/Build",
			expected: "my-workflow-test-build",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "@#$%^&*()",
			expected: "-",
		},
		{
			name:     "unicode characters",
			input:    "workflow-αβγ-test",
			expected: "workflow-test",
		},
		{
			name:     "mixed case with numbers",
			input:    "MyWorkflow123Test",
			expected: "myworkflow123test",
		},
		{
			name:     "multiple consecutive spaces",
			input:    "workflow   test",
			expected: "workflow-test",
		},
		{
			name:     "preserve hyphens",
			input:    "my-workflow-name",
			expected: "my-workflow-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeWorkflowName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeWorkflowName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShortenCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short command",
			input:    "ls -la",
			expected: "ls -la",
		},
		{
			name:     "exactly 20 characters",
			input:    "12345678901234567890",
			expected: "12345678901234567890",
		},
		{
			name:     "long command gets truncated",
			input:    "this is a very long command that exceeds the limit",
			expected: "this is a very long ...",
		},
		{
			name:     "newlines replaced with spaces",
			input:    "echo hello\nworld",
			expected: "echo hello world",
		},
		{
			name:     "multiple newlines",
			input:    "line1\nline2\nline3",
			expected: "line1 line2 line3",
		},
		{
			name:     "long command with newlines",
			input:    "echo this is\na very long\ncommand with newlines",
			expected: "echo this is a very ...",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenCommand(tt.input)
			if result != tt.expected {
				t.Errorf("ShortenCommand(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
