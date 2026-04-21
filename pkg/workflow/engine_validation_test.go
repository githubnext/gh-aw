//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestValidateEngineVersion tests the validateEngineVersion function
func TestValidateEngineVersion(t *testing.T) {
	tests := []struct {
		name        string
		engineCfg   *EngineConfig
		strictMode  bool
		expectWarn  bool
		expectError bool
	}{
		{
			name:        "no engine config",
			engineCfg:   nil,
			expectWarn:  false,
			expectError: false,
		},
		{
			name:        "empty version",
			engineCfg:   &EngineConfig{Version: ""},
			expectWarn:  false,
			expectError: false,
		},
		{
			name:        "pinned version",
			engineCfg:   &EngineConfig{Version: "2.1.92"},
			expectWarn:  false,
			expectError: false,
		},
		{
			name:        "latest version non-strict",
			engineCfg:   &EngineConfig{Version: "latest"},
			strictMode:  false,
			expectWarn:  true,
			expectError: false,
		},
		{
			name:        "LATEST uppercase non-strict",
			engineCfg:   &EngineConfig{Version: "LATEST"},
			strictMode:  false,
			expectWarn:  true,
			expectError: false,
		},
		{
			name:        "latest version strict mode",
			engineCfg:   &EngineConfig{Version: "latest"},
			strictMode:  true,
			expectWarn:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			workflowData := &WorkflowData{
				EngineConfig: tt.engineCfg,
			}

			err := compiler.validateEngineVersion(workflowData)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), "strict mode") {
					t.Errorf("Expected strict mode error, got: %s", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateEngineDriverScript(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *WorkflowData
		expectError bool
		errorSubstr string
	}{
		{
			name: "no engine config",
			workflow: &WorkflowData{
				EngineConfig: nil,
			},
			expectError: false,
		},
		{
			name: "no driver configured",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
			},
			expectError: false,
		},
		{
			name: "valid cjs driver on copilot",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "custom_driver.cjs"},
			},
			expectError: false,
		},
		{
			name: "valid mjs driver on copilot",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "custom_driver.mjs"},
			},
			expectError: false,
		},
		{
			name: "invalid extension",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "driver.sh"},
			},
			expectError: true,
			errorSubstr: "must be a Node.js script",
		},
		{
			name: "driver configured for any engine",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "claude", DriverScript: "driver.cjs"},
			},
			expectError: false,
		},
		{
			name: "invalid path traversal",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "../driver.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
		{
			name: "invalid path separator",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "nested/driver.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
		{
			name: "invalid shell metacharacter",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "driver;rm -rf /.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
		{
			name: "invalid leading whitespace",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: " driver.cjs"},
			},
			expectError: true,
			errorSubstr: "leading/trailing whitespace",
		},
		{
			name: "invalid leading hyphen",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", DriverScript: "-driver.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			err := compiler.validateEngineDriverScript(tt.workflow)

			if tt.expectError {
				require.Error(t, err, "Expected validation error")
				if tt.errorSubstr != "" {
					assert.Contains(t, err.Error(), tt.errorSubstr, "Expected error substring mismatch")
				}
				return
			}

			assert.NoError(t, err, "Expected driver validation to pass")
		})
	}
}
