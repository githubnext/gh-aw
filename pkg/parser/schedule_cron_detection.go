package parser

import (
	"regexp"
	"strings"
)

// This file contains cron expression detection and classification functions.
// These pure functions analyze cron strings and determine their type without
// depending on parser state.

// IsDailyCron checks if a cron expression represents a daily schedule at a fixed time
// (e.g., "0 0 * * *", "30 14 * * *", etc.)
func IsDailyCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	// Daily pattern: minute hour * * *
	// The minute and hour must be specific values (numbers), not wildcards
	// The day-of-month (3rd field) and month (4th field) must be "*"
	// The day-of-week (5th field) must be "*"

	// Check if minute and hour are numeric (not wildcards)
	minute := fields[0]
	hour := fields[1]

	// Minute and hour should be digits only (no *, /, -, ,)
	for _, ch := range minute {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	for _, ch := range hour {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return fields[2] == "*" && fields[3] == "*" && fields[4] == "*"
}

// IsHourlyCron checks if a cron expression represents an hourly interval with a fixed minute
// (e.g., "0 */1 * * *", "30 */2 * * *", etc.)
func IsHourlyCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	// Hourly pattern: minute */N * * * or minute *N * * *
	// The minute must be a specific value (number), not a wildcard
	// The hour must be an interval pattern (*/N)

	minute := fields[0]
	hour := fields[1]

	// Minute should be digits only (no *, /, -, ,)
	for _, ch := range minute {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	// Hour should be an interval pattern like */N
	if !strings.HasPrefix(hour, "*/") {
		return false
	}

	// Check remaining fields are wildcards
	return fields[2] == "*" && fields[3] == "*" && fields[4] == "*"
}

// IsWeeklyCron checks if a cron expression represents a weekly schedule at a fixed time
// (e.g., "0 0 * * 1", "30 14 * * 5", etc.)
func IsWeeklyCron(cron string) bool {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return false
	}
	// Weekly pattern: minute hour * * DOW
	// The minute and hour must be specific values (numbers), not wildcards
	// The day-of-month (3rd field) and month (4th field) must be "*"
	// The day-of-week (5th field) must be a specific day (0-6)

	// Check if minute and hour are numeric (not wildcards)
	minute := fields[0]
	hour := fields[1]

	// Minute and hour should be digits only (no *, /, -, ,)
	for _, ch := range minute {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	for _, ch := range hour {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	// Check day-of-month and month are wildcards
	if fields[2] != "*" || fields[3] != "*" {
		return false
	}

	// Check day-of-week is a specific day (0-6)
	dow := fields[4]
	for _, ch := range dow {
		if ch < '0' || ch > '6' {
			return false
		}
	}

	return true
}

// IsFuzzyCron checks if a cron expression is a fuzzy schedule placeholder
func IsFuzzyCron(cron string) bool {
	return strings.HasPrefix(cron, "FUZZY:")
}

// IsCronExpression checks if the input looks like a valid cron expression
// A valid cron expression has exactly 5 fields (minute, hour, day of month, month, day of week)
func IsCronExpression(input string) bool {
	// A cron expression has exactly 5 fields
	fields := strings.Fields(input)
	if len(fields) != 5 {
		return false
	}

	// Each field should match cron syntax (numbers, *, /, -, ,)
	cronFieldPattern := regexp.MustCompile(`^[\d\*\-/,]+$`)
	for _, field := range fields {
		if !cronFieldPattern.MatchString(field) {
			return false
		}
	}

	return true
}
