package workflow

import "testing"

func TestNormalizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "branch with spaces",
			input:    "assets/Documentation Unbloat",
			expected: "assets/DocumentationUnbloat",
		},
		{
			name:     "branch with multiple spaces",
			input:    "assets/My Test Branch Name",
			expected: "assets/MyTestBranchName",
		},
		{
			name:     "branch with special characters",
			input:    "assets/test@branch#name",
			expected: "assets/testbranchname",
		},
		{
			name:     "branch with valid characters only",
			input:    "assets/valid-branch_name/test",
			expected: "assets/valid-branch_name/test",
		},
		{
			name:     "branch with consecutive slashes",
			input:    "assets//test///branch",
			expected: "assets/test/branch",
		},
		{
			name:     "branch with leading/trailing slashes",
			input:    "/assets/test/",
			expected: "assets/test",
		},
		{
			name:     "branch with leading/trailing dashes",
			input:    "-assets/test-",
			expected: "assets/test",
		},
		{
			name:     "simple branch name",
			input:    "main",
			expected: "main",
		},
		{
			name:     "branch with dots (should be removed)",
			input:    "feature/test.branch.name",
			expected: "feature/testbranchname",
		},
		{
			name:     "branch with parentheses",
			input:    "assets/test(branch)name",
			expected: "assets/testbranchname",
		},
		{
			name:     "branch with unicode characters",
			input:    "assets/тест",
			expected: "assets",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeBranchName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
