package console

import "fmt"

func scaleBinaryBytes(size int64, units []string) (float64, string) {
	const unit = 1024
	if size < unit {
		return float64(size), "B"
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit && exp < len(units)-1; n /= unit {
		div *= unit
		exp++
	}

	return float64(size) / float64(div), units[exp]
}

// FormatFileSize formats file sizes in a human-readable way (e.g., "1.2 KB", "3.4 MB")
func FormatFileSize(size int64) string {
	if size == 0 {
		return "0 B"
	}

	value, unit := scaleBinaryBytes(size, []string{"KB", "MB", "GB", "TB"})
	if unit == "B" {
		return fmt.Sprintf("%d B", size)
	}

	return fmt.Sprintf("%.1f %s", value, unit)
}
