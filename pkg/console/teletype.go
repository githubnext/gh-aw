// Package console provides terminal UI components including teletype-style
// line rendering for smooth interactive experiences.
//
// # Teletype Component
//
// The teletype component displays lines of text with a smooth typing animation,
// simulating the appearance of a classic teletype machine. This prevents jarring
// screen updates during interactive workflows.
//
// # Implementation
//
// Uses Bubble Tea for smooth animations with configurable timing:
//   - Character-by-character rendering with configurable delay
//   - Automatic TTY detection (falls back to instant display in pipes/redirects)
//   - Accessibility support via ACCESSIBLE environment variable
//   - Thread-safe lifecycle management
//
// # Usage Example
//
//	teletype := console.NewTeletype()
//	teletype.PrintLine("Welcome to GitHub Agentic Workflows!")
//	teletype.PrintLine("This tool will walk you through adding a workflow.")
//	teletype.Wait() // Wait for all lines to finish rendering
//
// # Accessibility
//
// When ACCESSIBLE is set to any value, teletype animations are disabled and
// lines are displayed instantly to support screen readers and accessibility tools.
//
//	export ACCESSIBLE=1
//	gh aw add workflow  # Lines will display instantly
package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/githubnext/gh-aw/pkg/tty"
)

// Default teletype settings
const (
	// DefaultCharsPerSecond is the default typing speed (characters per second)
	DefaultCharsPerSecond = 40
	// MinCharsPerSecond is the minimum typing speed
	MinCharsPerSecond = 10
	// MaxCharsPerSecond is the maximum typing speed
	MaxCharsPerSecond = 200
)

// teletypeTickMsg is sent periodically to advance the teletype animation
type teletypeTickMsg time.Time

// teletypeLineMsg is sent to add a new line to the teletype
type teletypeLineMsg string

// teletypeModel is the Bubble Tea model for the teletype
type teletypeModel struct {
	lines            []string  // All lines to display
	currentLine      int       // Index of current line being typed
	currentCharIndex int       // Index of current character in current line
	charsPerSecond   int       // Typing speed
	writer           io.Writer // Output writer
	done             bool      // True when all lines are displayed
}

// Init initializes the teletype model
func (m teletypeModel) Init() tea.Cmd {
	if len(m.lines) == 0 {
		return tea.Quit
	}
	return m.tick()
}

// Update handles messages and updates the model
func (m teletypeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			// On Ctrl+C, display all remaining text instantly
			for m.currentLine < len(m.lines) {
				remaining := m.lines[m.currentLine][m.currentCharIndex:]
				fmt.Fprint(m.writer, remaining)
				if m.currentLine < len(m.lines)-1 {
					fmt.Fprintln(m.writer)
				}
				m.currentLine++
				m.currentCharIndex = 0
			}
			m.done = true
			return m, tea.Quit
		}

	case teletypeTickMsg:
		if m.currentLine >= len(m.lines) {
			m.done = true
			return m, tea.Quit
		}

		line := m.lines[m.currentLine]

		// Display next character
		if m.currentCharIndex < len(line) {
			fmt.Fprint(m.writer, string(line[m.currentCharIndex]))
			m.currentCharIndex++
			return m, m.tick()
		}

		// Current line is complete, move to next line
		if m.currentLine < len(m.lines)-1 {
			fmt.Fprintln(m.writer)
		}
		m.currentLine++
		m.currentCharIndex = 0

		// If there are more lines, continue ticking
		if m.currentLine < len(m.lines) {
			return m, m.tick()
		}

		// All lines complete
		m.done = true
		return m, tea.Quit

	case teletypeLineMsg:
		// Add a new line to the queue
		m.lines = append(m.lines, string(msg))
		return m, nil
	}

	return m, nil
}

// View renders the teletype (not used since we write directly)
func (m teletypeModel) View() string {
	return ""
}

// tick returns a command that sends a tick message after the appropriate delay
func (m teletypeModel) tick() tea.Cmd {
	delay := time.Second / time.Duration(m.charsPerSecond)
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return teletypeTickMsg(t)
	})
}

// Teletype wraps the teletype functionality with TTY detection and Bubble Tea program
type Teletype struct {
	program        *tea.Program
	model          *teletypeModel
	enabled        bool
	running        bool
	mu             sync.Mutex
	charsPerSecond int
	writer         io.Writer
}

// TeletypeOption is a functional option for configuring Teletype
type TeletypeOption func(*Teletype)

// WithCharsPerSecond sets the typing speed in characters per second
func WithCharsPerSecond(cps int) TeletypeOption {
	return func(t *Teletype) {
		if cps < MinCharsPerSecond {
			cps = MinCharsPerSecond
		}
		if cps > MaxCharsPerSecond {
			cps = MaxCharsPerSecond
		}
		t.charsPerSecond = cps
	}
}

// WithWriter sets the output writer (defaults to os.Stderr)
func WithWriter(w io.Writer) TeletypeOption {
	return func(t *Teletype) {
		t.writer = w
	}
}

// NewTeletype creates a new teletype with the given options.
// Automatically disabled when not running in a TTY or when ACCESSIBLE env var is set.
func NewTeletype(opts ...TeletypeOption) *Teletype {
	enabled := tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == ""

	t := &Teletype{
		enabled:        enabled,
		charsPerSecond: DefaultCharsPerSecond,
		writer:         os.Stderr,
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Start initializes the teletype for printing
func (t *Teletype) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return
	}

	t.running = true

	if t.enabled {
		t.model = &teletypeModel{
			lines:          []string{},
			charsPerSecond: t.charsPerSecond,
			writer:         t.writer,
		}
		t.program = tea.NewProgram(t.model, tea.WithOutput(t.writer), tea.WithoutRenderer())
		go func() { _, _ = t.program.Run() }()
	}
}

// PrintLine adds a line to be printed with teletype effect.
// If teletype is disabled, prints the line immediately.
// Lines are printed in the order they are added.
func (t *Teletype) PrintLine(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.enabled {
		// In non-TTY or accessible mode, print immediately
		fmt.Fprintln(t.writer, line)
		return
	}

	if !t.running {
		// Auto-start if not started
		t.running = true
		t.model = &teletypeModel{
			lines:          []string{line},
			charsPerSecond: t.charsPerSecond,
			writer:         t.writer,
		}
		t.program = tea.NewProgram(t.model, tea.WithOutput(t.writer), tea.WithoutRenderer())
		go func() { _, _ = t.program.Run() }()
		return
	}

	// Add line to existing model
	if t.model != nil {
		t.model.lines = append(t.model.lines, line)
	}
}

// Print adds text to the current line (without newline).
// If this is the first call or follows a PrintLine, starts a new line.
func (t *Teletype) Print(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.enabled {
		// In non-TTY or accessible mode, print immediately
		fmt.Fprint(t.writer, text)
		return
	}

	if !t.running {
		// Auto-start if not started
		t.running = true
		t.model = &teletypeModel{
			lines:          []string{text},
			charsPerSecond: t.charsPerSecond,
			writer:         t.writer,
		}
		t.program = tea.NewProgram(t.model, tea.WithOutput(t.writer), tea.WithoutRenderer())
		go func() { _, _ = t.program.Run() }()
		return
	}

	// Append to last line if it exists, otherwise create new line
	if t.model != nil && len(t.model.lines) > 0 {
		// If we're currently typing the last line, we need to append to it
		if t.model.currentLine == len(t.model.lines)-1 {
			t.model.lines[len(t.model.lines)-1] += text
		} else {
			// Last line is already complete, add a new one
			t.model.lines = append(t.model.lines, text)
		}
	}
}

// Wait blocks until all queued lines have been printed.
// Has no effect if teletype is disabled.
func (t *Teletype) Wait() {
	t.mu.Lock()
	running := t.running
	enabled := t.enabled
	program := t.program
	t.mu.Unlock()

	if !enabled || !running || program == nil {
		return
	}

	// Wait for program to finish
	program.Wait()

	t.mu.Lock()
	t.running = false
	t.mu.Unlock()
}

// Stop immediately stops the teletype and displays any remaining text instantly.
func (t *Teletype) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running || t.program == nil {
		return
	}

	// Display all remaining text instantly
	if t.model != nil {
		for i := t.model.currentLine; i < len(t.model.lines); i++ {
			if i == t.model.currentLine {
				// Print remaining characters in current line
				remaining := t.model.lines[i][t.model.currentCharIndex:]
				fmt.Fprint(t.writer, remaining)
			} else {
				// Print complete line
				fmt.Fprintln(t.writer)
				fmt.Fprint(t.writer, t.model.lines[i])
			}
		}
	}

	t.program.Quit()
	t.running = false
}

// IsEnabled returns whether teletype animations are enabled
func (t *Teletype) IsEnabled() bool {
	return t.enabled
}

// PrintLines is a convenience function that prints multiple lines and waits for completion.
// Each line is printed with a teletype effect in sequence.
func PrintLines(lines []string, opts ...TeletypeOption) {
	if len(lines) == 0 {
		return
	}

	teletype := NewTeletype(opts...)
	teletype.Start()

	for _, line := range lines {
		teletype.PrintLine(line)
	}

	teletype.Wait()
}

// PrintLinesSeparated prints lines with a blank line between each (useful for sections)
func PrintLinesSeparated(lines []string, opts ...TeletypeOption) {
	if len(lines) == 0 {
		return
	}

	teletype := NewTeletype(opts...)
	teletype.Start()

	for i, line := range lines {
		teletype.PrintLine(line)
		if i < len(lines)-1 {
			teletype.PrintLine("")
		}
	}

	teletype.Wait()
}

// PrintLinesWithPrefix prints lines with a common prefix (useful for lists)
func PrintLinesWithPrefix(lines []string, prefix string, opts ...TeletypeOption) {
	if len(lines) == 0 {
		return
	}

	prefixedLines := make([]string, len(lines))
	for i, line := range lines {
		prefixedLines[i] = prefix + line
	}

	PrintLines(prefixedLines, opts...)
}

// FormatAndPrintLines formats lines by replacing placeholders and prints with teletype effect
func FormatAndPrintLines(lines []string, replacements map[string]string, opts ...TeletypeOption) {
	formattedLines := make([]string, len(lines))
	for i, line := range lines {
		formatted := line
		for old, new := range replacements {
			formatted = strings.ReplaceAll(formatted, old, new)
		}
		formattedLines[i] = formatted
	}

	PrintLines(formattedLines, opts...)
}
