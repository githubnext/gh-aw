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
