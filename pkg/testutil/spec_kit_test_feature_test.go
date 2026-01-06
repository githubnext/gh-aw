package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsWorkflowExecuting validates that the spec-kit-execute workflow
// detection functionality works correctly.
func TestIsWorkflowExecuting(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "workflow executing returns true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWorkflowExecuting()
			assert.Equal(t, tt.expected, result, "IsWorkflowExecuting should return expected value")
		})
	}
}
