//go:build !js && !wasm

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
// message handling and rendering pipeline integration. It includes thread-safe
// lifecycle management:
//   - Thread-safe start/stop tracking with mutex protection
//   - Safe to call Stop/StopWithMessage before Start (no-op or message-only)
//   - Prevents multiple concurrent Start calls
//   - No deadlock when stopping before goroutine initializes
//   - Leverages Bubble Tea's message passing for updates
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

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/tty"
)

var spinnerLog = logger.New("console:spinner")

// updateMessageMsg is a custom message for updating the spinner message
type updateMessageMsg string

// spinnerModel is the Bubble Tea model for the spinner.
// Because we use tea.WithoutRenderer(), we must manually print in Update().
type spinnerModel struct {
	spinner spinner.Model
	message string
	output  *os.File
}

func (m spinnerModel) Init() tea.Cmd  { return m.spinner.Tick }
func (m spinnerModel) View() tea.View { return tea.View{} } // Not used with WithoutRenderer

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMessageMsg:
		m.message = string(msg)
		m.render()
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.render()
		return m, cmd
	}
	return m, nil
}

// render manually prints the spinner frame (required when using WithoutRenderer)
func (m spinnerModel) render() {
	if m.output != nil {
		fmt.Fprintf(m.output, "%s%s%s %s", ansiCarriageReturn, ansiClearLine, m.spinner.View(), m.message)
	}
}

// SpinnerWrapper wraps the spinner functionality with TTY detection and Bubble Tea program
type SpinnerWrapper struct {
	program *tea.Program
	enabled bool
	running bool
	mu      sync.Mutex
	wg      sync.WaitGroup
}

// NewSpinner creates a new spinner with the given message using MiniDot style.
// Automatically disabled when not running in a TTY or when ACCESSIBLE env var is set.
func NewSpinner(message string) *SpinnerWrapper {
	isTTY := tty.IsStderrTerminal()
	isAccessible := os.Getenv("ACCESSIBLE") != ""
	enabled := isTTY && !isAccessible
	spinnerLog.Printf("Creating spinner: message=%q, tty=%t, accessible=%t, enabled=%t", message, isTTY, isAccessible, enabled)
	s := &SpinnerWrapper{enabled: enabled}

	if enabled {
		model := spinnerModel{
			spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(styles.Info)),
			message: message,
			output:  os.Stderr,
		}
		// tea.WithInput(nil) disables stdin reading so the spinner does not consume key
		// events that should be handled by subsequent interactive forms (e.g. huh.Select).
		// Ctrl+C is still handled via OS signal delivery (SIGINT), which bubbletea
		// processes independently of the input reader.
		s.program = tea.NewProgram(model, tea.WithOutput(os.Stderr), tea.WithoutRenderer(), tea.WithInput(nil))
	}
	return s
}

func (s *SpinnerWrapper) Start() {
	if s.enabled && s.program != nil {
		s.mu.Lock()
		if s.running {
			s.mu.Unlock()
			spinnerLog.Print("Spinner already running, skipping Start")
			return
		}
		s.running = true
		s.wg.Add(1)
		s.mu.Unlock()
		spinnerLog.Print("Starting spinner")
		go func() {
			defer s.wg.Done()
			_, _ = s.program.Run()
		}()
	}
}

func (s *SpinnerWrapper) Stop() {
	if s.enabled && s.program != nil {
		s.mu.Lock()
		if s.running {
			s.running = false
			s.mu.Unlock()
			spinnerLog.Print("Stopping spinner")
			s.program.Quit()
			s.wg.Wait() // Wait for the goroutine to complete
			fmt.Fprintf(os.Stderr, "%s%s", ansiCarriageReturn, ansiClearLine)
		} else {
			s.mu.Unlock()
		}
	}
}

func (s *SpinnerWrapper) StopWithMessage(msg string) {
	if s.enabled && s.program != nil {
		s.mu.Lock()
		if s.running {
			s.running = false
			s.mu.Unlock()
			s.program.Quit()
			s.wg.Wait() // Wait for the goroutine to complete
			fmt.Fprintf(os.Stderr, "%s%s%s\n", ansiCarriageReturn, ansiClearLine, msg)
		} else {
			s.mu.Unlock()
			// Still print the message even if spinner wasn't running
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		}
	} else if msg != "" {
		// If spinner is disabled, still print the message for user feedback
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
}

func (s *SpinnerWrapper) UpdateMessage(message string) {
	if s.enabled && s.program != nil {
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()
		if running {
			s.program.Send(updateMessageMsg(message))
		}
	}
}
