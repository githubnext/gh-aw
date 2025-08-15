package workflow

import (
	"testing"
	"time"
)

func TestParseTimeDelta(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *TimeDelta
		expectError bool
		errorMsg    string
	}{
		// Valid cases
		{
			name:     "hours only",
			input:    "+25h",
			expected: &TimeDelta{Hours: 25},
		},
		{
			name:     "days only",
			input:    "+3d",
			expected: &TimeDelta{Days: 3},
		},
		{
			name:     "minutes only",
			input:    "+30m",
			expected: &TimeDelta{Minutes: 30},
		},
		{
			name:     "days and hours",
			input:    "+1d12h",
			expected: &TimeDelta{Days: 1, Hours: 12},
		},
		{
			name:     "all units",
			input:    "+2d5h30m",
			expected: &TimeDelta{Days: 2, Hours: 5, Minutes: 30},
		},
		{
			name:     "different order",
			input:    "+5h2d30m",
			expected: &TimeDelta{Days: 2, Hours: 5, Minutes: 30},
		},
		{
			name:     "single digit",
			input:    "+1d",
			expected: &TimeDelta{Days: 1},
		},
		{
			name:     "large numbers",
			input:    "+100h",
			expected: &TimeDelta{Hours: 100},
		},
		{
			name:     "zero values allowed in middle",
			input:    "+0d5h",
			expected: &TimeDelta{Days: 0, Hours: 5},
		},

		// Error cases
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errorMsg:    "empty time delta",
		},
		{
			name:        "no plus prefix",
			input:       "25h",
			expectError: true,
			errorMsg:    "time delta must start with '+'",
		},
		{
			name:        "only plus",
			input:       "+",
			expectError: true,
			errorMsg:    "empty time delta after '+'",
		},
		{
			name:        "no units",
			input:       "+25",
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
		{
			name:        "invalid unit",
			input:       "+25x",
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
		{
			name:        "duplicate units",
			input:       "+25h5h",
			expectError: true,
			errorMsg:    "duplicate unit 'h'",
		},
		{
			name:        "invalid characters",
			input:       "+25h5x",
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
		{
			name:        "negative numbers not allowed",
			input:       "+-5h",
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
		{
			name:        "too many days",
			input:       "+400d",
			expectError: true,
			errorMsg:    "time delta too large: 400 days exceeds maximum",
		},
		{
			name:        "too many hours",
			input:       "+9000h",
			expectError: true,
			errorMsg:    "time delta too large: 9000 hours exceeds maximum",
		},
		{
			name:        "extra characters",
			input:       "+5h extra",
			expectError: true,
			errorMsg:    "Extra characters detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeDelta(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseTimeDelta(%q) expected error but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("parseTimeDelta(%q) error = %v, want to contain %v", tt.input, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("parseTimeDelta(%q) unexpected error: %v", tt.input, err)
					return
				}
				if result == nil {
					t.Errorf("parseTimeDelta(%q) returned nil result", tt.input)
					return
				}
				if result.Days != tt.expected.Days || result.Hours != tt.expected.Hours || result.Minutes != tt.expected.Minutes {
					t.Errorf("parseTimeDelta(%q) = {Days: %d, Hours: %d, Minutes: %d}, want {Days: %d, Hours: %d, Minutes: %d}",
						tt.input, result.Days, result.Hours, result.Minutes, tt.expected.Days, tt.expected.Hours, tt.expected.Minutes)
				}
			}
		})
	}
}

func TestTimeDeltaToDuration(t *testing.T) {
	tests := []struct {
		name     string
		delta    *TimeDelta
		expected time.Duration
	}{
		{
			name:     "hours only",
			delta:    &TimeDelta{Hours: 25},
			expected: 25 * time.Hour,
		},
		{
			name:     "days only",
			delta:    &TimeDelta{Days: 3},
			expected: 3 * 24 * time.Hour,
		},
		{
			name:     "minutes only",
			delta:    &TimeDelta{Minutes: 30},
			expected: 30 * time.Minute,
		},
		{
			name:     "all units",
			delta:    &TimeDelta{Days: 2, Hours: 5, Minutes: 30},
			expected: 2*24*time.Hour + 5*time.Hour + 30*time.Minute,
		},
		{
			name:     "zero values",
			delta:    &TimeDelta{Days: 0, Hours: 0, Minutes: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.delta.toDuration()
			if result != tt.expected {
				t.Errorf("TimeDelta.toDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTimeDeltaString(t *testing.T) {
	tests := []struct {
		name     string
		delta    *TimeDelta
		expected string
	}{
		{
			name:     "hours only",
			delta:    &TimeDelta{Hours: 25},
			expected: "+25h",
		},
		{
			name:     "days only",
			delta:    &TimeDelta{Days: 3},
			expected: "+3d",
		},
		{
			name:     "minutes only",
			delta:    &TimeDelta{Minutes: 30},
			expected: "+30m",
		},
		{
			name:     "all units",
			delta:    &TimeDelta{Days: 2, Hours: 5, Minutes: 30},
			expected: "+2d5h30m",
		},
		{
			name:     "zero values",
			delta:    &TimeDelta{Days: 0, Hours: 0, Minutes: 0},
			expected: "0m",
		},
		{
			name:     "some zero values",
			delta:    &TimeDelta{Days: 1, Hours: 0, Minutes: 30},
			expected: "+1d30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.delta.String()
			if result != tt.expected {
				t.Errorf("TimeDelta.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsRelativeStopTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "relative time delta",
			input:    "+25h",
			expected: true,
		},
		{
			name:     "absolute timestamp",
			input:    "2025-12-31 23:59:59",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "just plus",
			input:    "+",
			expected: true,
		},
		{
			name:     "plus in middle",
			input:    "25h+5m",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRelativeStopTime(tt.input)
			if result != tt.expected {
				t.Errorf("isRelativeStopTime(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseAbsoluteDateTime(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedErr   bool
		expectedDay   int // Day of month to verify correct parsing
		expectedMonth time.Month
		expectedYear  int
	}{
		// Standard formats
		{
			name:          "standard YYYY-MM-DD HH:MM:SS",
			input:         "2025-06-01 14:30:00",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "ISO 8601 format",
			input:         "2025-06-01T14:30:00",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "date only YYYY-MM-DD",
			input:         "2025-06-01",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},

		// US format MM/DD/YYYY
		{
			name:          "US format MM/DD/YYYY",
			input:         "06/01/2025",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "US format with time",
			input:         "06/01/2025 14:30",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},

		// Readable formats
		{
			name:          "readable January 1, 2025",
			input:         "January 1, 2025",
			expectedDay:   1,
			expectedMonth: time.January,
			expectedYear:  2025,
		},
		{
			name:          "readable June 15, 2025",
			input:         "June 15, 2025",
			expectedDay:   15,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "readable with abbreviated month",
			input:         "Jun 15, 2025",
			expectedDay:   15,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "European style 15 June 2025",
			input:         "15 June 2025",
			expectedDay:   15,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "European style abbreviated",
			input:         "15 Jun 2025",
			expectedDay:   15,
			expectedMonth: time.June,
			expectedYear:  2025,
		},

		// Ordinal numbers
		{
			name:          "ordinal 1st June 2025",
			input:         "1st June 2025",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "ordinal June 1st 2025",
			input:         "June 1st 2025",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},
		{
			name:          "ordinal 2nd January 2026",
			input:         "2nd January 2026",
			expectedDay:   2,
			expectedMonth: time.January,
			expectedYear:  2026,
		},
		{
			name:          "ordinal 23rd December 2025",
			input:         "23rd December 2025",
			expectedDay:   23,
			expectedMonth: time.December,
			expectedYear:  2025,
		},
		{
			name:          "ordinal with time 1st June 2025 15:30",
			input:         "1st June 2025 15:30",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},

		// RFC formats
		{
			name:          "RFC3339 format",
			input:         "2025-06-01T14:30:00Z",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},

		// Edge cases
		{
			name:          "whitespace around date",
			input:         "  June 1, 2025  ",
			expectedDay:   1,
			expectedMonth: time.June,
			expectedYear:  2025,
		},

		// Error cases
		{
			name:        "invalid format",
			input:       "not-a-date",
			expectedErr: true,
		},
		{
			name:        "invalid month",
			input:       "Foo 1, 2025",
			expectedErr: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAbsoluteDateTime(tt.input)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("parseAbsoluteDateTime(%q) expected error but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseAbsoluteDateTime(%q) unexpected error: %v", tt.input, err)
				return
			}

			// Parse the result to verify it's correct
			parsed, err := time.Parse("2006-01-02 15:04:05", result)
			if err != nil {
				t.Errorf("parseAbsoluteDateTime(%q) result %q is not a valid timestamp: %v", tt.input, result, err)
				return
			}

			if parsed.Day() != tt.expectedDay {
				t.Errorf("parseAbsoluteDateTime(%q) day = %d, want %d", tt.input, parsed.Day(), tt.expectedDay)
			}
			if parsed.Month() != tt.expectedMonth {
				t.Errorf("parseAbsoluteDateTime(%q) month = %v, want %v", tt.input, parsed.Month(), tt.expectedMonth)
			}
			if parsed.Year() != tt.expectedYear {
				t.Errorf("parseAbsoluteDateTime(%q) year = %d, want %d", tt.input, parsed.Year(), tt.expectedYear)
			}
		})
	}
}

func TestResolveStopTime(t *testing.T) {
	baseTime := time.Date(2025, 8, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		stopTime    string
		compileTime time.Time
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty stop time",
			stopTime:    "",
			compileTime: baseTime,
			expected:    "",
		},
		{
			name:        "absolute time standard format",
			stopTime:    "2025-12-31 23:59:59",
			compileTime: baseTime,
			expected:    "2025-12-31 23:59:59",
		},
		{
			name:        "absolute time readable format",
			stopTime:    "June 1, 2025",
			compileTime: baseTime,
			expected:    "2025-06-01 00:00:00",
		},
		{
			name:        "absolute time with ordinal",
			stopTime:    "1st June 2025",
			compileTime: baseTime,
			expected:    "2025-06-01 00:00:00",
		},
		{
			name:        "absolute time US format",
			stopTime:    "06/01/2025 15:30",
			compileTime: baseTime,
			expected:    "2025-06-01 15:30:00",
		},
		{
			name:        "absolute time European style",
			stopTime:    "15 June 2025 14:30",
			compileTime: baseTime,
			expected:    "2025-06-15 14:30:00",
		},
		{
			name:        "relative hours",
			stopTime:    "+25h",
			compileTime: baseTime,
			expected:    "2025-08-16 13:00:00",
		},
		{
			name:        "relative days",
			stopTime:    "+3d",
			compileTime: baseTime,
			expected:    "2025-08-18 12:00:00",
		},
		{
			name:        "relative complex",
			stopTime:    "+1d12h30m",
			compileTime: baseTime,
			expected:    "2025-08-17 00:30:00",
		},
		{
			name:        "invalid relative format",
			stopTime:    "+invalid",
			compileTime: baseTime,
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
		{
			name:        "invalid absolute format",
			stopTime:    "not-a-date",
			compileTime: baseTime,
			expectError: true,
			errorMsg:    "unable to parse date-time",
		},
		{
			name:        "relative with different base time",
			stopTime:    "+24h",
			compileTime: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected:    "2026-01-01 00:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveStopTime(tt.stopTime, tt.compileTime)

			if tt.expectError {
				if err == nil {
					t.Errorf("resolveStopTime(%q, %v) expected error but got none", tt.stopTime, tt.compileTime)
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("resolveStopTime(%q, %v) error = %v, want to contain %v", tt.stopTime, tt.compileTime, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolveStopTime(%q, %v) unexpected error: %v", tt.stopTime, tt.compileTime, err)
					return
				}
				if result != tt.expected {
					t.Errorf("resolveStopTime(%q, %v) = %v, want %v", tt.stopTime, tt.compileTime, result, tt.expected)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
