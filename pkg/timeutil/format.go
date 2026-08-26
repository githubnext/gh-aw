package timeutil

import (
	"fmt"
	"math"
	"time"
)

// maxMsForDuration is the largest millisecond value that can be safely
// converted to a time.Duration (nanoseconds) without overflowing int64.
const maxMsForDuration = math.MaxInt64 / int64(time.Millisecond)

// msPerHour is the number of milliseconds in one hour, used to format
// millisecond values that would overflow time.Duration's nanosecond range.
const msPerHour = float64(time.Hour / time.Millisecond)

// round1 rounds v to one decimal place using standard rounding.
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// FormatDuration formats a duration for display like the debug npm package.
// It provides granular formatting from nanoseconds to hours.
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		if v := round1(d.Seconds()); v < 60 {
			return fmt.Sprintf("%.1fs", v)
		}
		// Rounding pushed the value into the next unit (e.g. 59999ms -> 1.0m).
	}
	if d < time.Hour {
		if v := round1(d.Minutes()); v < 60 {
			return fmt.Sprintf("%.1fm", v)
		}
		// Rounding pushed the value into the next unit (e.g. 3599999ms -> 1.0h).
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatDurationMs formats a duration given in milliseconds as a human-readable string.
// Examples: 500 -> "500ms", 1500 -> "1.5s", 90000 -> "1.5m"
// Values that would overflow time.Duration's nanosecond range are formatted
// directly in hours to avoid silent wraparound.
func FormatDurationMs(ms int) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	if int64(ms) > maxMsForDuration {
		return fmt.Sprintf("%.1fh", float64(ms)/msPerHour)
	}
	return FormatDuration(time.Duration(ms) * time.Millisecond)
}

// FormatDurationNs formats a duration given in nanoseconds as a human-readable string.
// Returns "—" for zero or negative values.
func FormatDurationNs(ns int64) string {
	if ns <= 0 {
		return "—"
	}
	return FormatDuration(time.Duration(ns))
}
