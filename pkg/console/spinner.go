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
// # Implementation
//
// This spinner uses idiomatic Bubble Tea patterns with tea.NewProgram() for proper
// message handling and rendering pipeline integration. This approach:
//   - Eliminates manual goroutine management
//   - Removes mutex requirements through Bubble Tea's message passing
//   - Leverages Bubble Tea's framerate optimization
//   - Provides standard architecture consistent with other console components
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

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/githubnext/gh-aw/pkg/styles"
	"github.com/githubnext/gh-aw/pkg/tty"
)

// updateMessageMsg is a custom message for updating the spinner message
type updateMessageMsg string

// spinnerModel is the Bubble Tea model for the spinner
type spinnerModel struct {
	spinner spinner.Model
	message string
}

// Init initializes the spinner model and starts the ticker
func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages and updates the spinner model
func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMessageMsg:
		m.message = string(msg)
		return m, nil
	case tea.KeyMsg:
		// Allow Ctrl+C to quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the spinner with its message
func (m spinnerModel) View() string {
	return fmt.Sprintf("\r%s %s", m.spinner.View(), m.message)
}

// SpinnerWrapper wraps the spinner functionality with TTY detection and Bubble Tea program
type SpinnerWrapper struct {
	program *tea.Program
	enabled bool
	running bool
}

// NewSpinner creates a new spinner with the given message using MiniDot style
// The spinner is automatically disabled when not running in a TTY or in accessibility mode
func NewSpinner(message string) *SpinnerWrapper {
	// Check if spinner should be enabled:
	// 1. Must be running in a TTY
	// 2. ACCESSIBLE environment variable must not be set
	enabled := tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == ""

	s := &SpinnerWrapper{
		enabled: enabled,
		running: false,
	}

	if enabled {
		// Create a new spinner model with MiniDot style and info color
		spinnerModel := spinnerModel{
			spinner: spinner.New(
				spinner.WithSpinner(spinner.MiniDot),
				spinner.WithStyle(styles.Info),
			),
			message: message,
		}

		// Create Bubble Tea program with output to stderr
		s.program = tea.NewProgram(
			spinnerModel,
			tea.WithOutput(os.Stderr),
			tea.WithoutRenderer(), // Use inline mode without alt screen
		)
	}

	return s
}

// Start begins the spinner animation
func (s *SpinnerWrapper) Start() {
	if !s.enabled || s.running {
		return
	}

	s.running = true

	// Start the program in the background
	go func() {
		_, _ = s.program.Run()
	}()
}

// Stop stops the spinner animation and clears the line
func (s *SpinnerWrapper) Stop() {
	if !s.enabled || !s.running {
		return
	}

	s.running = false

	// Send quit message to stop the program
	s.program.Quit()

	// Clear the line
	fmt.Fprint(os.Stderr, "\r\033[K")
}

// StopWithMessage stops the spinner and displays a final message
// The message will only be displayed if the spinner is enabled (TTY check)
func (s *SpinnerWrapper) StopWithMessage(msg string) {
	if !s.enabled || !s.running {
		return
	}

	s.running = false

	// Send quit message to stop the program
	s.program.Quit()

	// Clear the line and print the final message
	fmt.Fprintf(os.Stderr, "\r\033[K%s\n", msg)
}

// UpdateMessage updates the spinner message
func (s *SpinnerWrapper) UpdateMessage(message string) {
	if !s.enabled || !s.running {
		return
	}

	// Send update message through Bubble Tea's message passing
	s.program.Send(updateMessageMsg(message))
}

// IsEnabled returns whether the spinner is enabled (i.e., running in a TTY)
func (s *SpinnerWrapper) IsEnabled() bool {
	return s.enabled
}
