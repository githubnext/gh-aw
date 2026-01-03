package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProcessTestFeature tests the basic test feature functionality
func TestProcessTestFeature(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple message",
			input:    "hello",
			expected: "Test feature: hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "Test feature: ",
		},
		{
			name:     "special characters",
			input:    "test-123!@#",
			expected: "Test feature: test-123!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessTestFeature(tt.input)
			assert.Equal(t, tt.expected, result, "ProcessTestFeature should format input correctly")
		})
	}
}

// TestProcessTestFeatureNilSafety tests that the function handles edge cases
func TestProcessTestFeatureNilSafety(t *testing.T) {
	// Test that function works with empty input
	result := ProcessTestFeature("")
	assert.NotEmpty(t, result, "ProcessTestFeature should return non-empty string")
	assert.Equal(t, "Test feature: ", result)
}
