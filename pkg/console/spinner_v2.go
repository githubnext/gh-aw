// Package console provides terminal UI components including spinners for
// long-running operations.
//
// # Spinner V2 Component
//
// This is an improved spinner implementation that simplifies the Bubble Tea integration:
//   - Simplified state management (single enabled flag, no running state)
//   - Reduced code size (~30% smaller than original)
//   - No mutex required (Bubble Tea handles concurrency via message passing)
//   - Cleaner program lifecycle (checked via program != nil)
//
// Usage:
//
//	spinner := console.NewSpinnerV2("Loading...")
//	spinner.Start()
//	// Long-running operation
//	spinner.Stop()
package console

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/githubnext/gh-aw/pkg/styles"
	"github.com/githubnext/gh-aw/pkg/tty"
)

// spinnerModelV2 is the Bubble Tea model for the spinner
type spinnerModelV2 struct {
	spinner spinner.Model
	message string
}

func (m spinnerModelV2) Init() tea.Cmd { return m.spinner.Tick }
func (m spinnerModelV2) View() string  { return fmt.Sprintf("\r%s %s", m.spinner.View(), m.message) }

func (m spinnerModelV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// SpinnerV2 wraps the spinner functionality with TTY detection and Bubble Tea program
type SpinnerV2 struct {
	program *tea.Program
	enabled bool
}

// NewSpinnerV2 creates a new spinner with the given message using MiniDot style.
// Automatically disabled when not running in a TTY or when ACCESSIBLE env var is set.
func NewSpinnerV2(message string) *SpinnerV2 {
	enabled := tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == ""
	s := &SpinnerV2{enabled: enabled}

	if enabled {
		model := spinnerModelV2{
			spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(styles.Info)),
			message: message,
		}
		s.program = tea.NewProgram(model, tea.WithOutput(os.Stderr), tea.WithoutRenderer())
	}
	return s
}

func (s *SpinnerV2) Start() {
	if s.enabled && s.program != nil {
		go func() { _, _ = s.program.Run() }()
	}
}

func (s *SpinnerV2) Stop() {
	if s.enabled && s.program != nil {
		s.program.Quit()
		fmt.Fprint(os.Stderr, "\r\033[K")
	}
}

func (s *SpinnerV2) StopWithMessage(msg string) {
	if s.enabled && s.program != nil {
		s.program.Quit()
		fmt.Fprintf(os.Stderr, "\r\033[K%s\n", msg)
	}
}

func (s *SpinnerV2) UpdateMessage(message string) {
	if s.enabled && s.program != nil {
		s.program.Send(updateMessageMsg(message))
	}
}

func (s *SpinnerV2) IsEnabled() bool { return s.enabled }
