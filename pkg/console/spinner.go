package console

import (
	"time"

	"github.com/briandowns/spinner"
	"github.com/mattn/go-isatty"
)

// SpinnerWrapper wraps the spinner functionality with TTY detection
type SpinnerWrapper struct {
	spinner *spinner.Spinner
	enabled bool
}

// NewSpinner creates a new spinner with the given message
// The spinner is automatically disabled when not running in a TTY
func NewSpinner(message string) *SpinnerWrapper {
	enabled := isatty.IsTerminal(1) // Check if stdout is a terminal

	s := &SpinnerWrapper{
		enabled: enabled,
	}

	if enabled {
		s.spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.spinner.Suffix = " " + message
		_ = s.spinner.Color("cyan") // Ignore error as fallback is fine
	}

	return s
}

// Start begins the spinner animation
func (s *SpinnerWrapper) Start() {
	if s.enabled && s.spinner != nil {
		s.spinner.Start()
	}
}

// Stop stops the spinner animation
func (s *SpinnerWrapper) Stop() {
	if s.enabled && s.spinner != nil {
		s.spinner.Stop()
	}
}

// StopWithMessage stops the spinner and displays a final message
// The message will only be displayed if the spinner is enabled (TTY check)
func (s *SpinnerWrapper) StopWithMessage(msg string) {
	if s.enabled && s.spinner != nil {
		s.spinner.FinalMSG = msg + "\n"
		s.spinner.Stop()
	}
}

// UpdateMessage updates the spinner message
func (s *SpinnerWrapper) UpdateMessage(message string) {
	if s.enabled && s.spinner != nil {
		s.spinner.Suffix = " " + message
	}
}

// IsEnabled returns whether the spinner is enabled (i.e., running in a TTY)
func (s *SpinnerWrapper) IsEnabled() bool {
	return s.enabled
}
