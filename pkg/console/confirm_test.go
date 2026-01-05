package console

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAccessibleMode(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "no environment variables",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name:     "ACCESSIBLE set",
			envVars:  map[string]string{"ACCESSIBLE": "1"},
			expected: true,
		},
		{
			name:     "TERM=dumb",
			envVars:  map[string]string{"TERM": "dumb"},
			expected: true,
		},
		{
			name:     "NO_COLOR set",
			envVars:  map[string]string{"NO_COLOR": "1"},
			expected: true,
		},
		{
			name:     "regular TERM",
			envVars:  map[string]string{"TERM": "xterm-256color"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}

			// Clear other relevant env vars
			for _, key := range []string{"ACCESSIBLE", "TERM", "NO_COLOR"} {
				if _, exists := tt.envVars[key]; !exists {
					os.Unsetenv(key)
				}
			}

			// Test
			result := isAccessibleMode()
			assert.Equal(t, tt.expected, result, "isAccessibleMode() = %v, want %v", result, tt.expected)

			// Restore original environment
			for key, value := range originalEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		})
	}
}

func TestConfirmAction(t *testing.T) {
	// Note: This test can't fully test the interactive behavior without mocking
	// the terminal input, but we can verify the function signature and basic setup
	
	t.Run("function signature", func(t *testing.T) {
		// This test just verifies the function exists and has the right signature
		// Actual interactive testing would require a mock terminal
		_ = ConfirmAction
	})
}
