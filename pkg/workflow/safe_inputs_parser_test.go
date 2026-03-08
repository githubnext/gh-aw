//go:build !integration

package workflow

import (
	"testing"
)

func TestHasSafeInputs(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeInputsConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty tools",
			config:   &SafeInputsConfig{Tools: map[string]*SafeInputToolConfig{}},
			expected: false,
		},
		{
			name: "with tools",
			config: &SafeInputsConfig{
				Tools: map[string]*SafeInputToolConfig{
					"test": {Name: "test", Description: "Test tool"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSafeInputs(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsSafeInputsEnabled(t *testing.T) {
	// Test config with tools
	configWithTools := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"test": {Name: "test", Description: "Test tool"},
		},
	}

	tests := []struct {
		name         string
		config       *SafeInputsConfig
		workflowData *WorkflowData
		expected     bool
	}{
		{
			name:         "nil config - not enabled",
			config:       nil,
			workflowData: nil,
			expected:     false,
		},
		{
			name:         "empty tools - not enabled",
			config:       &SafeInputsConfig{Tools: map[string]*SafeInputToolConfig{}},
			workflowData: nil,
			expected:     false,
		},
		{
			name:         "with tools - enabled by default",
			config:       configWithTools,
			workflowData: nil,
			expected:     true,
		},
		{
			name:   "with tools and feature flag enabled - enabled (backward compat)",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"safe-inputs": true},
			},
			expected: true,
		},
		{
			name:   "with tools and feature flag disabled - still enabled (feature flag ignored)",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"safe-inputs": false},
			},
			expected: true,
		},
		{
			name:   "with tools and other features - enabled",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"other-feature": true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeInputsEnabled(tt.config, tt.workflowData)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsSafeInputsEnabledWithEnv(t *testing.T) {
	// Test config with tools
	configWithTools := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"test": {Name: "test", Description: "Test tool"},
		},
	}

	// Safe-inputs are enabled by default when configured, environment variable no longer needed
	t.Run("with tools - enabled regardless of GH_AW_FEATURES", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "safe-inputs")
		result := IsSafeInputsEnabled(configWithTools, nil)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("with tools and GH_AW_FEATURES=other - still enabled", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "other")
		result := IsSafeInputsEnabled(configWithTools, nil)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})
}

// TestParseSafeInputsAndExtractSafeInputsConfigConsistency verifies that ParseSafeInputs
// and extractSafeInputsConfig produce identical results for the same input.
// This ensures both functions use the shared helper correctly.
