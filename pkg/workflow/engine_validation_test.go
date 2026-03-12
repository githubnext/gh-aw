//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestValidateSingleEngineSpecification tests the validateSingleEngineSpecification function
func TestValidateSingleEngineSpecification(t *testing.T) {
	tests := []struct {
		name                string
		mainEngineSetting   string
		includedEnginesJSON []string
		expectedEngine      string
		expectError         bool
		errorMsg            string
	}{
		{
			name:                "no engine specified anywhere",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{},
			expectedEngine:      "",
			expectError:         false,
		},
		{
			name:                "engine only in main workflow",
			mainEngineSetting:   "copilot",
			includedEnginesJSON: []string{},
			expectedEngine:      "copilot",
			expectError:         false,
		},
		{
			name:                "engine only in included file (string format)",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{`"claude"`},
			expectedEngine:      "claude",
			expectError:         false,
		},
		{
			name:                "engine only in included file (object format)",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{`{"id": "codex", "model": "gpt-4"}`},
			expectedEngine:      "codex",
			expectError:         false,
		},
		{
			name:                "multiple engines in main and included",
			mainEngineSetting:   "copilot",
			includedEnginesJSON: []string{`"claude"`},
			expectedEngine:      "",
			expectError:         true,
			errorMsg:            "multiple engine fields found",
		},
		{
			name:                "multiple engines in different included files",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{`"copilot"`, `"claude"`},
			expectedEngine:      "",
			expectError:         true,
			errorMsg:            "multiple engine fields found",
		},
		{
			name:                "empty string in main engine setting",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{},
			expectedEngine:      "",
			expectError:         false,
		},
		{
			name:                "empty strings in included engines are ignored",
			mainEngineSetting:   "copilot",
			includedEnginesJSON: []string{"", ""},
			expectedEngine:      "copilot",
			expectError:         false,
		},
		{
			name:                "invalid JSON in included engine",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{`{invalid json}`},
			expectedEngine:      "",
			expectError:         true,
			errorMsg:            "failed to parse",
		},
		{
			name:                "included engine with invalid object format (no id)",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{`{"model": "gpt-4"}`},
			expectedEngine:      "",
			expectError:         true,
			errorMsg:            "invalid engine configuration",
		},
		{
			name:                "included engine with non-string id",
			mainEngineSetting:   "",
			includedEnginesJSON: []string{`{"id": 123}`},
			expectedEngine:      "",
			expectError:         true,
			errorMsg:            "invalid engine configuration",
		},
		{
			name:                "main engine takes precedence when only non-empty",
			mainEngineSetting:   "codex",
			includedEnginesJSON: []string{""},
			expectedEngine:      "codex",
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			result, err := compiler.validateSingleEngineSpecification(tt.mainEngineSetting, tt.includedEnginesJSON)

			if tt.expectError && err == nil {
				t.Error("Expected validation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected validation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}

			if !tt.expectError && result != tt.expectedEngine {
				t.Errorf("Expected engine %q, got %q", tt.expectedEngine, result)
			}
		})
	}
}

// TestValidateSingleEngineSpecificationErrorMessageQuality verifies error messages follow the style guide
func TestValidateSingleEngineSpecificationErrorMessageQuality(t *testing.T) {
	compiler := NewCompiler()

	t.Run("multiple engines error includes example", func(t *testing.T) {
		_, err := compiler.validateSingleEngineSpecification("copilot", []string{`"claude"`})

		if err == nil {
			t.Fatal("Expected validation to fail for multiple engines")
		}

		errorMsg := err.Error()

		// Error should explain what's wrong
		if !strings.Contains(errorMsg, "multiple engine fields found") {
			t.Errorf("Error should explain multiple engines found, got: %s", errorMsg)
		}

		// Error should include count of specifications
		if !strings.Contains(errorMsg, "2 engine specifications") {
			t.Errorf("Error should include count of engine specifications, got: %s", errorMsg)
		}

		// Error should include example
		if !strings.Contains(errorMsg, "Example:") && !strings.Contains(errorMsg, "engine: copilot") {
			t.Errorf("Error should include an example, got: %s", errorMsg)
		}
	})

	t.Run("parse error includes format examples", func(t *testing.T) {
		_, err := compiler.validateSingleEngineSpecification("", []string{`{invalid json}`})

		if err == nil {
			t.Fatal("Expected validation to fail for invalid JSON")
		}

		errorMsg := err.Error()

		// Error should mention parse failure
		if !strings.Contains(errorMsg, "failed to parse") {
			t.Errorf("Error should mention parse failure, got: %s", errorMsg)
		}

		// Error should show both string and object format examples
		if !strings.Contains(errorMsg, "engine: copilot") {
			t.Errorf("Error should include string format example, got: %s", errorMsg)
		}

		if !strings.Contains(errorMsg, "id: copilot") {
			t.Errorf("Error should include object format example, got: %s", errorMsg)
		}
	})

	t.Run("invalid configuration error includes format examples", func(t *testing.T) {
		_, err := compiler.validateSingleEngineSpecification("", []string{`{"model": "gpt-4"}`})

		if err == nil {
			t.Fatal("Expected validation to fail for configuration without id")
		}

		errorMsg := err.Error()

		// Error should explain the problem
		if !strings.Contains(errorMsg, "invalid engine configuration") {
			t.Errorf("Error should explain invalid configuration, got: %s", errorMsg)
		}

		// Error should mention missing 'id' field
		if !strings.Contains(errorMsg, "id") {
			t.Errorf("Error should mention 'id' field, got: %s", errorMsg)
		}

		// Error should show both string and object format examples
		if !strings.Contains(errorMsg, "engine: copilot") {
			t.Errorf("Error should include string format example, got: %s", errorMsg)
		}

		if !strings.Contains(errorMsg, "id: copilot") {
			t.Errorf("Error should include object format example, got: %s", errorMsg)
		}
	})
}

// TestValidatePluginSupport tests the validatePluginSupport function
func TestValidatePluginSupport(t *testing.T) {
	tests := []struct {
		name        string
		pluginInfo  *PluginInfo
		engineID    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no plugins with copilot engine",
			pluginInfo:  nil,
			engineID:    "copilot",
			expectError: false,
		},
		{
			name: "plugins with copilot engine (supported)",
			pluginInfo: &PluginInfo{
				Plugins: []string{"org/plugin1", "org/plugin2"},
			},
			engineID:    "copilot",
			expectError: false,
		},
		{
			name: "plugins with claude engine (not supported)",
			pluginInfo: &PluginInfo{
				Plugins: []string{"org/plugin1"},
			},
			engineID:    "claude",
			expectError: true,
			errorMsg:    "does not support plugins",
		},
		{
			name: "plugins with codex engine (not supported)",
			pluginInfo: &PluginInfo{
				Plugins: []string{"org/plugin1", "org/plugin2"},
			},
			engineID:    "codex",
			expectError: true,
			errorMsg:    "does not support plugins",
		},
		{
			name:        "empty plugin list",
			pluginInfo:  &PluginInfo{Plugins: []string{}},
			engineID:    "claude",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			engine, err := compiler.engineRegistry.GetEngine(tt.engineID)
			if err != nil {
				t.Fatalf("Failed to get engine: %v", err)
			}

			err = compiler.validatePluginSupport(tt.pluginInfo, engine)

			if tt.expectError && err == nil {
				t.Error("Expected validation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected validation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

// TestValidatePluginSupportErrorMessage verifies the plugin validation error message quality
func TestValidatePluginSupportErrorMessage(t *testing.T) {
	compiler := NewCompiler()
	claudeEngine, err := compiler.engineRegistry.GetEngine("claude")
	if err != nil {
		t.Fatalf("Failed to get claude engine: %v", err)
	}

	pluginInfo := &PluginInfo{
		Plugins: []string{"org/plugin1", "org/plugin2"},
	}

	err = compiler.validatePluginSupport(pluginInfo, claudeEngine)
	if err == nil {
		t.Fatal("Expected validation to fail for plugins with claude engine")
	}

	errorMsg := err.Error()

	// Error should mention the engine name
	if !strings.Contains(errorMsg, "claude") {
		t.Errorf("Error message should mention the engine name, got: %s", errorMsg)
	}

	// Error should list the plugins that can't be installed
	if !strings.Contains(errorMsg, "org/plugin1") || !strings.Contains(errorMsg, "org/plugin2") {
		t.Errorf("Error message should list the plugins, got: %s", errorMsg)
	}

	// Error should mention copilot as a supported engine (since it's the only one that supports plugins)
	if !strings.Contains(errorMsg, "copilot") {
		t.Errorf("Error message should mention copilot as supported engine, got: %s", errorMsg)
	}

	// Error should provide actionable fixes
	if !strings.Contains(errorMsg, "To fix this") {
		t.Errorf("Error message should provide actionable fixes, got: %s", errorMsg)
	}
}
