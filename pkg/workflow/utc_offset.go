package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var utcOffsetLog = logger.New("workflow:utc_offset")

var utcOffsetPattern = regexp.MustCompile(`^([+-])(\d{2}):(\d{2})$`)

func utcOffsetFormatError(value string) error {
	return fmt.Errorf("must be a numeric UTC offset like +00:00 or -08:00, got %q. Expected format ±HH:MM in the range -14:00 to +14:00. Example: utc: \"-08:00\"", value)
}

// NormalizeUTCOffset validates and normalizes a numeric UTC offset.
func NormalizeUTCOffset(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	matches := utcOffsetPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		utcOffsetLog.Printf("UTC offset %q does not match expected +HH:MM/-HH:MM format", trimmed)
		return "", utcOffsetFormatError(trimmed)
	}

	hours, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", utcOffsetFormatError(trimmed)
	}
	minutes, err := strconv.Atoi(matches[3])
	if err != nil {
		return "", utcOffsetFormatError(trimmed)
	}
	if hours > 14 || minutes > 59 || (hours == 14 && minutes != 0) {
		utcOffsetLog.Printf("UTC offset %q out of range (hours=%d, minutes=%d)", trimmed, hours, minutes)
		return "", utcOffsetFormatError(trimmed)
	}

	normalized := fmt.Sprintf("%s%02d:%02d", matches[1], hours, minutes)
	utcOffsetLog.Printf("Normalized UTC offset %q to %q", trimmed, normalized)
	return normalized, nil
}

// ParseUTCOffsetLocation converts a numeric UTC offset to a fixed time.Location.
func ParseUTCOffsetLocation(raw string) (*time.Location, error) {
	normalized, err := NormalizeUTCOffset(raw)
	if err != nil {
		return nil, err
	}

	hours, err := strconv.Atoi(normalized[1:3])
	if err != nil {
		return nil, fmt.Errorf("normalized UTC offset %q could not be parsed. Expected normalized format ±HH:MM. Example: +00:00. Underlying error: %w", normalized, err)
	}
	minutes, err := strconv.Atoi(normalized[4:6])
	if err != nil {
		return nil, fmt.Errorf("normalized UTC offset %q could not be parsed. Expected normalized format ±HH:MM. Example: -08:00. Underlying error: %w", normalized, err)
	}
	offsetSeconds := hours*60*60 + minutes*60
	if normalized[0] == '-' {
		offsetSeconds = -offsetSeconds
	}

	return time.FixedZone("UTC"+normalized, offsetSeconds), nil
}
