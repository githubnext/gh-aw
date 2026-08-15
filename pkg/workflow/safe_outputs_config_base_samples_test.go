package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSamplesValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []map[string]any
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice returns empty slice",
			input:    []any{},
			expected: []map[string]any{},
		},
		{
			name: "slice of map[string]any is returned as-is",
			input: []any{
				map[string]any{"a": 1},
				map[string]any{"b": "two"},
			},
			expected: []map[string]any{
				{"a": 1},
				{"b": "two"},
			},
		},
		{
			name: "slice with map[string]string is converted to map[string]any",
			input: []any{
				map[string]string{"x": "1", "y": "2"},
			},
			expected: []map[string]any{
				{"x": "1", "y": "2"},
			},
		},
		{
			name: "slice mixing map[string]any and map[string]string",
			input: []any{
				map[string]any{"a": 1},
				map[string]string{"b": "2"},
			},
			expected: []map[string]any{
				{"a": 1},
				{"b": "2"},
			},
		},
		{
			name: "slice with non-map entries are skipped",
			input: []any{
				"not a map",
				42,
				map[string]any{"a": 1},
				nil,
			},
			expected: []map[string]any{
				{"a": 1},
			},
		},
		{
			name:     "slice with only non-map entries returns empty slice",
			input:    []any{"a", 1, true},
			expected: []map[string]any{},
		},
		{
			name:     "single map[string]any is wrapped into one-element slice",
			input:    map[string]any{"key": "value"},
			expected: []map[string]any{{"key": "value"}},
		},
		{
			name:     "empty map[string]any is wrapped into one-element slice",
			input:    map[string]any{},
			expected: []map[string]any{{}},
		},
		{
			name:     "string input returns nil",
			input:    "not a valid shape",
			expected: nil,
		},
		{
			name:     "int input returns nil",
			input:    42,
			expected: nil,
		},
		{
			name:     "bool input returns nil",
			input:    true,
			expected: nil,
		},
		{
			name:     "map[string]string input (not map[string]any) returns nil",
			input:    map[string]string{"a": "b"},
			expected: nil,
		},
		{
			name:     "slice of strings input returns nil",
			input:    []string{"a", "b"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSamplesValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
