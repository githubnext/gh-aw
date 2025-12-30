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
		// Hours - should convert to days (minimum 1 day)
		{
			name:     "2 hours",
			input:    "2h",
			expected: 1, // 2 hours = 1 day (minimum)
		},
		{
			name:     "12 hours",
			input:    "12h",
			expected: 1, // 12 hours = 1 day (minimum)
		},
		{
			name:     "23 hours",
			input:    "23h",
			expected: 1, // 23 hours = 1 day (minimum)
		},
		{
			name:     "24 hours",
			input:    "24h",
			expected: 1, // 24 hours = 1 day
		},
		{
			name:     "48 hours",
			input:    "48h",
			expected: 2, // 48 hours = 2 days
		},
		{
			name:     "72 hours",
			input:    "72h",
			expected: 3, // 72 hours = 3 days
		},
		{
			name:     "uppercase hours",
			input:    "2H",
			expected: 1,
		},
		// Days
		{
			name:     "1 day",
			input:    "1d",
			expected: 1,
		},
		{
			name:     "7 days",
			input:    "7d",
			expected: 7,
		},
		{
			name:     "uppercase days",
			input:    "7D",
			expected: 7,
		},
		// Weeks
		{
			name:     "1 week",
			input:    "1w",
			expected: 7,
		},
		{
			name:     "2 weeks",
			input:    "2w",
			expected: 14,
		},
		{
			name:     "uppercase weeks",
			input:    "2W",
			expected: 14,
		},
		// Months
		{
			name:     "1 month",
			input:    "1m",
			expected: 30,
		},
		{
			name:     "3 months",
			input:    "3m",
			expected: 90,
		},
		{
			name:     "uppercase months",
			input:    "3M",
			expected: 90,
		},
		// Years
		{
			name:     "1 year",
			input:    "1y",
			expected: 365,
		},
		{
			name:     "2 years",
			input:    "2y",
			expected: 730,
		},
		{
			name:     "uppercase years",
			input:    "2Y",
			expected: 730,
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
		// Integer formats
		{
			name:     "integer days",
			config:   map[string]any{"expires": 7},
			expected: 7,
		},
		{
			name:     "int64",
			config:   map[string]any{"expires": int64(14)},
			expected: 14,
		},
		{
			name:     "float64",
			config:   map[string]any{"expires": float64(21)},
			expected: 21,
		},
		// String formats with hours
		{
			name:     "2 hours string",
			config:   map[string]any{"expires": "2h"},
			expected: 1, // 2 hours = 1 day (minimum)
		},
		{
			name:     "24 hours string",
			config:   map[string]any{"expires": "24h"},
			expected: 1,
		},
		{
			name:     "48 hours string",
			config:   map[string]any{"expires": "48h"},
			expected: 2,
		},
		// String formats with other units
		{
			name:     "7 days string",
			config:   map[string]any{"expires": "7d"},
			expected: 7,
		},
		{
			name:     "2 weeks string",
			config:   map[string]any{"expires": "2w"},
			expected: 14,
		},
		{
			name:     "1 month string",
			config:   map[string]any{"expires": "1m"},
			expected: 30,
		},
		{
			name:     "1 year string",
			config:   map[string]any{"expires": "1y"},
			expected: 365,
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
