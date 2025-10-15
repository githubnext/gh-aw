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
				t.Errorf("ExtractFrontmatterFromContent() Tools field is nil, want map")
				return
			}

			// Check tools length
			if len(tt.wantTools) != len(result.Tools) {
				t.Errorf("ExtractFrontmatterFromContent() tools length = %v, want %v", len(result.Tools), len(tt.wantTools))
			}

			// Check each tool
			for key, expectedValue := range tt.wantTools {
				actualValue, exists := result.Tools[key]
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
			gotTools := parseToolsFromFrontmatter(tt.frontmatter)

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
