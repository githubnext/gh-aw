package console

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/githubnext/gh-aw/pkg/styles"
)

// ProgressBar provides a reusable progress bar component with TTY detection
// and graceful fallback to text-based progress for non-TTY environments
type ProgressBar struct {
	progress progress.Model
	total    int64
	current  int64
}

// NewProgressBar creates a new progress bar with the specified total size
// The progress bar automatically adapts to TTY/non-TTY environments
func NewProgressBar(total int64) *ProgressBar {
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	// Use adaptive colors from theme system
	prog.FullColor = string(styles.ColorSuccess.Dark)
	prog.EmptyColor = string(styles.ColorComment.Dark)

	return &ProgressBar{
		progress: prog,
		total:    total,
		current:  0,
	}
}

// Update updates the current progress and returns a formatted string
// In TTY mode: Returns a visual progress bar with gradient
// In non-TTY mode: Returns text percentage with human-readable sizes
func (p *ProgressBar) Update(current int64) string {
	p.current = current

	// Handle edge case: avoid division by zero
	if p.total == 0 {
		if isTTY() {
			return p.progress.ViewAs(1.0)
		}
		return "100% (0B/0B)"
	}

	percent := float64(current) / float64(p.total)

	if !isTTY() {
		// Fallback for non-TTY: "50% (512MB/1024MB)"
		return fmt.Sprintf("%d%% (%s/%s)",
			int(percent*100),
			formatBytes(current),
			formatBytes(p.total))
	}

	return p.progress.ViewAs(percent)
}

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
