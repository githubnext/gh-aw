package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var utcOffsetPattern = regexp.MustCompile(`^([+-])(\d{2}):(\d{2})$`)

// NormalizeUTCOffset validates and normalizes a numeric UTC offset.
func NormalizeUTCOffset(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	matches := utcOffsetPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return "", fmt.Errorf("must be a numeric UTC offset like +00:00 or -08:00")
	}

	hours, _ := strconv.Atoi(matches[2])
	minutes, _ := strconv.Atoi(matches[3])
	if hours > 14 || minutes > 59 || (hours == 14 && minutes != 0) {
		return "", fmt.Errorf("must be a numeric UTC offset like +00:00 or -08:00")
	}

	return fmt.Sprintf("%s%02d:%02d", matches[1], hours, minutes), nil
}

// ParseUTCOffsetLocation converts a numeric UTC offset to a fixed time.Location.
func ParseUTCOffsetLocation(raw string) (*time.Location, error) {
	normalized, err := NormalizeUTCOffset(raw)
	if err != nil {
		return nil, err
	}

	hours, _ := strconv.Atoi(normalized[1:3])
	minutes, _ := strconv.Atoi(normalized[4:6])
	offsetSeconds := hours*60*60 + minutes*60
	if normalized[0] == '-' {
		offsetSeconds = -offsetSeconds
	}

	return time.FixedZone("UTC"+normalized, offsetSeconds), nil
}
