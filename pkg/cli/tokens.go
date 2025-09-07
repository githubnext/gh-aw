package cli

import "fmt"

// formatTokenCount formats byte count as estimated tokens (approximation: ~4 bytes per token)
func formatTokenCount(size int64) string {
	if size == 0 {
		return "0 tokens"
	}

	// Approximate conversion: ~4 bytes per token for English text
	tokens := size / 4

	if tokens < 1000 {
		return fmt.Sprintf("%d tokens", tokens)
	} else if tokens < 1000000 {
		// Format as thousands (k)
		k := float64(tokens) / 1000
		if k >= 100 {
			return fmt.Sprintf("%.0fk tokens", k)
		} else if k >= 10 {
			return fmt.Sprintf("%.1fk tokens", k)
		} else {
			return fmt.Sprintf("%.2fk tokens", k)
		}
	} else {
		// Format as millions (M)
		m := float64(tokens) / 1000000
		if m >= 100 {
			return fmt.Sprintf("%.0fM tokens", m)
		} else if m >= 10 {
			return fmt.Sprintf("%.1fM tokens", m)
		} else {
			return fmt.Sprintf("%.2fM tokens", m)
		}
	}
}
