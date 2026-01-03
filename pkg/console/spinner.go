package console

import (
	"fmt"
	"os"
	"sync"
	"time"

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

// NewSpinner creates a new spinner with the given message
// The spinner is automatically disabled when not running in a TTY
func NewSpinner(message string) *SpinnerWrapper {
	enabled := tty.IsStderrTerminal() // Check if stderr is a terminal (spinner writes to stderr)

	s := &SpinnerWrapper{
		message: message,
		enabled: enabled,
		stopCh:  make(chan struct{}),
	}

	if enabled {
		// Create a new spinner model with Dot style and info color
		s.model = spinner.New(
			spinner.WithSpinner(spinner.Dot),
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

	// Start the animation loop in a goroutine
	go s.animate()
}

// animate runs the spinner animation loop using Bubble Tea's built-in tick mechanism
func (s *SpinnerWrapper) animate() {
	// Use tea.Tick to schedule the next animation frame
	s.processCmd(tea.Tick(s.model.Spinner.FPS, func(t time.Time) tea.Msg {
		return s.model.Tick()
	}))
}

// processCmd executes a Bubble Tea command and handles the resulting message
func (s *SpinnerWrapper) processCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}

	go func() {
		msg := cmd()
		if msg == nil {
			return
		}

		// Check if we should stop
		select {
		case <-s.stopCh:
			return
		default:
		}

		s.mu.Lock()
		// Update the spinner model with the tick message
		var nextCmd tea.Cmd
		s.model, nextCmd = s.model.Update(msg)

		// Render the spinner with the message
		output := fmt.Sprintf("\r%s %s", s.model.View(), s.message)
		fmt.Fprint(os.Stderr, output)
		s.mu.Unlock()

		// Continue the animation loop
		s.processCmd(nextCmd)
	}()
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
