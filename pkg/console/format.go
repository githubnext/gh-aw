package console

import "fmt"

// FormatFileSize formats file sizes in a human-readable way (e.g., "1.2 KB", "3.4 MB")
func FormatFileSize(size int64) string {
	if size == 0 {
		return "0 B"
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(units) {
		exp = len(units) - 1
		div = int64(1) << (10 * (exp + 1))
	}

	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}

// FormatNumberOrEmpty formats a number or returns empty string if zero
func FormatNumberOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return FormatNumber(n)
}

// FormatCostOrEmpty formats cost or returns empty string if zero
func FormatCostOrEmpty(cost float64) string {
	if cost == 0 {
		return ""
	}
	return fmt.Sprintf("%.3f", cost)
}

// FormatIntOrEmpty formats an int or returns empty string if zero
func FormatIntOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}

// TruncateString truncates a string to maxLen with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen > 3 {
		return s[:maxLen-3] + "..."
	}
	return s[:maxLen]
}
