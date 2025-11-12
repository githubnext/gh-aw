package workflow

import (
	"strings"
	"testing"
)

func TestExtractCampaign(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid campaign with alphanumeric and hyphens",
			frontmatter: map[string]any{"campaign": "test-fp-12345"},
			expected:    "test-fp-12345",
			shouldError: false,
		},
		{
			name:        "Valid campaign with underscores",
			frontmatter: map[string]any{"campaign": "test_fp_12345"},
			expected:    "test_fp_12345",
			shouldError: false,
		},
		{
			name:        "Valid campaign exactly 8 characters",
			frontmatter: map[string]any{"campaign": "12345678"},
			expected:    "12345678",
			shouldError: false,
		},
		{
			name:        "Valid campaign with mixed case",
			frontmatter: map[string]any{"campaign": "TestFP_123"},
			expected:    "TestFP_123",
			shouldError: false,
		},
		{
			name:        "Missing campaign returns empty string",
			frontmatter: map[string]any{},
			expected:    "",
			shouldError: false,
		},
		{
			name:        "Campaign with leading/trailing spaces trimmed",
			frontmatter: map[string]any{"campaign": "  test-fp-12345  "},
			expected:    "test-fp-12345",
			shouldError: false,
		},
		{
			name:        "Campaign too short (7 chars)",
			frontmatter: map[string]any{"campaign": "1234567"},
			expected:    "",
			shouldError: true,
			errorMsg:    "campaign must be at least 8 characters long",
		},
		{
			name:        "Campaign with invalid character (@)",
			frontmatter: map[string]any{"campaign": "test@fp123"},
			expected:    "",
			shouldError: true,
			errorMsg:    "campaign contains invalid character",
		},
		{
			name:        "Campaign with invalid character (space)",
			frontmatter: map[string]any{"campaign": "test fp 123"},
			expected:    "",
			shouldError: true,
			errorMsg:    "campaign contains invalid character",
		},
		{
			name:        "Campaign with invalid character (.)",
			frontmatter: map[string]any{"campaign": "test.fp.123"},
			expected:    "",
			shouldError: true,
			errorMsg:    "campaign contains invalid character",
		},
		{
			name:        "Campaign not a string",
			frontmatter: map[string]any{"campaign": 12345678},
			expected:    "",
			shouldError: true,
			errorMsg:    "campaign must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			result, err := compiler.extractCampaign(tt.frontmatter)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}
