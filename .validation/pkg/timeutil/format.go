package timeutil

import (
	"fmt"
	"math"
	"time"
)

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
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatDurationMs formats a duration given in milliseconds as a human-readable string.
// Examples: 500 -> "500ms", 1500 -> "1.5s", 90000 -> "1m30s"
func FormatDurationMs(ms int) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := float64(ms) / 1000.0
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	minutes := int(seconds) / 60
	secs := math.Mod(seconds, 60)
	return fmt.Sprintf("%dm%.0fs", minutes, secs)
}

// FormatDurationNs formats a duration given in nanoseconds as a human-readable string.
// Returns "—" for zero or negative values. Uses Go's standard duration rounding to seconds.
func FormatDurationNs(ns int64) string {
	if ns <= 0 {
		return "—"
	}
	d := time.Duration(ns)
	return d.Round(time.Second).String()
}
