package parser

import (
	"testing"
)

func TestExtractFrontmatterFromContent_ToolsParsing(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantTools map[string]any
		wantErr   bool
	}{
		{
			name: "simple tools section with null values",
			content: `---
on: push
tools:
  github:
  edit:
  bash:
---

# Test Workflow`,
			wantTools: map[string]any{
				"github": nil,
				"edit":   nil,
				"bash":   nil,
			},
		},
		{
			name: "tools with array configuration",
			content: `---
on: push
tools:
  bash:
    - echo
    - ls
    - git status
---

# Test Workflow`,
			wantTools: map[string]any{
				"bash": []any{"echo", "ls", "git status"},
			},
		},
		{
			name: "tools with object configuration",
			content: `---
on: push
tools:
  github:
    allowed:
      - create_issue
      - add_comment
    mode: local
    read-only: true
---

# Test Workflow`,
			wantTools: map[string]any{
				"github": map[string]any{
					"allowed":   []any{"create_issue", "add_comment"},
					"mode":      "local",
					"read-only": true,
				},
			},
		},
		{
			name: "tools with playwright configuration",
			content: `---
on: push
tools:
  playwright:
    version: v1.41.0
    allowed_domains:
      - github.com
      - "*.example.com"
---

# Test Workflow`,
			wantTools: map[string]any{
				"playwright": map[string]any{
					"version":         "v1.41.0",
					"allowed_domains": []any{"github.com", "*.example.com"},
				},
			},
		},
		{
			name: "mixed tools configuration",
			content: `---
on: push
tools:
  github:
    allowed:
      - create_issue
  edit:
  bash:
    - echo
    - ls
  web-fetch:
  playwright:
    version: v1.41.0
---

# Test Workflow`,
			wantTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"create_issue"},
				},
				"edit":      nil,
				"bash":      []any{"echo", "ls"},
				"web-fetch": nil,
				"playwright": map[string]any{
					"version": "v1.41.0",
				},
			},
		},
		{
			name: "custom MCP server as tool",
			content: `---
on: push
tools:
  my-custom-server:
---

# Test Workflow`,
			wantTools: map[string]any{
				"my-custom-server": nil,
			},
		},
		{
			name: "no tools section",
			content: `---
on: push
permissions: read-all
---

# Test Workflow`,
			wantTools: map[string]any{},
		},
		{
			name: "empty tools section",
			content: `---
on: push
tools:
---

# Test Workflow`,
			wantTools: map[string]any{},
		},
		{
			name: "no frontmatter",
			content: `# Test Workflow

This is a test workflow without frontmatter.`,
			wantTools: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractFrontmatterFromContent(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractFrontmatterFromContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractFrontmatterFromContent() error = %v", err)
				return
			}

			// Check that Tools field is populated
			if result.Tools == nil {
				t.Errorf("ExtractFrontmatterFromContent() Tools field is nil, want ToolsConfig")
				return
			}

			// Convert ToolsConfig to map for comparison
			actualTools := result.Tools.ToMap()

			// Check tools length
			if len(tt.wantTools) != len(actualTools) {
				t.Errorf("ExtractFrontmatterFromContent() tools length = %v, want %v", len(actualTools), len(tt.wantTools))
			}

			// Check each tool
			for key, expectedValue := range tt.wantTools {
				actualValue, exists := actualTools[key]
				if !exists {
					t.Errorf("ExtractFrontmatterFromContent() missing tool key %v", key)
					continue
				}

				// Compare values based on type
				if !compareToolValues(actualValue, expectedValue) {
					t.Errorf("ExtractFrontmatterFromContent() tools[%v] = %#v, want %#v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestParseToolsFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantTools   map[string]any
	}{
		{
			name: "valid tools section",
			frontmatter: map[string]any{
				"on": "push",
				"tools": map[string]any{
					"github": nil,
					"edit":   nil,
				},
			},
			wantTools: map[string]any{
				"github": nil,
				"edit":   nil,
			},
		},
		{
			name: "no tools section",
			frontmatter: map[string]any{
				"on": "push",
			},
			wantTools: map[string]any{},
		},
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			wantTools:   map[string]any{},
		},
		{
			name: "tools is not a map (invalid)",
			frontmatter: map[string]any{
				"tools": "invalid",
			},
			wantTools: map[string]any{},
		},
		{
			name: "empty tools section",
			frontmatter: map[string]any{
				"tools": map[string]any{},
			},
			wantTools: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToolsConfig := parseToolsFromFrontmatter(tt.frontmatter)

			// Convert to map for comparison
			gotTools := gotToolsConfig.ToMap()

			if len(tt.wantTools) != len(gotTools) {
				t.Errorf("parseToolsFromFrontmatter() length = %v, want %v", len(gotTools), len(tt.wantTools))
			}

			for key, expectedValue := range tt.wantTools {
				actualValue, exists := gotTools[key]
				if !exists {
					t.Errorf("parseToolsFromFrontmatter() missing tool key %v", key)
					continue
				}

				if !compareToolValues(actualValue, expectedValue) {
					t.Errorf("parseToolsFromFrontmatter() tools[%v] = %#v, want %#v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

// compareToolValues compares two tool values, handling nil, arrays, and maps
func compareToolValues(actual, expected any) bool {
	// Handle nil cases
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		return false
	}

	// Handle array cases
	actualArray, actualIsArray := actual.([]any)
	expectedArray, expectedIsArray := expected.([]any)
	if actualIsArray && expectedIsArray {
		if len(actualArray) != len(expectedArray) {
			return false
		}
		for i := range actualArray {
			if actualArray[i] != expectedArray[i] {
				return false
			}
		}
		return true
	}

	// Handle map cases
	actualMap, actualIsMap := actual.(map[string]any)
	expectedMap, expectedIsMap := expected.(map[string]any)
	if actualIsMap && expectedIsMap {
		if len(actualMap) != len(expectedMap) {
			return false
		}
		for key, expectedVal := range expectedMap {
			actualVal, exists := actualMap[key]
			if !exists {
				return false
			}
			if !compareToolValues(actualVal, expectedVal) {
				return false
			}
		}
		return true
	}

	// Handle primitive types
	return actual == expected
}

func TestToolConfigHelperMethods(t *testing.T) {
	tests := []struct {
		name       string
		config     *ToolConfig
		wantNil    bool
		wantString string
		hasString  bool
		wantArray  []any
		hasArray   bool
		wantObject map[string]any
		hasObject  bool
	}{
		{
			name:      "nil configuration",
			config:    &ToolConfig{Raw: nil},
			wantNil:   true,
			hasString: false,
			hasArray:  false,
			hasObject: false,
		},
		{
			name:       "string configuration",
			config:     &ToolConfig{Raw: "simple"},
			wantNil:    false,
			wantString: "simple",
			hasString:  true,
			hasArray:   false,
			hasObject:  false,
		},
		{
			name:      "array configuration",
			config:    &ToolConfig{Raw: []any{"echo", "ls"}},
			wantNil:   false,
			hasString: false,
			wantArray: []any{"echo", "ls"},
			hasArray:  true,
			hasObject: false,
		},
		{
			name:       "object configuration",
			config:     &ToolConfig{Raw: map[string]any{"allowed": []any{"create_issue"}}},
			wantNil:    false,
			hasString:  false,
			hasArray:   false,
			wantObject: map[string]any{"allowed": []any{"create_issue"}},
			hasObject:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test IsNil
			if got := tt.config.IsNil(); got != tt.wantNil {
				t.Errorf("IsNil() = %v, want %v", got, tt.wantNil)
			}

			// Test AsString
			gotString, hasString := tt.config.AsString()
			if hasString != tt.hasString {
				t.Errorf("AsString() hasString = %v, want %v", hasString, tt.hasString)
			}
			if hasString && gotString != tt.wantString {
				t.Errorf("AsString() = %v, want %v", gotString, tt.wantString)
			}

			// Test AsArray
			gotArray, hasArray := tt.config.AsArray()
			if hasArray != tt.hasArray {
				t.Errorf("AsArray() hasArray = %v, want %v", hasArray, tt.hasArray)
			}
			if hasArray && !compareToolValues(gotArray, tt.wantArray) {
				t.Errorf("AsArray() = %v, want %v", gotArray, tt.wantArray)
			}

			// Test AsObject
			gotObject, hasObject := tt.config.AsObject()
			if hasObject != tt.hasObject {
				t.Errorf("AsObject() hasObject = %v, want %v", hasObject, tt.hasObject)
			}
			if hasObject && !compareToolValues(gotObject, tt.wantObject) {
				t.Errorf("AsObject() = %v, want %v", gotObject, tt.wantObject)
			}
		})
	}
}

func TestToolsConfigHelperMethods(t *testing.T) {
	toolsConfig := &ToolsConfig{
		Tools: map[string]*ToolConfig{
			"github": {Raw: map[string]any{"allowed": []any{"create_issue"}}},
			"edit":   {Raw: nil},
			"bash":   {Raw: []any{"echo", "ls"}},
		},
	}

	t.Run("Has returns true for existing tool", func(t *testing.T) {
		if !toolsConfig.Has("github") {
			t.Error("Has(github) = false, want true")
		}
	})

	t.Run("Has returns false for non-existing tool", func(t *testing.T) {
		if toolsConfig.Has("nonexistent") {
			t.Error("Has(nonexistent) = true, want false")
		}
	})

	t.Run("Get returns config for existing tool", func(t *testing.T) {
		config, exists := toolsConfig.Get("github")
		if !exists {
			t.Error("Get(github) exists = false, want true")
		}
		if config == nil {
			t.Error("Get(github) config = nil, want non-nil")
		}
	})

	t.Run("Get returns false for non-existing tool", func(t *testing.T) {
		_, exists := toolsConfig.Get("nonexistent")
		if exists {
			t.Error("Get(nonexistent) exists = true, want false")
		}
	})

	t.Run("ToMap returns correct map", func(t *testing.T) {
		m := toolsConfig.ToMap()
		if len(m) != 3 {
			t.Errorf("ToMap() length = %v, want 3", len(m))
		}
		if _, exists := m["github"]; !exists {
			t.Error("ToMap() missing github")
		}
	})

	t.Run("nil ToolsConfig", func(t *testing.T) {
		var nilConfig *ToolsConfig
		if nilConfig.Has("github") {
			t.Error("nil.Has(github) = true, want false")
		}
		if _, exists := nilConfig.Get("github"); exists {
			t.Error("nil.Get(github) exists = true, want false")
		}
		m := nilConfig.ToMap()
		if len(m) != 0 {
			t.Errorf("nil.ToMap() length = %v, want 0", len(m))
		}
	})
}
