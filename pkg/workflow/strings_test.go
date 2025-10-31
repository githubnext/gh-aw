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

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     *SanitizeOptions
		expected string
	}{
		// Test basic functionality with nil options
		{
			name:     "nil options - simple name",
			input:    "MyWorkflow",
			opts:     nil,
			expected: "myworkflow",
		},
		{
			name:     "nil options - with spaces",
			input:    "My Workflow Name",
			opts:     nil,
			expected: "my-workflow-name",
		},

		// Test with PreserveSpecialChars (SanitizeWorkflowName-like behavior)
		{
			name:  "preserve dots and underscores",
			input: "workflow.test_name",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "workflow.test_name",
		},
		{
			name:  "preserve dots only",
			input: "workflow.test_name",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.'},
			},
			expected: "workflow.test-name",
		},
		{
			name:  "preserve underscores only",
			input: "workflow.test_name",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'_'},
			},
			expected: "workflow-test_name",
		},
		{
			name:  "complex name with preservation",
			input: "My Workflow: Test/Build",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "my-workflow-test-build",
		},

		// Test TrimHyphens option
		{
			name:  "trim hyphens - leading and trailing",
			input: "---workflow---",
			opts: &SanitizeOptions{
				TrimHyphens: true,
			},
			expected: "workflow",
		},
		{
			name:  "no trim hyphens - leading and trailing consolidated",
			input: "---workflow---",
			opts: &SanitizeOptions{
				TrimHyphens: false,
			},
			expected: "-workflow-", // Multiple hyphens are always consolidated
		},
		{
			name:  "trim hyphens - with special chars at edges",
			input: "@@@workflow###",
			opts: &SanitizeOptions{
				TrimHyphens: true,
			},
			expected: "workflow",
		},

		// Test DefaultValue option
		{
			name:  "empty result with default",
			input: "@@@",
			opts: &SanitizeOptions{
				DefaultValue: "default-name",
			},
			expected: "default-name",
		},
		{
			name:  "empty result without default",
			input: "@@@",
			opts: &SanitizeOptions{
				DefaultValue: "",
			},
			expected: "",
		},
		{
			name:  "empty string with default",
			input: "",
			opts: &SanitizeOptions{
				DefaultValue: "github-agentic-workflow",
			},
			expected: "github-agentic-workflow",
		},

		// Test combined options (SanitizeIdentifier-like behavior)
		{
			name:  "identifier-like: simple name",
			input: "Test Workflow Name",
			opts: &SanitizeOptions{
				TrimHyphens:  true,
				DefaultValue: "github-agentic-workflow",
			},
			expected: "test-workflow-name",
		},
		{
			name:  "identifier-like: with underscores",
			input: "Test_Workflow_Name",
			opts: &SanitizeOptions{
				TrimHyphens:  true,
				DefaultValue: "github-agentic-workflow",
			},
			expected: "test-workflow-name",
		},
		{
			name:  "identifier-like: only special chars",
			input: "@#$%!",
			opts: &SanitizeOptions{
				TrimHyphens:  true,
				DefaultValue: "github-agentic-workflow",
			},
			expected: "github-agentic-workflow",
		},

		// Test edge cases
		{
			name:  "multiple consecutive hyphens",
			input: "test---multiple----hyphens",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "test-multiple-hyphens",
		},
		{
			name:  "unicode characters",
			input: "workflow-αβγ-test",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "workflow-test",
		},
		{
			name:  "common separators replacement",
			input: "path/to\\file:name",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "path-to-file-name",
		},
		{
			name:  "preserve hyphens in input",
			input: "my-workflow-name",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "my-workflow-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeName(tt.input, tt.opts)
			if result != tt.expected {
				t.Errorf("SanitizeName(%q, %+v) = %q, want %q", tt.input, tt.opts, result, tt.expected)
			}
		})
	}
}
