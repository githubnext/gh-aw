//go:build !integration

package workflow

import (
	"encoding/json"
	"reflect"
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

func TestMCPScriptArrayItems(t *testing.T) {
	compiler := NewCompiler()
	for _, itemType := range []MCPParamType{
		MCPParamTypeString, MCPParamTypeNumber, MCPParamTypeBoolean, MCPParamTypeArray, MCPParamTypeObject,
	} {
		t.Run(string(itemType), func(t *testing.T) {
			toolMap := map[string]any{
				"description": "Array input",
				"script":      "return args;",
				"inputs": map[string]any{
					"args": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": string(itemType)},
					},
					"untyped": map[string]any{"type": "array"},
				},
			}
			tool := parseMCPScriptToolConfig("example", toolMap)
			want := &MCPScriptParamItems{Type: itemType}
			if !reflect.DeepEqual(tool.Inputs["args"].Items, want) {
				t.Fatalf("parsed items = %#v, want %#v", tool.Inputs["args"].Items, want)
			}
			if tool.Inputs["untyped"].Items != nil {
				t.Fatal("array without items should remain unconstrained")
			}
			importJSON, err := json.Marshal(map[string]any{"example": toolMap})
			if err != nil {
				t.Fatal(err)
			}
			imported := compiler.mergeMCPScripts(nil, []string{string(importJSON)})
			if !reflect.DeepEqual(imported.Tools["example"].Inputs["args"].Items, want) {
				t.Fatal("imported item type was not preserved")
			}
			var generated MCPScriptsConfigJSON
			if err := json.Unmarshal([]byte(GenerateMCPScriptsToolsConfig(imported)), &generated); err != nil {
				t.Fatal(err)
			}
			props := generated.Tools[0].InputSchema["properties"].(map[string]any)
			items := props["args"].(map[string]any)["items"]
			if !reflect.DeepEqual(items, map[string]any{"type": string(itemType)}) {
				t.Fatalf("generated items = %#v", items)
			}
			if _, exists := props["untyped"].(map[string]any)["items"]; exists {
				t.Fatal("array without items should not emit an items schema")
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
		name     string
		config   *MCPScriptsConfig
		expected bool
	}{
		{
			name:     "nil config - not enabled",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty tools - not enabled",
			config:   &MCPScriptsConfig{Tools: map[string]*MCPScriptToolConfig{}},
			expected: false,
		},
		{
			name:     "with tools - enabled by default",
			config:   configWithTools,
			expected: true,
		},
		{
			name:     "with tools and feature flag enabled - enabled (backward compat)",
			config:   configWithTools,
			expected: true,
		},
		{
			name:     "with tools and feature flag disabled - still enabled (feature flag ignored)",
			config:   configWithTools,
			expected: true,
		},
		{
			name:     "with tools and other features - enabled",
			config:   configWithTools,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMCPScriptsEnabled(tt.config)
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
		result := IsMCPScriptsEnabled(configWithTools)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("with tools and GH_AW_FEATURES=other - still enabled", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "other")
		result := IsMCPScriptsEnabled(configWithTools)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})
}

// TestParseMCPScriptsAndExtractMCPScriptsConfigConsistency verifies that ParseMCPScripts
// and extractMCPScriptsConfig produce identical results for the same input.
// This ensures both functions use the shared helper correctly.

func TestParseMCPScriptToolConfigDependencies(t *testing.T) {
	t.Run("single dependency", func(t *testing.T) {
		tool := parseMCPScriptToolConfig("fetch", map[string]any{
			"description":  "fetch data",
			"py":           "print('ok')",
			"dependencies": []any{"requests"},
		})
		if !reflect.DeepEqual(tool.Dependencies, []string{"requests"}) {
			t.Fatalf("expected dependencies [requests], got %v", tool.Dependencies)
		}
	})

	t.Run("multiple dependencies", func(t *testing.T) {
		tool := parseMCPScriptToolConfig("analyze", map[string]any{
			"description":  "analyze",
			"script":       "return { ok: true }",
			"dependencies": []any{"lodash", "zod"},
		})
		if !reflect.DeepEqual(tool.Dependencies, []string{"lodash", "zod"}) {
			t.Fatalf("expected dependencies [lodash zod], got %v", tool.Dependencies)
		}
	})

	t.Run("empty dependencies", func(t *testing.T) {
		tool := parseMCPScriptToolConfig("noop", map[string]any{
			"description":  "noop",
			"run":          "echo ok",
			"dependencies": []any{},
		})
		if len(tool.Dependencies) != 0 {
			t.Fatalf("expected no dependencies, got %v", tool.Dependencies)
		}
	})

	t.Run("non-string dependencies are skipped", func(t *testing.T) {
		tool := parseMCPScriptToolConfig("mixed", map[string]any{
			"description":  "mixed deps",
			"py":           "print('ok')",
			"dependencies": []any{"requests", 123, true, "urllib3"},
		})
		if !reflect.DeepEqual(tool.Dependencies, []string{"requests", "urllib3"}) {
			t.Fatalf("expected string dependencies only, got %v", tool.Dependencies)
		}
	})
}

func TestMergeMCPScriptsPreservesDependencies(t *testing.T) {
	compiler := NewCompiler()
	main := &MCPScriptsConfig{
		Mode:  "http",
		Tools: map[string]*MCPScriptToolConfig{},
	}

	imported := `{
		"fetch-url": {
			"description": "fetch url",
			"py": "print('ok')",
			"dependencies": ["requests", "urllib3"]
		}
	}`

	merged := compiler.mergeMCPScripts(main, []string{imported})
	tool, ok := merged.Tools["fetch-url"]
	if !ok {
		t.Fatal("expected merged tool fetch-url")
	}
	if !reflect.DeepEqual(tool.Dependencies, []string{"requests", "urllib3"}) {
		t.Fatalf("expected dependencies [requests urllib3], got %v", tool.Dependencies)
	}
}
