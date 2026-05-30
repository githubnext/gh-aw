package console

import "fmt"

// formatBytes converts bytes to human-readable format (KB, MB, GB)
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	if bytes < KB {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	} else if bytes < GB {
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	}
	return fmt.Sprintf("%.2fGB", float64(bytes)/GB)
}
