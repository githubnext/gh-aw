//go:build !integration

package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				require.Error(t, err, "parseTimeDelta(%q) should return an error", tt.input)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "parseTimeDelta(%q) error message mismatch", tt.input)
				}
			} else {
				require.NoError(t, err, "parseTimeDelta(%q) unexpected error", tt.input)
				require.NotNil(t, result, "parseTimeDelta(%q) returned nil result", tt.input)
				assert.Equal(t, tt.expected, result, "parseTimeDelta(%q) result mismatch", tt.input)
			}
		})
	}
}

func TestParseTimeDeltaForStopAfter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *TimeDelta
		expectError bool
		errorMsg    string
	}{
		// Valid cases (no minutes allowed)
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
			name:     "weeks only",
			input:    "+1w",
			expected: &TimeDelta{Weeks: 1},
		},
		{
			name:     "months only",
			input:    "+1mo",
			expected: &TimeDelta{Months: 1},
		},
		{
			name:     "days and hours",
			input:    "+1d12h",
			expected: &TimeDelta{Days: 1, Hours: 12},
		},
		{
			name:     "all units except minutes",
			input:    "+1mo2w3d5h",
			expected: &TimeDelta{Months: 1, Weeks: 2, Days: 3, Hours: 5},
		},
		{
			name:     "different order",
			input:    "+5h2d",
			expected: &TimeDelta{Days: 2, Hours: 5},
		},

		// Error cases - minutes not allowed
		{
			name:        "minutes only",
			input:       "+30m",
			expectError: true,
			errorMsg:    "minute unit 'm' is not allowed for stop-after",
		},
		{
			name:        "days hours and minutes",
			input:       "+2d5h30m",
			expectError: true,
			errorMsg:    "minute unit 'm' is not allowed for stop-after",
		},
		{
			name:        "complex with minutes",
			input:       "+1d12h30m",
			expectError: true,
			errorMsg:    "minute unit 'm' is not allowed for stop-after",
		},
		{
			name:        "only minutes at end",
			input:       "+1d5m",
			expectError: true,
			errorMsg:    "minute unit 'm' is not allowed for stop-after",
		},

		// Other error cases (inherited from parseTimeDelta)
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
			name:        "too many days",
			input:       "+400d",
			expectError: true,
			errorMsg:    "time delta too large: 400 days exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeDeltaForStopAfter(tt.input)

			if tt.expectError {
				require.Error(t, err, "parseTimeDeltaForStopAfter(%q) should return an error", tt.input)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "parseTimeDeltaForStopAfter(%q) error message mismatch", tt.input)
				}
			} else {
				require.NoError(t, err, "parseTimeDeltaForStopAfter(%q) unexpected error", tt.input)
				require.NotNil(t, result, "parseTimeDeltaForStopAfter(%q) returned nil result", tt.input)
				assert.Equal(t, tt.expected, result, "parseTimeDeltaForStopAfter(%q) result mismatch", tt.input)
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
			assert.Equal(t, tt.expected, result, "TimeDelta.String() mismatch for delta %+v", tt.delta)
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
			assert.Equal(t, tt.expected, result, "isRelativeStopTime(%q) mismatch", tt.input)
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
				require.Error(t, err, "parseAbsoluteDateTime(%q) should return an error", tt.input)
				return
			}

			require.NoError(t, err, "parseAbsoluteDateTime(%q) unexpected error", tt.input)

			// Parse the result to verify it's correct
			parsed, err := time.Parse("2006-01-02 15:04:05", result)
			require.NoError(t, err, "parseAbsoluteDateTime(%q) result %q is not a valid timestamp", tt.input, result)

			assert.Equal(t, tt.expectedDay, parsed.Day(), "parseAbsoluteDateTime(%q) day mismatch", tt.input)
			assert.Equal(t, tt.expectedMonth, parsed.Month(), "parseAbsoluteDateTime(%q) month mismatch", tt.input)
			assert.Equal(t, tt.expectedYear, parsed.Year(), "parseAbsoluteDateTime(%q) year mismatch", tt.input)
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
			expectError: true,
			errorMsg:    "minute unit 'm' is not allowed for stop-after",
		},
		{
			name:        "relative complex without minutes",
			stopTime:    "+1d12h",
			compileTime: baseTime,
			expected:    "2025-08-17 00:00:00",
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
		{
			name:        "relative weeks",
			stopTime:    "+1w",
			compileTime: baseTime,
			expected:    "2025-08-22 12:00:00",
		},
		{
			name:        "relative months",
			stopTime:    "+1mo",
			compileTime: baseTime,
			expected:    "2025-09-15 12:00:00",
		},
		{
			name:        "relative months and weeks",
			stopTime:    "+1mo2w",
			compileTime: baseTime,
			expected:    "2025-09-29 12:00:00",
		},
		{
			name:        "relative complex with months",
			stopTime:    "+1mo1w2d5h",
			compileTime: baseTime,
			expected:    "2025-09-24 17:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveStopTime(tt.stopTime, tt.compileTime)

			if tt.expectError {
				require.Error(t, err, "resolveStopTime(%q) should return an error", tt.stopTime)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "resolveStopTime(%q) error message mismatch", tt.stopTime)
				}
			} else {
				require.NoError(t, err, "resolveStopTime(%q) unexpected error", tt.stopTime)
				assert.Equal(t, tt.expected, result, "resolveStopTime(%q) result mismatch", tt.stopTime)
			}
		})
	}
}

func TestIsRelativeDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "negative delta",
			input:    "-1d",
			expected: true,
		},
		{
			name:     "positive delta",
			input:    "+3d",
			expected: true,
		},
		{
			name:     "absolute date",
			input:    "2024-01-01",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "just minus",
			input:    "-",
			expected: true,
		},
		{
			name:     "just plus",
			input:    "+",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRelativeDate(tt.input)
			assert.Equal(t, tt.expected, result, "isRelativeDate(%q) mismatch", tt.input)
		})
	}
}

func TestParseRelativeDate(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectDelta   bool
		expectNeg     bool
		expectedDelta *TimeDelta
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "negative 1 day",
			input:         "-1d",
			expectDelta:   true,
			expectNeg:     true,
			expectedDelta: &TimeDelta{Days: 1},
		},
		{
			name:          "negative 1 week",
			input:         "-1w",
			expectDelta:   true,
			expectNeg:     true,
			expectedDelta: &TimeDelta{Weeks: 1},
		},
		{
			name:          "negative 1 month",
			input:         "-1mo",
			expectDelta:   true,
			expectNeg:     true,
			expectedDelta: &TimeDelta{Months: 1},
		},
		{
			name:          "positive 3 days",
			input:         "+3d",
			expectDelta:   true,
			expectNeg:     false,
			expectedDelta: &TimeDelta{Days: 3},
		},
		{
			name:          "complex negative delta",
			input:         "-1mo2w3d",
			expectDelta:   true,
			expectNeg:     true,
			expectedDelta: &TimeDelta{Months: 1, Weeks: 2, Days: 3},
		},
		{
			name:        "absolute date",
			input:       "2024-01-01",
			expectDelta: false,
			expectNeg:   false,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errorMsg:    "empty date string",
		},
		{
			name:        "invalid negative format",
			input:       "-invalid",
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta, isNeg, err := parseRelativeDate(tt.input)

			if tt.expectError {
				require.Error(t, err, "parseRelativeDate(%q) should return an error", tt.input)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "parseRelativeDate(%q) error message mismatch", tt.input)
				}
				return
			}

			require.NoError(t, err, "parseRelativeDate(%q) unexpected error", tt.input)

			if !tt.expectDelta {
				assert.Nil(t, delta, "parseRelativeDate(%q) expected no delta but got one", tt.input)
				return
			}

			require.NotNil(t, delta, "parseRelativeDate(%q) expected delta but got nil", tt.input)
			assert.Equal(t, tt.expectNeg, isNeg, "parseRelativeDate(%q) isNegative mismatch", tt.input)
			assert.Equal(t, tt.expectedDelta, delta, "parseRelativeDate(%q) delta mismatch", tt.input)
		})
	}
}

func TestResolveRelativeDate(t *testing.T) {
	baseTime := time.Date(2024, 8, 15, 12, 0, 0, 0, time.UTC) // Thursday, August 15, 2024 12:00:00 UTC

	tests := []struct {
		name        string
		input       string
		baseTime    time.Time
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "empty string",
			input:    "",
			baseTime: baseTime,
			expected: "",
		},
		{
			name:     "absolute date unchanged",
			input:    "2024-01-01",
			baseTime: baseTime,
			expected: "2024-01-01",
		},
		{
			name:     "negative 1 day (returns timestamp)",
			input:    "-1d",
			baseTime: baseTime,
			expected: "2024-08-14T12:00:00Z", // Full timestamp now
		},
		{
			name:     "negative 1 week (returns timestamp)",
			input:    "-1w",
			baseTime: baseTime,
			expected: "2024-08-08T12:00:00Z", // Full timestamp now
		},
		{
			name:     "negative 1 month (returns timestamp)",
			input:    "-1mo",
			baseTime: baseTime,
			expected: "2024-07-15T12:00:00Z", // Full timestamp now
		},
		{
			name:     "positive 3 days (returns timestamp)",
			input:    "+3d",
			baseTime: baseTime,
			expected: "2024-08-18T12:00:00Z", // Full timestamp now
		},
		{
			name:     "complex negative delta (returns timestamp)",
			input:    "-1mo2w3d",
			baseTime: baseTime,
			expected: "2024-06-28T12:00:00Z", // Full timestamp now
		},
		{
			name:     "negative 24 hours (returns timestamp)",
			input:    "-24h",
			baseTime: baseTime,
			expected: "2024-08-14T12:00:00Z",
		},
		{
			name:     "negative 2 hours (returns timestamp)",
			input:    "-2h",
			baseTime: baseTime,
			expected: "2024-08-15T10:00:00Z",
		},
		{
			name:     "negative 1 day 12 hours (returns timestamp)",
			input:    "-1d12h",
			baseTime: baseTime,
			expected: "2024-08-14T00:00:00Z",
		},
		{
			name:     "negative 30 minutes (returns timestamp)",
			input:    "-30m",
			baseTime: baseTime,
			expected: "2024-08-15T11:30:00Z",
		},
		{
			name:     "complex with hours (returns timestamp)",
			input:    "-2w3d5h",
			baseTime: baseTime,
			expected: "2024-07-29T07:00:00Z",
		},
		{
			name:     "edge case: late evening -24h",
			input:    "-24h",
			baseTime: time.Date(2024, 8, 15, 23, 45, 0, 0, time.UTC),
			expected: "2024-08-14T23:45:00Z",
		},
		{
			name:     "edge case: early morning -24h",
			input:    "-24h",
			baseTime: time.Date(2024, 8, 15, 0, 15, 0, 0, time.UTC),
			expected: "2024-08-14T00:15:00Z",
		},
		{
			name:        "invalid relative format",
			input:       "-invalid",
			baseTime:    baseTime,
			expectError: true,
			errorMsg:    "invalid time delta format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveRelativeDate(tt.input, tt.baseTime)

			if tt.expectError {
				require.Error(t, err, "ResolveRelativeDate(%q) should return an error", tt.input)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "ResolveRelativeDate(%q) error message mismatch", tt.input)
				}
				return
			}

			require.NoError(t, err, "ResolveRelativeDate(%q) unexpected error", tt.input)
			assert.Equal(t, tt.expected, result, "ResolveRelativeDate(%q) result mismatch", tt.input)
		})
	}
}
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
			assert.Equal(t, tt.expected, result, "parseRelativeTimeSpec(%q) mismatch", tt.input)
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
		// Boolean false explicitly disables expiration
		{
			name:     "false disables expiration",
			config:   map[string]any{"expires": false},
			expected: -1, // -1 indicates explicitly disabled
		},
		// Boolean true is invalid
		{
			name:     "true is invalid",
			config:   map[string]any{"expires": true},
			expected: 0, // true is not a valid expires value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseExpiresFromConfig(tt.config)
			assert.Equal(t, tt.expected, result, "parseExpiresFromConfig(%v) mismatch", tt.config)
		})
	}
}
