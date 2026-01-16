package workflow

import (
	"testing"
)

func TestParseRelativeTimeSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		// Hours - minimum 2 hours required
		{
			name:     "1 hour - below minimum",
			input:    "1h",
			expected: 0, // Rejected: less than 2h minimum
		},
		{
			name:     "2 hours - at minimum",
			input:    "2h",
			expected: 2, // 2 hours
		},
		{
			name:     "12 hours",
			input:    "12h",
			expected: 12, // 12 hours
		},
		{
			name:     "23 hours",
			input:    "23h",
			expected: 23, // 23 hours
		},
		{
			name:     "24 hours",
			input:    "24h",
			expected: 24, // 24 hours = 1 day
		},
		{
			name:     "48 hours",
			input:    "48h",
			expected: 48, // 48 hours = 2 days
		},
		{
			name:     "72 hours",
			input:    "72h",
			expected: 72, // 72 hours = 3 days
		},
		{
			name:     "uppercase hours - at minimum",
			input:    "2H",
			expected: 2,
		},
		{
			name:     "uppercase hours - below minimum",
			input:    "1H",
			expected: 0,
		},
		// Days
		{
			name:     "1 day",
			input:    "1d",
			expected: 24, // 1 day = 24 hours
		},
		{
			name:     "7 days",
			input:    "7d",
			expected: 168, // 7 days = 168 hours
		},
		{
			name:     "uppercase days",
			input:    "7D",
			expected: 168,
		},
		// Weeks
		{
			name:     "1 week",
			input:    "1w",
			expected: 168, // 1 week = 7 days = 168 hours
		},
		{
			name:     "2 weeks",
			input:    "2w",
			expected: 336, // 2 weeks = 14 days = 336 hours
		},
		{
			name:     "uppercase weeks",
			input:    "2W",
			expected: 336,
		},
		// Months
		{
			name:     "1 month",
			input:    "1m",
			expected: 720, // 1 month = 30 days = 720 hours
		},
		{
			name:     "3 months",
			input:    "3m",
			expected: 2160, // 3 months = 90 days = 2160 hours
		},
		{
			name:     "uppercase months",
			input:    "3M",
			expected: 2160,
		},
		// Years
		{
			name:     "1 year",
			input:    "1y",
			expected: 8760, // 1 year = 365 days = 8760 hours
		},
		{
			name:     "2 years",
			input:    "2y",
			expected: 17520, // 2 years = 730 days = 17520 hours
		},
		{
			name:     "uppercase years",
			input:    "2Y",
			expected: 17520,
		},
		// Invalid inputs
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "invalid unit",
			input:    "7x",
			expected: 0,
		},
		{
			name:     "no number",
			input:    "d",
			expected: 0,
		},
		{
			name:     "negative number",
			input:    "-7d",
			expected: 0,
		},
		{
			name:     "zero",
			input:    "0d",
			expected: 0,
		},
		{
			name:     "non-numeric",
			input:    "abcd",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRelativeTimeSpec(tt.input)
			if result != tt.expected {
				t.Errorf("parseRelativeTimeSpec(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseExpiresFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		expected int
	}{
		// Integer formats (treated as days for backward compatibility, converted to hours)
		{
			name:     "integer days",
			config:   map[string]any{"expires": 7},
			expected: 168, // 7 days = 168 hours
		},
		{
			name:     "int64",
			config:   map[string]any{"expires": int64(14)},
			expected: 336, // 14 days = 336 hours
		},
		{
			name:     "float64",
			config:   map[string]any{"expires": float64(21)},
			expected: 504, // 21 days = 504 hours
		},
		// String formats with hours
		{
			name:     "1 hour string - below minimum",
			config:   map[string]any{"expires": "1h"},
			expected: 0, // Rejected: less than 2h minimum
		},
		{
			name:     "2 hours string - at minimum",
			config:   map[string]any{"expires": "2h"},
			expected: 2, // 2 hours
		},
		{
			name:     "24 hours string",
			config:   map[string]any{"expires": "24h"},
			expected: 24, // 24 hours
		},
		{
			name:     "48 hours string",
			config:   map[string]any{"expires": "48h"},
			expected: 48, // 48 hours
		},
		// String formats with other units
		{
			name:     "7 days string",
			config:   map[string]any{"expires": "7d"},
			expected: 168, // 7 days = 168 hours
		},
		{
			name:     "2 weeks string",
			config:   map[string]any{"expires": "2w"},
			expected: 336, // 2 weeks = 14 days = 336 hours
		},
		{
			name:     "1 month string",
			config:   map[string]any{"expires": "1m"},
			expected: 720, // 1 month = 30 days = 720 hours
		},
		{
			name:     "1 year string",
			config:   map[string]any{"expires": "1y"},
			expected: 8760, // 1 year = 365 days = 8760 hours
		},
		// Missing or invalid
		{
			name:     "no expires field",
			config:   map[string]any{},
			expected: 0,
		},
		{
			name:     "invalid string",
			config:   map[string]any{"expires": "invalid"},
			expected: 0,
		},
		{
			name:     "wrong type",
			config:   map[string]any{"expires": true},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseExpiresFromConfig(tt.config)
			if result != tt.expected {
				t.Errorf("parseExpiresFromConfig(%v) = %d, expected %d", tt.config, result, tt.expected)
			}
		})
	}
}

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

// TestParseIntValueTruncation tests float truncation scenarios
func TestParseIntValueTruncation(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		expected       int
		shouldTruncate bool
	}{
		{
			name:           "clean conversion - no truncation",
			value:          60.0,
			expected:       60,
			shouldTruncate: false,
		},
		{
			name:           "truncation required - 60.5",
			value:          60.5,
			expected:       60,
			shouldTruncate: true,
		},
		{
			name:           "truncation required - 60.7",
			value:          60.7,
			expected:       60,
			shouldTruncate: true,
		},
		{
			name:           "clean conversion - 100.0",
			value:          100.0,
			expected:       100,
			shouldTruncate: false,
		},
		{
			name:           "truncation required - 123.99",
			value:          123.99,
			expected:       123,
			shouldTruncate: true,
		},
		{
			name:           "truncation required - negative with fraction",
			value:          -5.5,
			expected:       -5,
			shouldTruncate: true,
		},
		{
			name:           "clean conversion - negative integer",
			value:          -10.0,
			expected:       -10,
			shouldTruncate: false,
		},
		{
			name:           "truncation required - small fraction",
			value:          1.1,
			expected:       1,
			shouldTruncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := parseIntValue(tt.value)
			if !ok {
				t.Errorf("parseIntValue() should return ok=true for float64")
			}
			if result != tt.expected {
				t.Errorf("parseIntValue(%v) = %v, want %v", tt.value, result, tt.expected)
			}
			// Note: We can't directly test if warning was logged, but we verify the conversion is correct
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
