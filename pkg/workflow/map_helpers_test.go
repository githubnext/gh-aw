package workflow

import (
	"testing"
)

func TestParseIntValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected int
		ok       bool
	}{
		{
			name:     "int value",
			value:    42,
			expected: 42,
			ok:       true,
		},
		{
			name:     "int64 value",
			value:    int64(100),
			expected: 100,
			ok:       true,
		},
		{
			name:     "uint64 value",
			value:    uint64(200),
			expected: 200,
			ok:       true,
		},
		{
			name:     "float64 value",
			value:    float64(3.14),
			expected: 3,
			ok:       true,
		},
		{
			name:     "string value (not supported)",
			value:    "42",
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil value",
			value:    nil,
			expected: 0,
			ok:       false,
		},
		{
			name:     "bool value (not supported)",
			value:    true,
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := parseIntValue(tt.value)
			if ok != tt.ok {
				t.Errorf("parseIntValue() ok = %v, want %v", ok, tt.ok)
			}
			if result != tt.expected {
				t.Errorf("parseIntValue() result = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterMapKeys(t *testing.T) {
	tests := []struct {
		name        string
		original    map[string]any
		excludeKeys []string
		expected    map[string]any
	}{
		{
			name: "filter single key",
			original: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			excludeKeys: []string{"key2"},
			expected: map[string]any{
				"key1": "value1",
				"key3": "value3",
			},
		},
		{
			name: "filter multiple keys",
			original: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
				"key4": "value4",
			},
			excludeKeys: []string{"key1", "key3"},
			expected: map[string]any{
				"key2": "value2",
				"key4": "value4",
			},
		},
		{
			name: "filter no keys",
			original: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			excludeKeys: []string{},
			expected: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "filter non-existent key",
			original: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			excludeKeys: []string{"key3"},
			expected: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:        "empty original map",
			original:    map[string]any{},
			excludeKeys: []string{"key1"},
			expected:    map[string]any{},
		},
		{
			name: "filter all keys",
			original: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			excludeKeys: []string{"key1", "key2"},
			expected:    map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterMapKeys(tt.original, tt.excludeKeys...)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("filterMapKeys() length = %v, want %v", len(result), len(tt.expected))
			}

			// Check each key-value pair
			for key, expectedValue := range tt.expected {
				resultValue, exists := result[key]
				if !exists {
					t.Errorf("filterMapKeys() missing key %v", key)
				}
				if resultValue != expectedValue {
					t.Errorf("filterMapKeys() value for key %v = %v, want %v", key, resultValue, expectedValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, exists := tt.expected[key]; !exists {
					t.Errorf("filterMapKeys() unexpected key %v", key)
				}
			}
		})
	}
}
