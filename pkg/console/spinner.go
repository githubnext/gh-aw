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
//   - Simplified state management (single enabled flag, no running state)
//   - No mutex required (Bubble Tea handles concurrency via message passing)
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

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }
func (m spinnerModel) View() string  { return fmt.Sprintf("\r%s %s", m.spinner.View(), m.message) }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMessageMsg:
		m.message = string(msg)
		return m, nil
	case tea.KeyMsg:
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

// SpinnerWrapper wraps the spinner functionality with TTY detection and Bubble Tea program
type SpinnerWrapper struct {
	program *tea.Program
	enabled bool
}

// NewSpinner creates a new spinner with the given message using MiniDot style.
// Automatically disabled when not running in a TTY or when ACCESSIBLE env var is set.
func NewSpinner(message string) *SpinnerWrapper {
	enabled := tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == ""
	s := &SpinnerWrapper{enabled: enabled}

	if enabled {
		model := spinnerModel{
			spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(styles.Info)),
			message: message,
		}
		s.program = tea.NewProgram(model, tea.WithOutput(os.Stderr), tea.WithoutRenderer())
	}
	return s
}

func (s *SpinnerWrapper) Start() {
	if s.enabled && s.program != nil {
		go func() { _, _ = s.program.Run() }()
	}
}

func (s *SpinnerWrapper) Stop() {
	if s.enabled && s.program != nil {
		s.program.Quit()
		fmt.Fprint(os.Stderr, "\r\033[K")
	}
}

func (s *SpinnerWrapper) StopWithMessage(msg string) {
	if s.enabled && s.program != nil {
		s.program.Quit()
		fmt.Fprintf(os.Stderr, "\r\033[K%s\n", msg)
	}
}

func (s *SpinnerWrapper) UpdateMessage(message string) {
	if s.enabled && s.program != nil {
		s.program.Send(updateMessageMsg(message))
	}
}

func (s *SpinnerWrapper) IsEnabled() bool { return s.enabled }
