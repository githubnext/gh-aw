package stringutil

import "testing"

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "message with no secrets",
			message:  "This is a regular error message",
			expected: "This is a regular error message",
		},
		{
			name:     "message with snake_case secret",
			message:  "Error accessing MY_SECRET_KEY",
			expected: "Error accessing [REDACTED]",
		},
		{
			name:     "message with multiple secrets",
			message:  "Failed to use API_TOKEN and DATABASE_PASSWORD",
			expected: "Failed to use [REDACTED] and [REDACTED]",
		},
		{
			name:     "message with PascalCase secret",
			message:  "Invalid GitHubToken provided",
			expected: "Invalid [REDACTED] provided",
		},
		{
			name:     "message with workflow keyword (not redacted)",
			message:  "Error in GITHUB_ACTIONS workflow",
			expected: "Error in [REDACTED] workflow",
		},
		{
			name:     "message with GITHUB keyword (not redacted)",
			message:  "GITHUB is not responding",
			expected: "GITHUB is not responding",
		},
		{
			name:     "message with PATH keyword (not redacted)",
			message:  "PATH variable is not set",
			expected: "PATH variable is not set",
		},
		{
			name:     "complex message with mixed secrets",
			message:  "Failed to authenticate with DEPLOY_KEY and ApiSecret",
			expected: "Failed to authenticate with [REDACTED] and [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			if result != tt.expected {
				t.Errorf("SanitizeErrorMessage(%q) = %q; want %q", tt.message, result, tt.expected)
			}
		})
	}
}

func BenchmarkSanitizeErrorMessage(b *testing.B) {
	message := "Failed to use API_TOKEN and DATABASE_PASSWORD with GitHubToken"
	for i := 0; i < b.N; i++ {
		SanitizeErrorMessage(message)
	}
}
