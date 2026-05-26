package cli

import (
	"math"
	"time"
)

// formatUnixTimestamp converts a Unix timestamp (float64) to a human-readable time string (HH:MM:SS).
func formatUnixTimestamp(ts float64) string {
	if ts <= 0 {
		return "-"
	}
	sec := int64(math.Floor(ts))
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()
	return t.Format("15:04:05")
}
