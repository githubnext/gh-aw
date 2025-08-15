package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TimeDelta represents a time duration that can be added to a base time
type TimeDelta struct {
	Hours   int
	Days    int
	Minutes int
}

// parseTimeDelta parses a relative time delta string like "+25h", "+3d", "+1d12h30m", etc.
// Supported formats:
// - +25h (25 hours)
// - +3d (3 days)
// - +30m (30 minutes)
// - +1d12h (1 day and 12 hours)
// - +2d5h30m (2 days, 5 hours, 30 minutes)
func parseTimeDelta(deltaStr string) (*TimeDelta, error) {
	if deltaStr == "" {
		return nil, fmt.Errorf("empty time delta")
	}

	// Must start with '+'
	if !strings.HasPrefix(deltaStr, "+") {
		return nil, fmt.Errorf("time delta must start with '+', got: %s", deltaStr)
	}

	// Remove the '+' prefix
	deltaStr = deltaStr[1:]

	if deltaStr == "" {
		return nil, fmt.Errorf("empty time delta after '+'")
	}

	// Parse components using regex
	// Pattern matches: number followed by d/h/m
	pattern := regexp.MustCompile(`(\d+)([dhm])`)
	matches := pattern.FindAllStringSubmatch(deltaStr, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d, +1d12h30m", deltaStr)
	}

	// Check that all characters are consumed by matches
	consumed := 0
	for _, match := range matches {
		consumed += len(match[0])
	}
	if consumed != len(deltaStr) {
		return nil, fmt.Errorf("invalid time delta format: +%s. Extra characters detected", deltaStr)
	}

	delta := &TimeDelta{}
	seenUnits := make(map[string]bool)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		valueStr := match[1]
		unit := match[2]

		// Check for duplicate units
		if seenUnits[unit] {
			return nil, fmt.Errorf("duplicate unit '%s' in time delta: +%s", unit, deltaStr)
		}
		seenUnits[unit] = true

		value, err := strconv.Atoi(valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid number '%s' in time delta: +%s", valueStr, deltaStr)
		}

		if value < 0 {
			return nil, fmt.Errorf("negative values not allowed in time delta: +%s", deltaStr)
		}

		switch unit {
		case "d":
			delta.Days = value
		case "h":
			delta.Hours = value
		case "m":
			delta.Minutes = value
		default:
			return nil, fmt.Errorf("unsupported time unit '%s' in time delta: +%s", unit, deltaStr)
		}
	}

	// Validate reasonable limits
	if delta.Days > 365 {
		return nil, fmt.Errorf("time delta too large: %d days exceeds maximum of 365 days", delta.Days)
	}
	if delta.Hours > 8760 { // 365 * 24
		return nil, fmt.Errorf("time delta too large: %d hours exceeds maximum of 8760 hours", delta.Hours)
	}
	if delta.Minutes > 525600 { // 365 * 24 * 60
		return nil, fmt.Errorf("time delta too large: %d minutes exceeds maximum of 525600 minutes", delta.Minutes)
	}

	return delta, nil
}

// toDuration converts a TimeDelta to a Go time.Duration
func (td *TimeDelta) toDuration() time.Duration {
	duration := time.Duration(td.Days) * 24 * time.Hour
	duration += time.Duration(td.Hours) * time.Hour
	duration += time.Duration(td.Minutes) * time.Minute
	return duration
}

// String returns a human-readable representation of the TimeDelta
func (td *TimeDelta) String() string {
	var parts []string
	if td.Days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", td.Days))
	}
	if td.Hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", td.Hours))
	}
	if td.Minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", td.Minutes))
	}
	if len(parts) == 0 {
		return "0m"
	}
	return "+" + strings.Join(parts, "")
}

// isRelativeStopTime checks if a stop-time value is a relative time delta
func isRelativeStopTime(stopTime string) bool {
	return strings.HasPrefix(stopTime, "+")
}

// parseAbsoluteDateTime parses various date-time formats and returns a standardized timestamp
func parseAbsoluteDateTime(dateTimeStr string) (string, error) {
	// Try multiple date-time formats in order of preference
	formats := []string{
		// Standard formats
		"2006-01-02 15:04:05",  // YYYY-MM-DD HH:MM:SS
		"2006-01-02T15:04:05",  // ISO 8601 without timezone
		"2006-01-02T15:04:05Z", // ISO 8601 UTC
		"2006-01-02 15:04",     // YYYY-MM-DD HH:MM
		"2006-01-02",           // YYYY-MM-DD (defaults to start of day)

		// Alternative formats
		"01/02/2006 15:04:05", // MM/DD/YYYY HH:MM:SS
		"01/02/2006 15:04",    // MM/DD/YYYY HH:MM
		"01/02/2006",          // MM/DD/YYYY
		"02/01/2006 15:04:05", // DD/MM/YYYY HH:MM:SS
		"02/01/2006 15:04",    // DD/MM/YYYY HH:MM
		"02/01/2006",          // DD/MM/YYYY

		// Readable formats
		"January 2, 2006 15:04:05", // January 2, 2006 15:04:05
		"January 2, 2006 15:04",    // January 2, 2006 15:04
		"January 2, 2006",          // January 2, 2006
		"Jan 2, 2006 15:04:05",     // Jan 2, 2006 15:04:05
		"Jan 2, 2006 15:04",        // Jan 2, 2006 15:04
		"Jan 2, 2006",              // Jan 2, 2006
		"2 January 2006 15:04:05",  // 2 January 2006 15:04:05
		"2 January 2006 15:04",     // 2 January 2006 15:04
		"2 January 2006",           // 2 January 2006
		"2 Jan 2006 15:04:05",      // 2 Jan 2006 15:04:05
		"2 Jan 2006 15:04",         // 2 Jan 2006 15:04
		"2 Jan 2006",               // 2 Jan 2006
		"January 2 2006 15:04:05",  // January 2 2006 15:04:05 (no comma)
		"January 2 2006 15:04",     // January 2 2006 15:04 (no comma)
		"January 2 2006",           // January 2 2006 (no comma)
		"Jan 2 2006 15:04:05",      // Jan 2 2006 15:04:05 (no comma)
		"Jan 2 2006 15:04",         // Jan 2 2006 15:04 (no comma)
		"Jan 2 2006",               // Jan 2 2006 (no comma)

		// RFC formats
		time.RFC3339, // 2006-01-02T15:04:05Z07:00
		time.RFC822,  // 02 Jan 06 15:04 MST
		time.RFC850,  // Monday, 02-Jan-06 15:04:05 MST
		time.RFC1123, // Mon, 02 Jan 2006 15:04:05 MST
	}

	// Clean up the input string
	dateTimeStr = strings.TrimSpace(dateTimeStr)

	// Handle ordinal numbers (1st, 2nd, 3rd, 4th, etc.)
	ordinalPattern := regexp.MustCompile(`\b(\d+)(st|nd|rd|th)\b`)
	dateTimeStr = ordinalPattern.ReplaceAllString(dateTimeStr, "$1")

	// Try to parse with each format
	for _, format := range formats {
		if parsed, err := time.Parse(format, dateTimeStr); err == nil {
			// Successfully parsed, convert to UTC and return in standard format
			return parsed.UTC().Format("2006-01-02 15:04:05"), nil
		}
	}

	// Try with more flexible ordinal handling - sometimes the ordinal removal creates double spaces
	normalizedStr := strings.ReplaceAll(dateTimeStr, "  ", " ")
	normalizedStr = strings.TrimSpace(normalizedStr)

	for _, format := range formats {
		if parsed, err := time.Parse(format, normalizedStr); err == nil {
			// Successfully parsed, convert to UTC and return in standard format
			return parsed.UTC().Format("2006-01-02 15:04:05"), nil
		}
	}

	// If none of the standard formats work, try some smart parsing
	// Handle formats like "June 1st 2025", "1st June 2025", etc.
	smartFormats := []string{
		"January 2nd 2006",
		"2nd January 2006",
		"Jan 2nd 2006",
		"2nd Jan 2006",
		"January 2nd 2006 15:04",
		"2nd January 2006 15:04",
		"Jan 2nd 2006 15:04",
		"2nd Jan 2006 15:04",
		"January 2nd 2006 15:04:05",
		"2nd January 2006 15:04:05",
		"Jan 2nd 2006 15:04:05",
		"2nd Jan 2006 15:04:05",
	}

	for _, format := range smartFormats {
		if parsed, err := time.Parse(format, dateTimeStr); err == nil {
			return parsed.UTC().Format("2006-01-02 15:04:05"), nil
		}
	}

	return "", fmt.Errorf("unable to parse date-time: %s. Supported formats include: YYYY-MM-DD HH:MM:SS, MM/DD/YYYY, January 2 2006, 1st June 2025, etc.", dateTimeStr)
}

// resolveStopTime resolves a stop-time value to an absolute timestamp
// If the stop-time is relative (starts with '+'), it calculates the absolute time
// from the compilation time. Otherwise, it parses the absolute time using various formats.
func resolveStopTime(stopTime string, compilationTime time.Time) (string, error) {
	if stopTime == "" {
		return "", nil
	}

	if isRelativeStopTime(stopTime) {
		// Parse the relative time delta
		delta, err := parseTimeDelta(stopTime)
		if err != nil {
			return "", err
		}

		// Calculate absolute time in UTC
		absoluteTime := compilationTime.UTC().Add(delta.toDuration())

		// Format in the expected format: "YYYY-MM-DD HH:MM:SS"
		return absoluteTime.Format("2006-01-02 15:04:05"), nil
	}

	// Parse absolute date-time with flexible format support
	return parseAbsoluteDateTime(stopTime)
}
