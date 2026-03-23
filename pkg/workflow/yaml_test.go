//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestCleanYAMLNullValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "workflow_dispatch with null",
			input: `on:
  schedule:
  - cron: "0 0 * * *"
  workflow_dispatch: null`,
			expected: `on:
  schedule:
  - cron: "0 0 * * *"
  workflow_dispatch:`,
		},
		{
			name: "multiple null values",
			input: `on:
  workflow_dispatch: null
  workflow_call: null
  schedule:
  - cron: "0 0 * * *"`,
			expected: `on:
  workflow_dispatch:
  workflow_call:
  schedule:
  - cron: "0 0 * * *"`,
		},
		{
			name: "null with extra whitespace",
			input: `on:
  workflow_dispatch:   null  
  schedule:
  - cron: "0 0 * * *"`,
			expected: `on:
  workflow_dispatch:
  schedule:
  - cron: "0 0 * * *"`,
		},
		{
			name:     "string containing null should not be modified",
			input:    `description: "This is a null value"`,
			expected: `description: "This is a null value"`,
		},
		{
			name: "null in the middle of line should not be modified",
			input: `key: null is not good
  workflow_dispatch: null`,
			expected: `key: null is not good
  workflow_dispatch:`,
		},
		{
			name: "no null values",
			input: `on:
  workflow_dispatch:
    inputs:
      issue_url:
        required: true`,
			expected: `on:
  workflow_dispatch:
    inputs:
      issue_url:
        required: true`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanYAMLNullValues(tt.input)
			if result != tt.expected {
				t.Errorf("CleanYAMLNullValues() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
					tt.input, tt.expected, result)
			}
		})
	}
}

func TestUnquoteYAMLKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		key      string
		expected string
	}{
		{
			name: "unquote 'on' at start of line",
			input: `"on":
  issues:
    types:
    - opened`,
			key: "on",
			expected: `on:
  issues:
    types:
    - opened`,
		},
		{
			name: "unquote 'on' with indentation",
			input: `  "on":
    issues:
      types:
      - opened`,
			key: "on",
			expected: `  on:
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
			name: "unquote multiple occurrences at start of lines",
			input: `"on":
  issues:
    types:
    - opened
"on":
  push:
    branches:
    - main`,
			key: "on",
			expected: `on:
  issues:
    types:
    - opened
on:
  push:
    branches:
    - main`,
		},
		{
			name: "unquote other keys",
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
			name: "no change when key is not quoted",
			input: `on:
  issues:
    types:
    - opened`,
			key: "on",
			expected: `on:
  issues:
    types:
    - opened`,
		},
		{
			name: "unquote with tabs",
			input: `		"on":
		  issues:`,
			key: "on",
			expected: `		on:
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
	compiler := NewCompiler()

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

// BenchmarkUnquoteYAMLKey measures single-pass regex replacement performance.
// This benchmark guards against regressions where the implementation calls
// FindStringSubmatch inside a ReplaceAllStringFunc callback (double regex pass).
func BenchmarkUnquoteYAMLKey(b *testing.B) {
	// A realistic workflow YAML with the "on" key quoted by the marshaler
	yamlStr := `"on":
  push:
    branches:
    - main
  pull_request:
    types:
    - opened
    - synchronize
  issues:
    types:
    - opened
  workflow_dispatch:
jobs:
  agent:
    runs-on: ubuntu-latest
    steps:
    - name: Run
      run: echo hello
`
	b.ReportAllocs()
	for b.Loop() {
		_ = UnquoteYAMLKey(yamlStr, "on")
	}
}

func TestFormatYAMLValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		// string cases
		{name: "string: true keyword quoted", value: "true", expected: "'true'"},
		{name: "string: false keyword quoted", value: "false", expected: "'false'"},
		{name: "string: null keyword quoted", value: "null", expected: "'null'"},
		{name: "string: numeric string quoted", value: "42", expected: "'42'"},
		{name: "string: float string quoted", value: "3.14", expected: "'3.14'"},
		{name: "string: plain string quoted", value: "hello", expected: "'hello'"},
		{name: "string: empty string quoted", value: "", expected: "''"},
		// bool cases
		{name: "bool: true", value: true, expected: "true"},
		{name: "bool: false", value: false, expected: "false"},
		// integer cases
		{name: "int", value: int(42), expected: "42"},
		{name: "int8", value: int8(8), expected: "8"},
		{name: "int16", value: int16(16), expected: "16"},
		{name: "int32", value: int32(32), expected: "32"},
		{name: "int64", value: int64(64), expected: "64"},
		{name: "int: zero", value: int(0), expected: "0"},
		{name: "int: negative", value: int(-1), expected: "-1"},
		// unsigned integer cases
		{name: "uint", value: uint(10), expected: "10"},
		{name: "uint8", value: uint8(8), expected: "8"},
		{name: "uint16", value: uint16(16), expected: "16"},
		{name: "uint32", value: uint32(32), expected: "32"},
		{name: "uint64", value: uint64(64), expected: "64"},
		// float cases
		{name: "float32", value: float32(1.5), expected: "1.5"},
		{name: "float64", value: float64(2.5), expected: "2.5"},
		// default fallback
		{name: "nil: quoted", value: nil, expected: "'<nil>'"},
		{name: "struct: quoted", value: struct{ A int }{A: 1}, expected: "'{1}'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatYAMLValue(tt.value)
			if result != tt.expected {
				t.Errorf("formatYAMLValue(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}
