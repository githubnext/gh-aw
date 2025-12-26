package stringutil

import (
	"strings"
	"testing"
)

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

// Additional edge case tests

func TestSanitizeErrorMessage_AllWorkflowKeywords(t *testing.T) {
	// Test all common workflow keywords that should NOT be redacted
	keywords := []string{
		"GITHUB", "ACTIONS", "WORKFLOW", "RUNNER", "JOB", "STEP",
		"MATRIX", "ENV", "PATH", "HOME", "SHELL", "INPUTS", "OUTPUTS",
		"NEEDS", "STRATEGY", "CONCURRENCY", "IF", "WITH", "USES", "RUN",
		"WORKING_DIRECTORY", "CONTINUE_ON_ERROR", "TIMEOUT_MINUTES",
	}

	for _, keyword := range keywords {
		message := "Error with " + keyword + " configuration"
		result := SanitizeErrorMessage(message)
		if !strings.Contains(result, keyword) {
			t.Errorf("Workflow keyword %q should not be redacted, got: %q", keyword, result)
		}
	}
}

func TestSanitizeErrorMessage_MultipleOccurrences(t *testing.T) {
	message := "MY_SECRET is used twice: MY_SECRET here and MY_SECRET there"
	result := SanitizeErrorMessage(message)
	expected := "[REDACTED] is used twice: [REDACTED] here and [REDACTED] there"

	if result != expected {
		t.Errorf("SanitizeErrorMessage(%q) = %q; want %q", message, result, expected)
	}
}

func TestSanitizeErrorMessage_MixedCase(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "lowercase not matched",
			message:  "error with my_secret_key",
			expected: "error with my_secret_key",
		},
		{
			name:     "mixed case not matched",
			message:  "error with My_Secret_Key",
			expected: "error with My_Secret_Key",
		},
		{
			name:     "all uppercase matched",
			message:  "error with MY_SECRET_KEY",
			expected: "error with [REDACTED]",
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

func TestSanitizeErrorMessage_PascalCaseVariants(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		shouldRedact bool
	}{
		{"Token suffix", "Invalid GitHubToken", true},
		{"Key suffix", "Missing ApiKey", true},
		{"Secret suffix", "Bad DeploySecret", true},
		{"Password suffix", "Wrong DatabasePassword", true},
		{"Credential suffix", "Invalid AwsCredential", true},
		{"Auth suffix", "Failed BasicAuth", true},
		{"No suffix", "Invalid GitHubActions", false},
		{"lowercase", "Invalid githubtoken", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			containsRedacted := strings.Contains(result, "[REDACTED]")

			if tt.shouldRedact && !containsRedacted {
				t.Errorf("Expected message to be redacted: %q", tt.message)
			}
			if !tt.shouldRedact && containsRedacted {
				t.Errorf("Expected message NOT to be redacted: %q", tt.message)
			}
		})
	}
}

func TestSanitizeErrorMessage_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "very long message",
			message:  "Error: " + strings.Repeat("MY_SECRET_KEY ", 100),
			expected: "Error: " + strings.Repeat("[REDACTED] ", 100),
		},
		{
			name:     "only secrets",
			message:  "API_KEY DATABASE_PASSWORD GitHubToken",
			expected: "[REDACTED] [REDACTED] [REDACTED]",
		},
		{
			name:     "secrets at start and end",
			message:  "MY_API_KEY in the middle DATABASE_SECRET",
			expected: "[REDACTED] in the middle [REDACTED]",
		},
		{
			name:     "secret with numbers",
			message:  "Error with API_KEY_V2 and SECRET_123",
			expected: "Error with [REDACTED] and [REDACTED]",
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

func TestSanitizeErrorMessage_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "GitHub Actions error",
			message:  "Failed to authenticate: GITHUB_TOKEN is invalid",
			expected: "Failed to authenticate: [REDACTED] is invalid",
		},
		{
			name:     "AWS credentials error",
			message:  "AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are required",
			expected: "[REDACTED] and [REDACTED] are required",
		},
		{
			name:     "Database connection error",
			message:  "Could not connect using DB_PASSWORD: connection refused",
			expected: "Could not connect using [REDACTED]: connection refused",
		},
		{
			name:     "API error with token",
			message:  "Request failed with ApiToken: 401 Unauthorized",
			expected: "Request failed with [REDACTED]: 401 Unauthorized",
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

func BenchmarkSanitizeErrorMessage_NoSecrets(b *testing.B) {
	message := "This is a regular error message with no secrets to redact"
	for i := 0; i < b.N; i++ {
		SanitizeErrorMessage(message)
	}
}

func BenchmarkSanitizeErrorMessage_ManySecrets(b *testing.B) {
	message := "Error with API_KEY, DATABASE_PASSWORD, AWS_SECRET, GitHubToken, and DeploySecret"
	for i := 0; i < b.N; i++ {
		SanitizeErrorMessage(message)
	}
}
