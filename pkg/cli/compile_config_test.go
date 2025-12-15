package cli

import (
	"strings"
	"testing"
)

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:             "GitHub token should be redacted",
			input:            "Error: failed with ghp_1234567890123456789012345678901234567890",
			shouldContain:    []string{"[REDACTED_GITHUB_TOKEN]"},
			shouldNotContain: []string{"ghp_123456"},
		},
		{
			name:             "Password in error message should be redacted",
			input:            "authentication failed: password=mysecretpass123",
			shouldContain:    []string{"password=[REDACTED]"},
			shouldNotContain: []string{"mysecretpass123"},
		},
		{
			name:             "Token should be redacted",
			input:            "API error: token=sk_test_1234567890abcdefghijklmnopqrst",
			shouldContain:    []string{"token=[REDACTED]"},
			shouldNotContain: []string{"sk_test_"},
		},
		{
			name:             "AWS key should be redacted",
			input:            "AWS credentials invalid: AKIAIOSFODNN7EXAMPLE",
			shouldContain:    []string{"[REDACTED_AWS_KEY]"},
			shouldNotContain: []string{"AKIAIOSF"},
		},
		{
			name:             "Normal error message should not be changed",
			input:            "file not found: workflow.md",
			shouldContain:    []string{"file not found: workflow.md"},
			shouldNotContain: []string{"[REDACTED]"},
		},
		{
			name:             "Very long message should be truncated",
			input:            strings.Repeat("a", 1200),
			shouldContain:    []string{"[truncated for security]"},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeErrorMessage(tt.input)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("sanitizeErrorMessage() should contain %q, got: %q", expected, result)
				}
			}

			for _, notExpected := range tt.shouldNotContain {
				if strings.Contains(result, notExpected) {
					t.Errorf("sanitizeErrorMessage() should NOT contain %q, got: %q", notExpected, result)
				}
			}
		})
	}
}

func TestSanitizeErrorMessage_SecretKeys(t *testing.T) {
	// Test that secret keys (the original vulnerability) are properly redacted
	input := "validation failed with secrets: MY_SECRET_KEY=abc123xyz789def456ghi, OTHER_KEY=token_value_here"
	result := sanitizeErrorMessage(input)

	// Should not contain the actual secret values
	if strings.Contains(result, "abc123xyz789def456ghi") {
		t.Errorf("Secret value was not redacted: %s", result)
	}
	if strings.Contains(result, "token_value_here") {
		t.Errorf("Token value was not redacted: %s", result)
	}

	// Should indicate that something was redacted
	if !strings.Contains(result, "[REDACTED") {
		t.Errorf("Expected redaction marker in result: %s", result)
	}
}
