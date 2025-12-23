package logger

import (
	"regexp"
	"strings"
)

// Pre-compiled regexes for performance (avoid recompiling in hot paths).
var (
	// Timestamp patterns for log cleanup
	// Pattern 1: ISO 8601 with T or space separator (e.g., "2024-01-01T12:00:00.123Z " or "2024-01-01 12:00:00 ").
	timestampPattern1 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(\.\d+)?([+-]\d{2}:\d{2}|Z)?\s*`)
	// Pattern 2: Bracketed date-time (e.g., "[2024-01-01 12:00:00] ").
	timestampPattern2 = regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\]\s*`)
	// Pattern 3: Bracketed time only (e.g., "[12:00:00] ").
	timestampPattern3 = regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\]\s+`)
	// Pattern 4: Time only with optional milliseconds (e.g., "12:00:00.123 ").
	timestampPattern4 = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(\.\d+)?\s+`)

	// Log level pattern for message cleanup (case-insensitive).
	logLevelPattern = regexp.MustCompile(`(?i)^\[?(ERROR|WARNING|WARN|INFO|DEBUG)\]?\s*[:-]?\s*`)
)

// ExtractErrorMessage extracts a clean error message from a log line.
// It removes timestamps, log level prefixes, and other common noise.
// If the message is longer than 200 characters, it will be truncated.
func ExtractErrorMessage(line string) string {
	// Remove common timestamp patterns using pre-compiled regexes
	cleanedLine := line
	cleanedLine = timestampPattern1.ReplaceAllString(cleanedLine, "")
	cleanedLine = timestampPattern2.ReplaceAllString(cleanedLine, "")
	cleanedLine = timestampPattern3.ReplaceAllString(cleanedLine, "")
	cleanedLine = timestampPattern4.ReplaceAllString(cleanedLine, "")

	// Remove common log level prefixes using pre-compiled regex
	cleanedLine = logLevelPattern.ReplaceAllString(cleanedLine, "")

	// Trim whitespace
	cleanedLine = strings.TrimSpace(cleanedLine)

	// If the line is too long (>200 chars), truncate it
	if len(cleanedLine) > 200 {
		cleanedLine = cleanedLine[:197] + "..."
	}

	return cleanedLine
}
