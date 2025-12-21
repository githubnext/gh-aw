package workflow

import (
	"strings"
	"testing"
)

// TestValidateEngine tests the validateEngine function
func TestValidateEngine(t *testing.T) {
	tests := []struct {
		name        string
		engineID    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty engine ID is valid (uses default)",
			engineID:    "",
			expectError: false,
		},
		{
			name:        "copilot engine is valid",
			engineID:    "copilot",
			expectError: false,
		},
		{
			name:        "claude engine is valid",
			engineID:    "claude",
			expectError: false,
		},
		{
			name:        "codex engine is valid",
			engineID:    "codex",
			expectError: false,
		},
		{
			name:        "custom engine is valid",
			engineID:    "custom",
			expectError: false,
		},
		{
			name:        "invalid engine ID",
			engineID:    "invalid-engine",
			expectError: true,
			errorMsg:    "invalid engine",
		},
		{
			name:        "unknown engine ID",
			engineID:    "gpt-7",
			expectError: true,
			errorMsg:    "invalid engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "")
			err := compiler.validateEngine(tt.engineID)

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

// TestValidateEngineErrorMessageQuality verifies that error messages follow the style guide
func TestValidateEngineErrorMessageQuality(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	err := compiler.validateEngine("invalid-engine")

	if err == nil {
		t.Fatal("Expected validation to fail for invalid engine")
	}

	errorMsg := err.Error()

	// Error should list all valid engine options
	if !strings.Contains(errorMsg, "copilot") || !strings.Contains(errorMsg, "claude") ||
		!strings.Contains(errorMsg, "codex") || !strings.Contains(errorMsg, "custom") {
		t.Errorf("Error message should list all valid engines (copilot, claude, codex, custom), got: %s", errorMsg)
	}

	// Error should include an example
	if !strings.Contains(errorMsg, "Example:") && !strings.Contains(errorMsg, "engine: copilot") {
		t.Errorf("Error message should include an example, got: %s", errorMsg)
	}
}

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
			mainEngineSetting:   "custom",
			includedEnginesJSON: []string{""},
			expectedEngine:      "custom",
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "")
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
	compiler := NewCompiler(false, "", "")

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

// TestValidateCopilotNetworkConfig tests the validateCopilotNetworkConfig function
func TestValidateCopilotNetworkConfig(t *testing.T) {
	tests := []struct {
		name               string
		engineID           string
		networkPermissions *NetworkPermissions
		tools              *Tools
		expectError        bool
		errorMsg           string
	}{
		{
			name:     "non-Copilot engine is allowed api.github.com",
			engineID: "claude",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"api.github.com", "anthropic.com"},
			},
			tools:       nil,
			expectError: false,
		},
		{
			name:     "Copilot engine without api.github.com in network allowed",
			engineID: "copilot",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"example.com", "trusted.org"},
			},
			tools:       nil,
			expectError: false,
		},
		{
			name:     "Copilot engine with api.github.com but has GitHub MCP configured",
			engineID: "copilot",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"api.github.com", "example.com"},
			},
			tools: NewTools(map[string]any{
				"github": map[string]any{
					"mode":     "remote",
					"toolsets": []any{"default"},
				},
			}),
			expectError: true,
			errorMsg:    "cannot directly access api.github.com",
		},
		{
			name:     "Copilot engine with api.github.com and no GitHub MCP configured",
			engineID: "copilot",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"api.github.com", "example.com"},
			},
			tools:       NewTools(map[string]any{}),
			expectError: true,
			errorMsg:    "cannot directly access api.github.com",
		},
		{
			name:               "Copilot engine with empty network permissions",
			engineID:           "copilot",
			networkPermissions: &NetworkPermissions{Mode: "defaults"},
			tools:              nil,
			expectError:        false,
		},
		{
			name:               "Copilot engine with nil network permissions",
			engineID:           "copilot",
			networkPermissions: nil,
			tools:              nil,
			expectError:        false,
		},
		{
			name:     "Copilot engine with api.github.com among other domains",
			engineID: "copilot",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"example.com", "api.github.com", "trusted.org"},
			},
			tools:       nil,
			expectError: true,
			errorMsg:    "cannot directly access api.github.com",
		},
		{
			name:     "Copilot engine with only api.github.com in allowed list",
			engineID: "copilot",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"api.github.com"},
			},
			tools:       nil,
			expectError: true,
			errorMsg:    "cannot directly access api.github.com",
		},
		{
			name:     "codex engine is allowed api.github.com",
			engineID: "codex",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"api.github.com", "api.openai.com"},
			},
			tools:       nil,
			expectError: false,
		},
		{
			name:     "custom engine is allowed api.github.com",
			engineID: "custom",
			networkPermissions: &NetworkPermissions{
				Mode:    "custom",
				Allowed: []string{"api.github.com"},
			},
			tools:       nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "")
			err := compiler.validateCopilotNetworkConfig(tt.engineID, tt.networkPermissions, tt.tools)

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

// TestValidateCopilotNetworkConfigErrorMessageQuality verifies error messages are helpful
func TestValidateCopilotNetworkConfigErrorMessageQuality(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	t.Run("error message suggests GitHub MCP configuration when not present", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Mode:    "custom",
			Allowed: []string{"api.github.com"},
		}
		tools := NewTools(map[string]any{})

		err := compiler.validateCopilotNetworkConfig("copilot", networkPermissions, tools)

		if err == nil {
			t.Fatal("Expected validation to fail for Copilot with api.github.com")
		}

		errorMsg := err.Error()

		// Error should explain the problem
		if !strings.Contains(errorMsg, "cannot directly access api.github.com") {
			t.Errorf("Error should explain api.github.com cannot be accessed directly, got: %s", errorMsg)
		}

		// Error should mention GitHub MCP server requirement
		if !strings.Contains(errorMsg, "GitHub MCP server") {
			t.Errorf("Error should mention GitHub MCP server, got: %s", errorMsg)
		}

		// Error should include remote mode example
		if !strings.Contains(errorMsg, "mode: remote") {
			t.Errorf("Error should include remote mode example, got: %s", errorMsg)
		}

		// Error should include local mode example
		if !strings.Contains(errorMsg, "mode: local") {
			t.Errorf("Error should include local mode example, got: %s", errorMsg)
		}

		// Error should suggest removing api.github.com
		if !strings.Contains(errorMsg, "remove 'api.github.com'") {
			t.Errorf("Error should suggest removing api.github.com, got: %s", errorMsg)
		}

		// Error should include documentation link
		if !strings.Contains(errorMsg, "https://githubnext.github.io/gh-aw") {
			t.Errorf("Error should include documentation link, got: %s", errorMsg)
		}
	})

	t.Run("error message mentions existing GitHub MCP config when present", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Mode:    "custom",
			Allowed: []string{"api.github.com", "example.com"},
		}
		tools := NewTools(map[string]any{
			"github": map[string]any{
				"mode":     "remote",
				"toolsets": []any{"default"},
			},
		})

		err := compiler.validateCopilotNetworkConfig("copilot", networkPermissions, tools)

		if err == nil {
			t.Fatal("Expected validation to fail for Copilot with api.github.com")
		}

		errorMsg := err.Error()

		// Error should mention that GitHub MCP is already configured
		if !strings.Contains(errorMsg, "GitHub MCP configured") {
			t.Errorf("Error should mention GitHub MCP is configured, got: %s", errorMsg)
		}

		// Error should suggest removing api.github.com from network list
		if !strings.Contains(errorMsg, "remove 'api.github.com'") {
			t.Errorf("Error should suggest removing api.github.com from network list, got: %s", errorMsg)
		}
	})
}
