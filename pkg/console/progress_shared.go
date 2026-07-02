package console

import "fmt"

// formatBytes converts bytes to human-readable format (KB, MB, GB)
func formatBytes(bytes int64) string {
	value, unit := scaleBinaryBytes(bytes, []string{"KB", "MB", "GB"})
	if unit == "B" {
		return fmt.Sprintf("%dB", bytes)
	}
	if unit == "GB" {
		return fmt.Sprintf("%.2fGB", value)
	}
	return fmt.Sprintf("%.1f%s", value, unit)
}
