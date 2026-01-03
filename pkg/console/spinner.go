// Package console provides terminal UI components including spinners for
// long-running operations.
//
// # Spinner Component
//
// The spinner provides visual feedback during long-running operations with a minimal
// dot animation (⣾ ⣽ ⣻ ⢿ ⡿ ⣟ ⣯ ⣷). It automatically adapts to the environment:
//   - TTY Detection: Spinners only animate in terminal environments (disabled in pipes/redirects)
//   - Accessibility: Respects ACCESSIBLE environment variable to disable animations
//   - Color Adaptation: Uses lipgloss adaptive colors for light/dark terminal themes
//
// # Usage Example
//
//	spinner := console.NewSpinner("Loading...")
//	spinner.Start()
//	// Long-running operation
//	spinner.Stop()
//
// # Accessibility
//
// Spinners respect the ACCESSIBLE environment variable. When ACCESSIBLE is set to any value,
// spinner animations are disabled to support screen readers and accessibility tools.
//
//	export ACCESSIBLE=1
//	gh aw compile workflow.md  # Spinners will be disabled
package console

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/githubnext/gh-aw/pkg/styles"
	"github.com/githubnext/gh-aw/pkg/tty"
)

// SpinnerWrapper wraps the spinner functionality with TTY detection
type SpinnerWrapper struct {
	model   spinner.Model
	message string
	enabled bool
	running bool
	stopCh  chan struct{}
	mu      sync.Mutex
}

// NewSpinner creates a new spinner with the given message using MiniDot style
// The spinner is automatically disabled when not running in a TTY or in accessibility mode
func NewSpinner(message string) *SpinnerWrapper {
	// Check if spinner should be enabled:
	// 1. Must be running in a TTY
	// 2. ACCESSIBLE environment variable must not be set
	enabled := tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == ""

	s := &SpinnerWrapper{
		message: message,
		enabled: enabled,
		stopCh:  make(chan struct{}),
	}

	if enabled {
		// Create a new spinner model with MiniDot style and info color
		s.model = spinner.New(
			spinner.WithSpinner(spinner.MiniDot),
			spinner.WithStyle(styles.Info),
		)
	}

	return s
}

// Start begins the spinner animation
func (s *SpinnerWrapper) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled || s.running {
		return
	}

	s.running = true
	s.stopCh = make(chan struct{})

	// Start the animation loop
	go s.animate()
}

// animate runs the spinner animation loop using Bubble Tea's Cmd pattern
func (s *SpinnerWrapper) animate() {
	msg := s.model.Tick()

	for {
		// Update the model and render
		s.mu.Lock()
		var cmd tea.Cmd
		s.model, cmd = s.model.Update(msg)
		fmt.Fprintf(os.Stderr, "\r%s %s", s.model.View(), s.message)
		s.mu.Unlock()

		// Execute the Cmd to get the next tick message (blocks for FPS duration)
		if cmd == nil {
			return
		}

		// Run cmd in goroutine to allow cancellation
		done := make(chan tea.Msg, 1)
		go func() {
			done <- cmd()
		}()

		select {
		case <-s.stopCh:
			return
		case msg = <-done:
		}
	}
}

// Stop stops the spinner animation
func (s *SpinnerWrapper) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled || !s.running {
		return
	}

	close(s.stopCh)
	s.running = false

	// Clear the line
	fmt.Fprint(os.Stderr, "\r\033[K")
}

// StopWithMessage stops the spinner and displays a final message
// The message will only be displayed if the spinner is enabled (TTY check)
func (s *SpinnerWrapper) StopWithMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled || !s.running {
		return
	}

	close(s.stopCh)
	s.running = false

	// Clear the line and print the final message
	fmt.Fprintf(os.Stderr, "\r\033[K%s\n", msg)
}

// UpdateMessage updates the spinner message
func (s *SpinnerWrapper) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.message = message
}

// IsEnabled returns whether the spinner is enabled (i.e., running in a TTY)
func (s *SpinnerWrapper) IsEnabled() bool {
	return s.enabled
}
