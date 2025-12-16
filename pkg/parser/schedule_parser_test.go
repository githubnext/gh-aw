package parser

import (
	"testing"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCron   string
		expectedOrig   string
		shouldError    bool
		errorSubstring string
	}{
		// Daily schedules
		{
			name:         "daily default time",
			input:        "daily",
			expectedCron: "FUZZY:DAILY * * *",
			expectedOrig: "daily",
		},
		{
			name:         "daily at 02:00",
			input:        "daily at 02:00",
			expectedCron: "0 2 * * *",
			expectedOrig: "daily at 02:00",
		},
		{
			name:         "daily at midnight",
			input:        "daily at midnight",
			expectedCron: "0 0 * * *",
			expectedOrig: "daily at midnight",
		},
		{
			name:         "daily at noon",
			input:        "daily at noon",
			expectedCron: "0 12 * * *",
			expectedOrig: "daily at noon",
		},
		{
			name:         "daily at 3pm",
			input:        "daily at 3pm",
			expectedCron: "0 15 * * *",
			expectedOrig: "daily at 3pm",
		},
		{
			name:         "daily at 1am",
			input:        "daily at 1am",
			expectedCron: "0 1 * * *",
			expectedOrig: "daily at 1am",
		},
		{
			name:         "daily at 12am (midnight)",
			input:        "daily at 12am",
			expectedCron: "0 0 * * *",
			expectedOrig: "daily at 12am",
		},
		{
			name:         "daily at 12pm (noon)",
			input:        "daily at 12pm",
			expectedCron: "0 12 * * *",
			expectedOrig: "daily at 12pm",
		},
		{
			name:         "daily at 11pm",
			input:        "daily at 11pm",
			expectedCron: "0 23 * * *",
			expectedOrig: "daily at 11pm",
		},
		{
			name:         "daily at 6am",
			input:        "daily at 6am",
			expectedCron: "0 6 * * *",
			expectedOrig: "daily at 6am",
		},

		// Weekly schedules
		{
			name:         "weekly on monday",
			input:        "weekly on monday",
			expectedCron: "0 0 * * 1",
			expectedOrig: "weekly on monday",
		},
		{
			name:         "weekly on monday at 06:30",
			input:        "weekly on monday at 06:30",
			expectedCron: "30 6 * * 1",
			expectedOrig: "weekly on monday at 06:30",
		},
		{
			name:         "weekly on sunday",
			input:        "weekly on sunday",
			expectedCron: "0 0 * * 0",
			expectedOrig: "weekly on sunday",
		},
		{
			name:         "weekly on friday at 17:00",
			input:        "weekly on friday at 17:00",
			expectedCron: "0 17 * * 5",
			expectedOrig: "weekly on friday at 17:00",
		},
		{
			name:         "weekly on saturday at midnight",
			input:        "weekly on saturday at midnight",
			expectedCron: "0 0 * * 6",
			expectedOrig: "weekly on saturday at midnight",
		},
		{
			name:         "daily at 02:00 utc+9",
			input:        "daily at 02:00 utc+9",
			expectedCron: "0 17 * * *",
			expectedOrig: "daily at 02:00 utc+9",
		},
		{
			name:         "daily at 14:00 utc-5",
			input:        "daily at 14:00 utc-5",
			expectedCron: "0 19 * * *",
			expectedOrig: "daily at 14:00 utc-5",
		},
		{
			name:         "daily at 09:30 utc+05:30",
			input:        "daily at 09:30 utc+05:30",
			expectedCron: "0 4 * * *",
			expectedOrig: "daily at 09:30 utc+05:30",
		},
		{
			name:         "weekly on monday at 08:00 utc+0",
			input:        "weekly on monday at 08:00 utc+0",
			expectedCron: "0 8 * * 1",
			expectedOrig: "weekly on monday at 08:00 utc+0",
		},
		{
			name:         "monthly on 15 at 12:00 utc-8",
			input:        "monthly on 15 at 12:00 utc-8",
			expectedCron: "0 20 15 * *",
			expectedOrig: "monthly on 15 at 12:00 utc-8",
		},
		{
			name:         "daily at 3pm utc+9",
			input:        "daily at 3pm utc+9",
			expectedCron: "0 6 * * *",
			expectedOrig: "daily at 3pm utc+9",
		},
		{
			name:         "daily at 9am utc-5",
			input:        "daily at 9am utc-5",
			expectedCron: "0 14 * * *",
			expectedOrig: "daily at 9am utc-5",
		},
		{
			name:         "daily at 12pm utc+1",
			input:        "daily at 12pm utc+1",
			expectedCron: "0 11 * * *",
			expectedOrig: "daily at 12pm utc+1",
		},
		{
			name:         "daily at 12am utc-8",
			input:        "daily at 12am utc-8",
			expectedCron: "0 8 * * *",
			expectedOrig: "daily at 12am utc-8",
		},
		{
			name:         "daily at 11pm utc+05:30",
			input:        "daily at 11pm utc+05:30",
			expectedCron: "30 17 * * *",
			expectedOrig: "daily at 11pm utc+05:30",
		},
		{
			name:         "weekly on monday at 8am utc+9",
			input:        "weekly on monday at 8am utc+9",
			expectedCron: "0 23 * * 1",
			expectedOrig: "weekly on monday at 8am utc+9",
		},
		{
			name:         "weekly on friday at 6pm utc-7",
			input:        "weekly on friday at 6pm utc-7",
			expectedCron: "0 1 * * 5",
			expectedOrig: "weekly on friday at 6pm utc-7",
		},
		{
			name:         "monthly on 15 at 10am utc+2",
			input:        "monthly on 15 at 10am utc+2",
			expectedCron: "0 8 15 * *",
			expectedOrig: "monthly on 15 at 10am utc+2",
		},
		{
			name:         "monthly on 1 at 7pm utc-3",
			input:        "monthly on 1 at 7pm utc-3",
			expectedCron: "0 22 1 * *",
			expectedOrig: "monthly on 1 at 7pm utc-3",
		},
		{
			name:         "weekly on friday at 5pm",
			input:        "weekly on friday at 5pm",
			expectedCron: "0 17 * * 5",
			expectedOrig: "weekly on friday at 5pm",
		},
		{
			name:         "monthly on 15 at 9am",
			input:        "monthly on 15 at 9am",
			expectedCron: "0 9 15 * *",
			expectedOrig: "monthly on 15 at 9am",
		},

		// Monthly schedules
		{
			name:         "monthly on 1st",
			input:        "monthly on 1",
			expectedCron: "0 0 1 * *",
			expectedOrig: "monthly on 1",
		},
		{
			name:         "monthly on 15th",
			input:        "monthly on 15",
			expectedCron: "0 0 15 * *",
			expectedOrig: "monthly on 15",
		},
		{
			name:         "monthly on 15th at 09:00",
			input:        "monthly on 15 at 09:00",
			expectedCron: "0 9 15 * *",
			expectedOrig: "monthly on 15 at 09:00",
		},
		{
			name:         "monthly on 31st",
			input:        "monthly on 31",
			expectedCron: "0 0 31 * *",
			expectedOrig: "monthly on 31",
		},

		// Interval schedules
		{
			name:         "every 10 minutes",
			input:        "every 10 minutes",
			expectedCron: "*/10 * * * *",
			expectedOrig: "every 10 minutes",
		},
		{
			name:         "every 5 minutes",
			input:        "every 5 minutes",
			expectedCron: "*/5 * * * *",
			expectedOrig: "every 5 minutes",
		},
		{
			name:         "every 30 minutes",
			input:        "every 30 minutes",
			expectedCron: "*/30 * * * *",
			expectedOrig: "every 30 minutes",
		},
		{
			name:         "every 1 hour",
			input:        "every 1 hour",
			expectedCron: "0 */1 * * *",
			expectedOrig: "every 1 hour",
		},
		{
			name:         "every 2 hours",
			input:        "every 2 hours",
			expectedCron: "0 */2 * * *",
			expectedOrig: "every 2 hours",
		},
		{
			name:         "every 6 hours",
			input:        "every 6 hours",
			expectedCron: "0 */6 * * *",
			expectedOrig: "every 6 hours",
		},
		{
			name:         "every 12 hours",
			input:        "every 12 hours",
			expectedCron: "0 */12 * * *",
			expectedOrig: "every 12 hours",
		},

		// Short duration formats (like stop-after)
		{
			name:         "every 30m",
			input:        "every 30m",
			expectedCron: "*/30 * * * *",
			expectedOrig: "every 30m",
		},
		{
			name:         "every 1h",
			input:        "every 1h",
			expectedCron: "0 */1 * * *",
			expectedOrig: "every 1h",
		},
		{
			name:         "every 2h",
			input:        "every 2h",
			expectedCron: "0 */2 * * *",
			expectedOrig: "every 2h",
		},
		{
			name:         "every 6h",
			input:        "every 6h",
			expectedCron: "0 */6 * * *",
			expectedOrig: "every 6h",
		},
		{
			name:         "every 1d",
			input:        "every 1d",
			expectedCron: "0 0 * * *",
			expectedOrig: "every 1d",
		},
		{
			name:         "every 2d",
			input:        "every 2d",
			expectedCron: "0 0 */2 * *",
			expectedOrig: "every 2d",
		},
		{
			name:         "every 1w",
			input:        "every 1w",
			expectedCron: "0 0 * * 0",
			expectedOrig: "every 1w",
		},
		{
			name:         "every 2w",
			input:        "every 2w",
			expectedCron: "0 0 */14 * *",
			expectedOrig: "every 2w",
		},
		{
			name:         "every 1mo",
			input:        "every 1mo",
			expectedCron: "0 0 1 * *",
			expectedOrig: "every 1mo",
		},
		{
			name:         "every 2mo",
			input:        "every 2mo",
			expectedCron: "0 0 1 */2 *",
			expectedOrig: "every 2mo",
		},

		// Case insensitivity
		{
			name:         "DAILY uppercase",
			input:        "DAILY",
			expectedCron: "FUZZY:DAILY * * *",
			expectedOrig: "DAILY",
		},
		{
			name:         "Weekly On Monday mixed case",
			input:        "Weekly On Monday",
			expectedCron: "0 0 * * 1",
			expectedOrig: "Weekly On Monday",
		},

		// Already cron expressions (should pass through)
		{
			name:         "existing cron expression",
			input:        "0 9 * * 1",
			expectedCron: "0 9 * * 1",
			expectedOrig: "",
		},
		{
			name:         "complex cron expression",
			input:        "*/15 * * * *",
			expectedCron: "*/15 * * * *",
			expectedOrig: "",
		},
		{
			name:         "cron with ranges",
			input:        "0 14 * * 1-5",
			expectedCron: "0 14 * * 1-5",
			expectedOrig: "",
		},

		// Error cases
		{
			name:           "empty string",
			input:          "",
			shouldError:    true,
			errorSubstring: "cannot be empty",
		},
		{
			name:           "interval with time conflict",
			input:          "every 10 minutes at 06:00",
			shouldError:    true,
			errorSubstring: "cannot have 'at time'",
		},
		{
			name:           "invalid interval number",
			input:          "every abc minutes",
			shouldError:    true,
			errorSubstring: "invalid interval",
		},
		{
			name:           "invalid interval unit",
			input:          "every 10 days",
			shouldError:    true,
			errorSubstring: "unsupported interval unit",
		},
		{
			name:           "weekly without on",
			input:          "weekly monday",
			shouldError:    true,
			errorSubstring: "requires 'on <weekday>'",
		},
		{
			name:           "weekly invalid weekday",
			input:          "weekly on funday",
			shouldError:    true,
			errorSubstring: "invalid weekday",
		},
		{
			name:           "monthly without on",
			input:          "monthly 15",
			shouldError:    true,
			errorSubstring: "requires 'on <day>'",
		},
		{
			name:           "monthly invalid day",
			input:          "monthly on 32",
			shouldError:    true,
			errorSubstring: "invalid day of month",
		},
		{
			name:           "monthly day out of range",
			input:          "monthly on 0",
			shouldError:    true,
			errorSubstring: "invalid day of month",
		},
		{
			name:           "unsupported schedule type",
			input:          "hourly",
			shouldError:    true,
			errorSubstring: "unsupported schedule type",
		},
		{
			name:           "negative interval",
			input:          "every -5 minutes",
			shouldError:    true,
			errorSubstring: "invalid interval",
		},
		{
			name:           "zero interval",
			input:          "every 0 minutes",
			shouldError:    true,
			errorSubstring: "invalid interval",
		},
		// Minimum duration validation (5 minutes)
		{
			name:           "interval below minimum - 1m",
			input:          "every 1m",
			shouldError:    true,
			errorSubstring: "minimum schedule interval is 5 minutes",
		},
		{
			name:           "interval below minimum - 2 minutes",
			input:          "every 2 minutes",
			shouldError:    true,
			errorSubstring: "minimum schedule interval is 5 minutes",
		},
		{
			name:           "interval below minimum - 4m",
			input:          "every 4m",
			shouldError:    true,
			errorSubstring: "minimum schedule interval is 5 minutes",
		},
		{
			name:         "interval at minimum - 5m",
			input:        "every 5m",
			expectedCron: "*/5 * * * *",
			expectedOrig: "every 5m",
		},
		{
			name:         "interval at minimum - 5 minutes",
			input:        "every 5 minutes",
			expectedCron: "*/5 * * * *",
			expectedOrig: "every 5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, orig, err := ParseSchedule(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorSubstring)
					return
				}
				if tt.errorSubstring != "" && !containsSubstring(err.Error(), tt.errorSubstring) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cron != tt.expectedCron {
				t.Errorf("expected cron '%s', got '%s'", tt.expectedCron, cron)
			}

			if orig != tt.expectedOrig {
				t.Errorf("expected original '%s', got '%s'", tt.expectedOrig, orig)
			}
		})
	}
}

func TestMapWeekday(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sunday", "0"},
		{"Sunday", "0"},
		{"sun", "0"},
		{"monday", "1"},
		{"Monday", "1"},
		{"mon", "1"},
		{"tuesday", "2"},
		{"tue", "2"},
		{"wednesday", "3"},
		{"wed", "3"},
		{"thursday", "4"},
		{"thu", "4"},
		{"friday", "5"},
		{"fri", "5"},
		{"saturday", "6"},
		{"sat", "6"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapWeekday(tt.input)
			if result != tt.expected {
				t.Errorf("mapWeekday(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input        string
		expectedMin  string
		expectedHour string
	}{
		{"midnight", "0", "0"},
		{"noon", "0", "12"},
		{"00:00", "0", "0"},
		{"12:00", "0", "12"},
		{"06:30", "30", "6"},
		{"23:59", "59", "23"},
		{"09:15", "15", "9"},
		// AM/PM formats
		{"1am", "0", "1"},
		{"3pm", "0", "15"},
		{"12am", "0", "0"},  // midnight
		{"12pm", "0", "12"}, // noon
		{"11pm", "0", "23"},
		{"6am", "0", "6"},
		{"9am", "0", "9"},
		{"5pm", "0", "17"},
		{"10pm", "0", "22"},
		// Invalid formats fall back to defaults
		{"invalid", "0", "0"},
		{"25:00", "0", "0"},
		{"12:60", "0", "0"},
		{"12", "0", "0"},
		{"13pm", "0", "0"}, // invalid hour for 12-hour format
		{"0am", "0", "0"},  // invalid hour for 12-hour format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			min, hour := parseTime(tt.input)
			if min != tt.expectedMin || hour != tt.expectedHour {
				t.Errorf("parseTime(%q) = (%q, %q), want (%q, %q)",
					tt.input, min, hour, tt.expectedMin, tt.expectedHour)
			}
		})
	}
}

func TestIsCronExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0 0 * * *", true},
		{"*/15 * * * *", true},
		{"0 14 * * 1-5", true},
		{"30 6 * * 1", true},
		{"0 12 25 12 *", true},
		{"daily", false},
		{"weekly on monday", false},
		{"every 10 minutes", false},
		{"0 0 * *", false},         // Too few fields
		{"0 0 * * * *", false},     // Too many fields
		{"0 0 * * * extra", false}, // Extra tokens
		{"invalid cron expression", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isCronExpression(tt.input)
			if result != tt.expected {
				t.Errorf("isCronExpression(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// containsSubstring checks if s contains substr (case-insensitive)
func containsSubstring(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func TestIsDailyCron(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0 0 * * *", true},
		{"30 14 * * *", true},
		{"0 9 * * *", true},
		{"*/15 * * * *", false},  // interval
		{"0 0 1 * *", false},     // monthly
		{"0 0 * * 1", false},     // weekly
		{"0 14 * * 1-5", false},  // weekdays only
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsDailyCron(tt.input)
			if result != tt.expected {
				t.Errorf("IsDailyCron(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsFuzzyCron(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"FUZZY:DAILY * * *", true},
		{"0 0 * * *", false},
		{"daily", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsFuzzyCron(tt.input)
			if result != tt.expected {
				t.Errorf("IsFuzzyCron(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestScatterSchedule(t *testing.T) {
	tests := []struct {
		name               string
		fuzzyCron          string
		workflowIdentifier string
		expectError        bool
	}{
		{
			name:               "valid fuzzy daily",
			fuzzyCron:          "FUZZY:DAILY * * *",
			workflowIdentifier: "workflow1",
			expectError:        false,
		},
		{
			name:               "not a fuzzy cron",
			fuzzyCron:          "0 0 * * *",
			workflowIdentifier: "workflow1",
			expectError:        true,
		},
		{
			name:               "invalid fuzzy pattern",
			fuzzyCron:          "FUZZY:INVALID",
			workflowIdentifier: "workflow1",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ScatterSchedule(tt.fuzzyCron, tt.workflowIdentifier)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Check that result is a valid cron expression
			if !isCronExpression(result) {
				t.Errorf("ScatterSchedule returned invalid cron: %s", result)
			}
			// Check that result is daily pattern
			if !IsDailyCron(result) {
				t.Errorf("ScatterSchedule returned non-daily cron: %s", result)
			}
		})
	}
}

func TestScatterScheduleDeterministic(t *testing.T) {
	// Test that scattering is deterministic - same input produces same output
	workflows := []string{"workflow-a", "workflow-b", "workflow-c", "workflow-a"}
	
	results := make([]string, len(workflows))
	for i, wf := range workflows {
		result, err := ScatterSchedule("FUZZY:DAILY * * *", wf)
		if err != nil {
			t.Fatalf("unexpected error for workflow %s: %v", wf, err)
		}
		results[i] = result
	}
	
	// workflow-a should produce the same result both times
	if results[0] != results[3] {
		t.Errorf("ScatterSchedule not deterministic: workflow-a produced %s and %s", results[0], results[3])
	}
	
	// Different workflows should produce different results (with high probability)
	if results[0] == results[1] && results[1] == results[2] {
		t.Errorf("ScatterSchedule produced identical results for all workflows: %s", results[0])
	}
}
