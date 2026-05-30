package cli

import (
	"fmt"
	"strconv"
)

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

// formatVolumeChange formats the volume change as a human-readable string
func formatVolumeChange(total1, total2 int) string {
	if total1 == 0 {
		return "+∞"
	}
	pctChange := (float64(total2-total1) / float64(total1)) * 100
	if pctChange >= 0 {
		return "+" + formatPercent(pctChange)
	}
	return formatPercent(pctChange)
}

// formatPercentagePointChange formats the change between two ratio values (0.0-1.0) as a
// percentage-point delta (e.g. "+1.5pp", "-2.3pp")
func formatPercentagePointChange(ratio1, ratio2 float64) string {
	delta := (ratio2 - ratio1) * 100
	if delta >= 0 {
		return fmt.Sprintf("+%.1fpp", delta)
	}
	return fmt.Sprintf("%.1fpp", delta)
}

// formatCountChange formats the absolute change in a count value (e.g. "+3", "-1")
func formatCountChange(count1, count2 int) string {
	delta := count2 - count1
	if delta >= 0 {
		return fmt.Sprintf("+%d", delta)
	}
	return strconv.Itoa(delta)
}
