package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pre-compiled regexes for time parsing (performance optimization)
var (
	timeDeltaPattern = regexp.MustCompile(`(\d+)(mo|w|d|h|m)`)
	ordinalPattern   = regexp.MustCompile(`\b(\d+)(st|nd|rd|th)\b`)
)

// TimeDelta represents a time duration that can be added to a base time
type TimeDelta struct {
	Hours   int
	Days    int
	Minutes int
	Weeks   int
	Months  int
}

// parseTimeDelta parses a relative time delta string like "+25h", "+3d", "+1w", "+1mo", "+1d12h30m", etc.
// Supported formats:
// - +25h (25 hours)
// - +3d (3 days)
// - +1w (1 week)
// - +1mo (1 month)
// - +30m (30 minutes)
// - +1d12h (1 day and 12 hours)
// - +2d5h30m (2 days, 5 hours, 30 minutes)
// - +1mo2w3d (1 month, 2 weeks, 3 days)
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
	// Pattern matches: number followed by mo/w/d/h/m (months, weeks, days, hours, minutes)
	matches := timeDeltaPattern.FindAllStringSubmatch(deltaStr, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d, +1w, +1mo, +1d12h30m", deltaStr)
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
		case "mo":
			delta.Months = value
		case "w":
			delta.Weeks = value
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
	if delta.Months > 12 {
		return nil, fmt.Errorf("time delta too large: %d months exceeds maximum of 12 months", delta.Months)
	}
	if delta.Weeks > 52 {
		return nil, fmt.Errorf("time delta too large: %d weeks exceeds maximum of 52 weeks", delta.Weeks)
	}
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

// String returns a human-readable representation of the TimeDelta
func (td *TimeDelta) String() string {
	var parts []string
	if td.Months > 0 {
		parts = append(parts, fmt.Sprintf("%dmo", td.Months))
	}
	if td.Weeks > 0 {
		parts = append(parts, fmt.Sprintf("%dw", td.Weeks))
	}
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

	return "", fmt.Errorf("unable to parse date-time: %s. Supported formats include: YYYY-MM-DD HH:MM:SS, MM/DD/YYYY, January 2 2006, 1st June 2025, etc", dateTimeStr)
}

// isRelativeDate checks if a date string is a relative time delta (starts with + or -)
func isRelativeDate(dateStr string) bool {
	return strings.HasPrefix(dateStr, "+") || strings.HasPrefix(dateStr, "-")
}

// parseRelativeDate parses a relative date string like "-1d", "-1w", "-1mo", "+3d", etc.
// Supports both positive (+) and negative (-) deltas for log filtering use cases.
// Supported formats:
// - -1d (1 day ago)
// - -1w (1 week ago)
// - -1mo (1 month ago)
// - +3d (3 days from now)
// - -2w3d (2 weeks and 3 days ago)
func parseRelativeDate(dateStr string) (*TimeDelta, bool, error) {
	if dateStr == "" {
		return nil, false, fmt.Errorf("empty date string")
	}

	// Check if it's a relative date
	if !isRelativeDate(dateStr) {
		return nil, false, nil // Not a relative date, caller should handle as absolute
	}

	// Determine if it's negative (going backwards in time)
	isNegative := strings.HasPrefix(dateStr, "-")

	// Convert to positive format for parsing with existing parseTimeDelta
	var deltaStr string
	if isNegative {
		deltaStr = "+" + dateStr[1:] // Replace - with +
	} else {
		deltaStr = dateStr // Already has +
	}

	// Parse using existing function
	delta, err := parseTimeDelta(deltaStr)
	if err != nil {
		return nil, false, err
	}

	return delta, isNegative, nil
}

// ResolveRelativeDate resolves a relative date string to an absolute timestamp
// suitable for use with GitHub CLI.
// If the date string is not relative, it returns the original string.
//
// Returns a full ISO 8601 timestamp (YYYY-MM-DDTHH:MM:SSZ) for precise filtering.
func ResolveRelativeDate(dateStr string, baseTime time.Time) (string, error) {
	if dateStr == "" {
		return "", nil
	}

	// Check if it's a relative date
	delta, isNegative, err := parseRelativeDate(dateStr)
	if err != nil {
		return "", err
	}
	if delta == nil {
		// Not a relative date, return as-is
		return dateStr, nil
	}

	// Calculate the absolute time using precise calculation
	// Always use AddDate for months, weeks, and days for maximum precision
	absoluteTime := baseTime.UTC()
	if isNegative {
		absoluteTime = absoluteTime.AddDate(0, -delta.Months, -delta.Weeks*7-delta.Days)
		absoluteTime = absoluteTime.Add(-time.Duration(delta.Hours)*time.Hour - time.Duration(delta.Minutes)*time.Minute)
	} else {
		absoluteTime = absoluteTime.AddDate(0, delta.Months, delta.Weeks*7+delta.Days)
		absoluteTime = absoluteTime.Add(time.Duration(delta.Hours)*time.Hour + time.Duration(delta.Minutes)*time.Minute)
	}

	// Return full ISO 8601 timestamp for precise filtering
	return absoluteTime.Format(time.RFC3339), nil
}
