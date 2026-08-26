package timeutil

import (
	"fmt"
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
// Examples: 500 -> "500ms", 1500 -> "1.5s", 90000 -> "1.5m"
func FormatDurationMs(ms int) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
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
