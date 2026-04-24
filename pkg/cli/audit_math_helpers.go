package cli

import "fmt"

// safePercent returns percentage of part/total, returning 0 when total is 0.
func safePercent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

// formatPercent formats a float percentage with no decimal places
func formatPercent(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}
