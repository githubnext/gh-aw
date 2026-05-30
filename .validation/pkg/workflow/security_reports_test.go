//go:build !integration

package workflow

import (
	"testing"
)

// TestCodeScanningAlertsConfig tests the parsing of create-code-scanning-alert configuration
func TestCodeScanningAlertsConfig(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		frontmatter    map[string]any
		expectedConfig *CreateCodeScanningAlertsConfig
	}{
		{
			name: "basic code scanning alert configuration",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-code-scanning-alert": nil,
				},
			},
			expectedConfig: &CreateCodeScanningAlertsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: nil}}, // 0 means unlimited
		},
		{
			name: "code scanning alert with max configuration",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-code-scanning-alert": map[string]any{
						"max": 50,
					},
				},
			},
			expectedConfig: &CreateCodeScanningAlertsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("50")}},
		},
		{
			name: "code scanning alert with driver configuration",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-code-scanning-alert": map[string]any{
						"driver": "Custom Security Scanner",
					},
				},
			},
			expectedConfig: &CreateCodeScanningAlertsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: nil}, Driver: "Custom Security Scanner"},
		},
		{
			name: "code scanning alert with max and driver configuration",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-code-scanning-alert": map[string]any{
						"max":    25,
						"driver": "Advanced Scanner",
					},
				},
			},
			expectedConfig: &CreateCodeScanningAlertsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("25")}, Driver: "Advanced Scanner"},
		},
		{
			name: "no code scanning alert configuration",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": nil,
				},
			},
			expectedConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)

			if tt.expectedConfig == nil {
				if config == nil || config.CreateCodeScanningAlerts == nil {
					return // Expected no config
				}
				t.Errorf("Expected no CreateCodeScanningAlerts config, but got: %+v", config.CreateCodeScanningAlerts)
				return
			}

			if config == nil || config.CreateCodeScanningAlerts == nil {
				t.Errorf("Expected CreateCodeScanningAlerts config, but got nil")
				return
			}

			if (config.CreateCodeScanningAlerts.Max == nil) != (tt.expectedConfig.Max == nil) ||
				(config.CreateCodeScanningAlerts.Max != nil && *config.CreateCodeScanningAlerts.Max != *tt.expectedConfig.Max) {
				t.Errorf("Expected Max=%v, got Max=%v", tt.expectedConfig.Max, config.CreateCodeScanningAlerts.Max)
			}

			if config.CreateCodeScanningAlerts.Driver != tt.expectedConfig.Driver {
				t.Errorf("Expected Driver=%s, got Driver=%s", tt.expectedConfig.Driver, config.CreateCodeScanningAlerts.Driver)
			}
		})
	}
}

// TestParseCodeScanningAlertsConfig tests the parsing function directly
func TestParseCodeScanningAlertsConfig(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		outputMap      map[string]any
		expectedMax    *string
		expectedDriver string
		expectNil      bool
	}{
		{
			name: "basic configuration",
			outputMap: map[string]any{
				"create-code-scanning-alert": nil,
			},
			expectedMax:    nil,
			expectedDriver: "",
			expectNil:      false,
		},
		{
			name: "configuration with max",
			outputMap: map[string]any{
				"create-code-scanning-alert": map[string]any{
					"max": 100,
				},
			},
			expectedMax:    strPtr("100"),
			expectedDriver: "",
			expectNil:      false,
		},
		{
			name: "configuration with driver",
			outputMap: map[string]any{
				"create-code-scanning-alert": map[string]any{
					"driver": "Test Security Scanner",
				},
			},
			expectedMax:    nil,
			expectedDriver: "Test Security Scanner",
			expectNil:      false,
		},
		{
			name: "configuration with max and driver",
			outputMap: map[string]any{
				"create-code-scanning-alert": map[string]any{
					"max":    50,
					"driver": "Combined Scanner",
				},
			},
			expectedMax:    strPtr("50"),
			expectedDriver: "Combined Scanner",
			expectNil:      false,
		},
		{
			name: "no configuration",
			outputMap: map[string]any{
				"other-config": nil,
			},
			expectedMax:    nil,
			expectedDriver: "",
			expectNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.parseCodeScanningAlertsConfig(tt.outputMap)

			if tt.expectNil {
				if config != nil {
					t.Errorf("Expected nil config, got: %+v", config)
				}
				return
			}

			if config == nil {
				t.Errorf("Expected config, got nil")
				return
			}

			if (config.Max == nil) != (tt.expectedMax == nil) ||
				(config.Max != nil && *config.Max != *tt.expectedMax) {
				t.Errorf("Expected Max=%v, got Max=%v", tt.expectedMax, config.Max)
			}

			if config.Driver != tt.expectedDriver {
				t.Errorf("Expected Driver=%s, got Driver=%s", tt.expectedDriver, config.Driver)
			}
		})
	}
}
