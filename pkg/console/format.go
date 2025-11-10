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
