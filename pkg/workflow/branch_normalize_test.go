package workflow

import (
	"strings"
	"testing"
)

func TestNormalizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid branch name unchanged",
			input:    "assets/my-workflow",
			expected: "assets/my-workflow",
		},
		{
			name:     "Remove invalid characters",
			input:    "assets/${{ github.workflow }}",
			expected: "assets/-github.workflow",
		},
		{
			name:     "Alphanumeric and valid characters",
			input:    "feature/ABC-123_test",
			expected: "feature/ABC-123_test",
		},
		{
			name:     "Multiple invalid characters replaced with single dash",
			input:    "branch@@##name",
			expected: "branch-name",
		},
		{
			name:     "Leading and trailing dashes removed",
			input:    "---branch-name---",
			expected: "branch-name",
		},
		{
			name:     "Max length 128 characters",
			input:    "a" + strings.Repeat("b", 150),
			expected: "a" + strings.Repeat("b", 127), // 128 characters total (a + 127 b's)
		},
		{
			name:     "Trailing dash after truncation removed",
			input:    strings.Repeat("a", 127) + "-" + strings.Repeat("b", 10),
			expected: strings.Repeat("a", 127), // Truncated to 128, then trailing dash removed
		},
		{
			name:     "Special characters in GitHub expression",
			input:    "assets/${{ github.event.issue.number }}",
			expected: "assets/-github.event.issue.number",
		},
		{
			name:     "Spaces replaced with dashes",
			input:    "my branch name",
			expected: "my-branch-name",
		},
		{
			name:     "Mixed valid and invalid characters",
			input:    "test/branch-123_ABC@#$xyz",
			expected: "test/branch-123_ABC-xyz",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only invalid characters",
			input:    "@#$%",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeBranchNameLength(t *testing.T) {
	// Test that output is never longer than 128 characters
	inputs := []string{
		strings.Repeat("a", 200),
		strings.Repeat("abc", 100),
		"prefix/" + strings.Repeat("x", 150),
	}

	for _, input := range inputs {
		result := normalizeBranchName(input)
		if len(result) > 128 {
			t.Errorf("normalizeBranchName(%q) returned string longer than 128 characters: %d", input, len(result))
		}
	}
}
