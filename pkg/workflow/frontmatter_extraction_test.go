package workflow

import (
	"testing"
)

func TestExtractYAMLValue(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name        string
		frontmatter map[string]any
		key         string
		expected    string
	}{
		{
			name:        "string value",
			frontmatter: map[string]any{"name": "test-workflow"},
			key:         "name",
			expected:    "test-workflow",
		},
		{
			name:        "int value",
			frontmatter: map[string]any{"timeout": 42},
			key:         "timeout",
			expected:    "42",
		},
		{
			name:        "int64 value",
			frontmatter: map[string]any{"count": int64(12345)},
			key:         "count",
			expected:    "12345",
		},
		{
			name:        "uint64 value",
			frontmatter: map[string]any{"id": uint64(99999)},
			key:         "id",
			expected:    "99999",
		},
		{
			name:        "float64 value",
			frontmatter: map[string]any{"version": 3.14},
			key:         "version",
			expected:    "3",
		},
		{
			name:        "float64 whole number",
			frontmatter: map[string]any{"port": 8080.0},
			key:         "port",
			expected:    "8080",
		},
		{
			name:        "key not found",
			frontmatter: map[string]any{"name": "test"},
			key:         "missing",
			expected:    "",
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			key:         "name",
			expected:    "",
		},
		{
			name:        "unsupported type (array)",
			frontmatter: map[string]any{"items": []string{"a", "b"}},
			key:         "items",
			expected:    "",
		},
		{
			name:        "unsupported type (map)",
			frontmatter: map[string]any{"config": map[string]string{"key": "value"}},
			key:         "config",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractYAMLValue(tt.frontmatter, tt.key)
			if result != tt.expected {
				t.Errorf("extractYAMLValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractFeatures(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    map[string]any
	}{
		{
			name: "valid features map with boolean values",
			frontmatter: map[string]any{
				"features": map[string]any{
					"feature1": true,
					"feature2": false,
					"feature3": true,
				},
			},
			expected: map[string]any{
				"feature1": true,
				"feature2": false,
				"feature3": true,
			},
		},
		{
			name:        "features key not present",
			frontmatter: map[string]any{"other": "value"},
			expected:    nil,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			expected:    nil,
		},
		{
			name: "features is not a map",
			frontmatter: map[string]any{
				"features": "not a map",
			},
			expected: nil,
		},
		{
			name: "features map with mixed value types",
			frontmatter: map[string]any{
				"features": map[string]any{
					"feature1":   true,
					"feature2":   "string value",
					"action-tag": "2d4c6ce24c55704d72ec674d1f5c357831435180",
				},
			},
			expected: map[string]any{
				"feature1":   true,
				"feature2":   "string value",
				"action-tag": "2d4c6ce24c55704d72ec674d1f5c357831435180",
			},
		},
		{
			name: "empty features map",
			frontmatter: map[string]any{
				"features": map[string]any{},
			},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractFeatures(tt.frontmatter)

			if result == nil && tt.expected == nil {
				return
			}

			if (result == nil) != (tt.expected == nil) {
				t.Errorf("extractFeatures() = %v, want %v", result, tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("extractFeatures() length = %d, want %d", len(result), len(tt.expected))
				return
			}

			for key, expectedVal := range tt.expected {
				if actualVal, ok := result[key]; !ok || actualVal != expectedVal {
					t.Errorf("extractFeatures()[%q] = %v, want %v", key, actualVal, expectedVal)
				}
			}
		})
	}
}

func TestExtractToolsStartupTimeout(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name     string
		tools    map[string]any
		expected int
	}{
		{
			name: "int timeout",
			tools: map[string]any{
				"startup-timeout": 30,
			},
			expected: 30,
		},
		{
			name: "int64 timeout",
			tools: map[string]any{
				"startup-timeout": int64(60),
			},
			expected: 60,
		},
		{
			name: "uint timeout",
			tools: map[string]any{
				"startup-timeout": uint(45),
			},
			expected: 45,
		},
		{
			name: "uint64 timeout",
			tools: map[string]any{
				"startup-timeout": uint64(90),
			},
			expected: 90,
		},
		{
			name: "float64 timeout",
			tools: map[string]any{
				"startup-timeout": 120.0,
			},
			expected: 120,
		},
		{
			name:     "nil tools",
			tools:    nil,
			expected: 0,
		},
		{
			name:     "empty tools map",
			tools:    map[string]any{},
			expected: 0,
		},
		{
			name: "startup-timeout not present",
			tools: map[string]any{
				"other-field": "value",
			},
			expected: 0,
		},
		{
			name: "invalid type (string)",
			tools: map[string]any{
				"startup-timeout": "not a number",
			},
			expected: 0,
		},
		{
			name: "invalid type (array)",
			tools: map[string]any{
				"startup-timeout": []int{1, 2, 3},
			},
			expected: 0,
		},
		{
			name: "zero timeout",
			tools: map[string]any{
				"startup-timeout": 0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractToolsStartupTimeout(tt.tools)
			if result != tt.expected {
				t.Errorf("extractToolsStartupTimeout() = %d, want %d", result, tt.expected)
			}
		})
	}
}
