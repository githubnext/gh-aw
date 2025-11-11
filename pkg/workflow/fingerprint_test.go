package workflow

import (
	"strings"
	"testing"
)

func TestExtractFingerprint(t *testing.T) {
tests := []struct {
name        string
frontmatter map[string]any
expected    string
shouldError bool
errorMsg    string
}{
{
name:        "Valid fingerprint with alphanumeric and hyphens",
frontmatter: map[string]any{"fingerprint": "test-fp-12345"},
expected:    "test-fp-12345",
shouldError: false,
},
{
name:        "Valid fingerprint with underscores",
frontmatter: map[string]any{"fingerprint": "test_fp_12345"},
expected:    "test_fp_12345",
shouldError: false,
},
{
name:        "Valid fingerprint exactly 8 characters",
frontmatter: map[string]any{"fingerprint": "12345678"},
expected:    "12345678",
shouldError: false,
},
{
name:        "Valid fingerprint with mixed case",
frontmatter: map[string]any{"fingerprint": "TestFP_123"},
expected:    "TestFP_123",
shouldError: false,
},
{
name:        "Missing fingerprint returns empty string",
frontmatter: map[string]any{},
expected:    "",
shouldError: false,
},
{
name:        "Fingerprint with leading/trailing spaces trimmed",
frontmatter: map[string]any{"fingerprint": "  test-fp-12345  "},
expected:    "test-fp-12345",
shouldError: false,
},
{
name:        "Fingerprint too short (7 chars)",
frontmatter: map[string]any{"fingerprint": "1234567"},
expected:    "",
shouldError: true,
errorMsg:    "fingerprint must be at least 8 characters long",
},
{
name:        "Fingerprint with invalid character (@)",
frontmatter: map[string]any{"fingerprint": "test@fp123"},
expected:    "",
shouldError: true,
errorMsg:    "fingerprint contains invalid character",
},
{
name:        "Fingerprint with invalid character (space)",
frontmatter: map[string]any{"fingerprint": "test fp 123"},
expected:    "",
shouldError: true,
errorMsg:    "fingerprint contains invalid character",
},
{
name:        "Fingerprint with invalid character (.)",
frontmatter: map[string]any{"fingerprint": "test.fp.123"},
expected:    "",
shouldError: true,
errorMsg:    "fingerprint contains invalid character",
},
{
name:        "Fingerprint not a string",
frontmatter: map[string]any{"fingerprint": 12345678},
expected:    "",
shouldError: true,
errorMsg:    "fingerprint must be a string",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
compiler := &Compiler{}
result, err := compiler.extractFingerprint(tt.frontmatter)

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
