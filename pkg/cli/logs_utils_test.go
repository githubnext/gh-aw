package cli

import (
	"os"
	"testing"

	"github.com/githubnext/gh-aw/pkg/logger"
)

func TestGetIntFromEnv(t *testing.T) {
	// Save original env value
	const testEnvVar = "GH_AW_TEST_INT_VALUE"
	originalValue := os.Getenv(testEnvVar)
	defer func() {
		if originalValue != "" {
			os.Setenv(testEnvVar, originalValue)
		} else {
			os.Unsetenv(testEnvVar)
		}
	}()

	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		minValue     int
		maxValue     int
		expected     int
	}{
		{
			name:         "default when env var not set",
			envValue:     "",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     10,
		},
		{
			name:         "valid value within range",
			envValue:     "50",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     50,
		},
		{
			name:         "valid value at minimum",
			envValue:     "1",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     1,
		},
		{
			name:         "valid value at maximum",
			envValue:     "100",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     100,
		},
		{
			name:         "invalid non-numeric value",
			envValue:     "invalid",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     10,
		},
		{
			name:         "invalid value below minimum",
			envValue:     "0",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     10,
		},
		{
			name:         "invalid negative value",
			envValue:     "-5",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     10,
		},
		{
			name:         "invalid value above maximum",
			envValue:     "101",
			defaultValue: 10,
			minValue:     1,
			maxValue:     100,
			expected:     10,
		},
		{
			name:         "different valid range",
			envValue:     "25",
			defaultValue: 5,
			minValue:     10,
			maxValue:     50,
			expected:     25,
		},
		{
			name:         "different valid range - out of bounds",
			envValue:     "5",
			defaultValue: 20,
			minValue:     10,
			maxValue:     50,
			expected:     20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv(testEnvVar, tt.envValue)
			} else {
				os.Unsetenv(testEnvVar)
			}

			// Test the function
			log := logger.New("test:getIntFromEnv")
			result := getIntFromEnv(testEnvVar, tt.defaultValue, tt.minValue, tt.maxValue, log)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetIntFromEnv_WithoutLogger(t *testing.T) {
	// Test that the function works without a logger (nil logger)
	const testEnvVar = "GH_AW_TEST_INT_NO_LOG"
	originalValue := os.Getenv(testEnvVar)
	defer func() {
		if originalValue != "" {
			os.Setenv(testEnvVar, originalValue)
		} else {
			os.Unsetenv(testEnvVar)
		}
	}()

	os.Setenv(testEnvVar, "42")
	result := getIntFromEnv(testEnvVar, 10, 1, 100, nil)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}
}
