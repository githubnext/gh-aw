package workflow

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestUnquoteYAMLKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		key      string
		expected string
	}{
		{
			name: "do not unquote 'on' - YAML boolean keyword",
			input: `"on":
  issues:
    types:
    - opened`,
			key: "on",
			expected: `"on":
  issues:
    types:
    - opened`,
		},
		{
			name: "do not unquote 'on' with indentation - YAML boolean keyword",
			input: `  "on":
    issues:
      types:
      - opened`,
			key: "on",
			expected: `  "on":
    issues:
      types:
      - opened`,
		},
		{
			name:     "do not unquote 'on' in middle of line",
			input:    `key: "on":value`,
			key:      "on",
			expected: `key: "on":value`,
		},
		{
			name:     "do not unquote 'on' in string value",
			input:    `description: "This is about on: something"`,
			key:      "on",
			expected: `description: "This is about on: something"`,
		},
		{
			name: "do not unquote 'on' multiple occurrences - YAML boolean keyword",
			input: `"on":
  issues:
    types:
    - opened
"on":
  push:
    branches:
    - main`,
			key: "on",
			expected: `"on":
  issues:
    types:
    - opened
"on":
  push:
    branches:
    - main`,
		},
		{
			name: "do not unquote 'yes' - YAML boolean keyword",
			input: `"yes":
  value: test`,
			key: "yes",
			expected: `"yes":
  value: test`,
		},
		{
			name: "do not unquote 'no' - YAML boolean keyword",
			input: `"no":
  value: test`,
			key: "no",
			expected: `"no":
  value: test`,
		},
		{
			name: "do not unquote 'true' - YAML boolean keyword",
			input: `"true":
  value: test`,
			key: "true",
			expected: `"true":
  value: test`,
		},
		{
			name: "do not unquote 'false' - YAML boolean keyword",
			input: `"false":
  value: test`,
			key: "false",
			expected: `"false":
  value: test`,
		},
		{
			name: "do not unquote 'off' - YAML boolean keyword",
			input: `"off":
  value: test`,
			key: "off",
			expected: `"off":
  value: test`,
		},
		{
			name: "unquote 'if' - not a YAML boolean keyword",
			input: `"if":
  github.actor == 'bot'`,
			key: "if",
			expected: `if:
  github.actor == 'bot'`,
		},
		{
			name: "handle key with special regex characters",
			input: `"key.with.dots":
  value: test`,
			key: "key.with.dots",
			expected: `key.with.dots:
  value: test`,
		},
		{
			name: "no change when key is already unquoted - YAML boolean keyword preserved",
			input: `"on":
  issues:
    types:
    - opened`,
			key: "on",
			expected: `"on":
  issues:
    types:
    - opened`,
		},
		{
			name: "do not unquote 'on' with tabs - YAML boolean keyword",
			input: `		"on":
		  issues:`,
			key: "on",
			expected: `		"on":
		  issues:`,
		},
		{
			name:     "empty string",
			input:    "",
			key:      "on",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			key:      "on",
			expected: "\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnquoteYAMLKey(tt.input, tt.key)
			if result != tt.expected {
				t.Errorf("UnquoteYAMLKey() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
					tt.input, tt.expected, result)
			}
		})
	}
}

func TestMarshalWithFieldOrder(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]any
		priorityFields []string
		expectedOrder  []string
	}{
		{
			name: "on section with events in priority order",
			data: map[string]any{
				"workflow_dispatch": map[string]any{},
				"push": map[string]any{
					"branches": []string{"main"},
				},
				"issues": map[string]any{
					"types": []string{"opened"},
				},
			},
			priorityFields: []string{"push", "pull_request", "issues", "workflow_dispatch"},
			expectedOrder:  []string{"push", "issues", "workflow_dispatch"},
		},
		{
			name: "permissions with mixed order",
			data: map[string]any{
				"pull-requests": "write",
				"contents":      "read",
				"issues":        "write",
			},
			priorityFields: []string{"actions", "contents", "issues", "pull-requests"},
			expectedOrder:  []string{"contents", "issues", "pull-requests"},
		},
		{
			name: "alphabetical fallback for non-priority fields",
			data: map[string]any{
				"zebra": "value",
				"alpha": "value",
				"beta":  "value",
			},
			priorityFields: []string{},
			expectedOrder:  []string{"alpha", "beta", "zebra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlBytes, err := MarshalWithFieldOrder(tt.data, tt.priorityFields)
			if err != nil {
				t.Errorf("MarshalWithFieldOrder() error = %v", err)
				return
			}

			yamlStr := string(yamlBytes)
			t.Logf("Generated YAML:\n%s", yamlStr)

			// Parse the YAML to extract the field order
			var parsed yaml.MapSlice
			if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
				t.Errorf("Failed to parse generated YAML: %v", err)
				return
			}

			// Extract the field names in order
			var actualOrder []string
			for _, item := range parsed {
				if key, ok := item.Key.(string); ok {
					actualOrder = append(actualOrder, key)
				}
			}

			// Verify the order matches expected
			if len(actualOrder) != len(tt.expectedOrder) {
				t.Errorf("Expected %d fields, got %d", len(tt.expectedOrder), len(actualOrder))
			}

			for i, expected := range tt.expectedOrder {
				if i >= len(actualOrder) || actualOrder[i] != expected {
					t.Errorf("Field %d: expected %q, got %q. Full order: %v", i, expected, actualOrder[i], actualOrder)
				}
			}
		})
	}
}

func TestExtractTopLevelYAMLSectionWithOrdering(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name          string
		frontmatter   map[string]any
		key           string
		expectedOrder []string
	}{
		{
			name: "on section orders events alphabetically",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_dispatch": map[string]any{},
					"push": map[string]any{
						"branches": []string{"main"},
					},
					"issues": map[string]any{
						"types": []string{"opened"},
					},
				},
			},
			key:           "on",
			expectedOrder: []string{"issues", "push", "workflow_dispatch"},
		},
		{
			name: "permissions section orders alphabetically",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"pull-requests": "write",
					"contents":      "read",
					"issues":        "write",
				},
			},
			key:           "permissions",
			expectedOrder: []string{"contents", "issues", "pull-requests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractTopLevelYAMLSection(tt.frontmatter, tt.key)
			if result == "" {
				t.Error("Expected non-empty result")
				return
			}

			t.Logf("Generated YAML:\n%s", result)

			// Parse the YAML to verify field order
			var parsed map[string]yaml.MapSlice
			if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("Failed to parse generated YAML: %v", err)
				return
			}

			section, ok := parsed[tt.key]
			if !ok {
				t.Errorf("Expected section %q not found in parsed YAML", tt.key)
				return
			}

			// Extract field names in order
			var actualOrder []string
			for _, item := range section {
				if key, ok := item.Key.(string); ok {
					actualOrder = append(actualOrder, key)
				}
			}

			// Verify order
			if len(actualOrder) != len(tt.expectedOrder) {
				t.Errorf("Expected %d fields, got %d", len(tt.expectedOrder), len(actualOrder))
			}

			for i, expected := range tt.expectedOrder {
				if i >= len(actualOrder) || actualOrder[i] != expected {
					t.Errorf("Field %d: expected %q, got %q. Full order: %v", i, expected, actualOrder[i], actualOrder)
				}
			}

			// Also check that the YAML string has the fields in the right order
			lines := strings.Split(result, "\n")
			fieldLineIndices := make(map[string]int)
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				for _, field := range tt.expectedOrder {
					if strings.HasPrefix(trimmed, field+":") {
						fieldLineIndices[field] = i
					}
				}
			}

			// Verify that the line indices are in ascending order for the expected fields
			for i := 1; i < len(tt.expectedOrder); i++ {
				prev := tt.expectedOrder[i-1]
				curr := tt.expectedOrder[i]
				prevIdx, prevOk := fieldLineIndices[prev]
				currIdx, currOk := fieldLineIndices[curr]
				if prevOk && currOk && prevIdx >= currIdx {
					t.Errorf("Field %q (line %d) should come before %q (line %d)", prev, prevIdx, curr, currIdx)
				}
			}
		})
	}
}
