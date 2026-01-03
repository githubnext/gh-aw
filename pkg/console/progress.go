package console

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
)

// ProgressBar provides a reusable progress bar component with TTY detection
// and graceful fallback to text-based progress for non-TTY environments.
//
// Visual Features:
//   - Scaled gradient effect from purple (#BD93F9) to cyan (#8BE9FD)
//   - Smooth color transitions using bubbles v0.21.0+ gradient capabilities
//   - Gradient scales with filled portion for enhanced visual feedback
//   - Works well in both light and dark terminal themes
//
// The gradient provides visual appeal without affecting functionality:
//   - TTY mode: Visual progress bar with smooth gradient transitions
//   - Non-TTY mode: Text-based percentage with human-readable byte sizes
type ProgressBar struct {
	progress progress.Model
	total    int64
	current  int64
}

// NewProgressBar creates a new progress bar with the specified total size
// The progress bar automatically adapts to TTY/non-TTY environments
func NewProgressBar(total int64) *ProgressBar {
	// Use scaled gradient for improved visual effect
	// The gradient blends from purple to cyan, creating a smooth
	// color transition as progress advances. WithScaledGradient
	// ensures the gradient scales with the filled portion for better
	// visual feedback.
	//
	// Color choices:
	// - Start (0%): #BD93F9 (purple) - vibrant, attention-grabbing
	// - End (100%): #8BE9FD (cyan) - cool, completion feeling
	// These colors work well in both light and dark terminal themes
	prog := progress.New(
		progress.WithScaledGradient("#BD93F9", "#8BE9FD"),
		progress.WithWidth(40),
	)

	// Use muted color for empty portion to maintain focus on progress
	prog.EmptyColor = "#6272A4" // Muted purple-gray

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
