// Package console provides terminal UI components including a teletype effect
// for animated text output in interactive CLI commands.
//
// # Teletype Component
//
// The teletype effect provides a pleasant typing animation for text output,
// making it more engaging and easier to read by displaying text character by
// character. It automatically adapts to the environment:
//   - TTY Detection: Animation only works in terminal environments (instant display in pipes/redirects)
//   - Accessibility: Respects ACCESSIBLE environment variable to disable animations
//   - Color Support: Works with existing console formatting functions
//
// # Implementation
//
// This component uses Bubble Tea's message passing for smooth character-by-character
// animation with configurable speed. It includes:
//   - Customizable typing speed (characters per second)
//   - Support for multi-line text
//   - Thread-safe lifecycle management
//   - Graceful fallback for non-TTY environments
//
// # Usage Example
//
//	console.TeletypeWrite(os.Stderr, "Welcome to GitHub Agentic Workflows!\n")
//	console.TeletypeWrite(os.Stderr, console.FormatSuccessMessage("Setup complete!"))
//
// # Integration with Spinners
//
// The teletype effect works well with spinners. Spinners use carriage return
// and line clearing (\r\033[K) to update in place, while teletype writes
// characters sequentially. Best practices:
//
//   - Use SpinnerWrapper.StopWithMessage() to display a message after the spinner
//   - After spinner.Stop(), teletype can write on the cleared line
//   - Avoid starting teletype while a spinner is running
//
// Example pattern:
//
//	spinner := console.NewSpinner("Loading...")
//	spinner.Start()
//	// Long operation
//	spinner.StopWithMessage(console.FormatSuccessMessage("Loaded!"))
//	console.TeletypeWriteln(os.Stderr, "Processing data...")
//
// # Accessibility
//
// The teletype effect respects the ACCESSIBLE environment variable. When set,
// text is displayed instantly without animation to support screen readers.
//
//	export ACCESSIBLE=1
//	gh aw add workflow  # Text displays instantly
package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/githubnext/gh-aw/pkg/tty"
)

// teletypeTickMsg is sent on each character display interval
type teletypeTickMsg time.Time

// teletypeModel is the Bubble Tea model for the teletype effect.
// Because we use tea.WithoutRenderer(), we must manually print in Update().
type teletypeModel struct {
	text         string
	currentIndex int
	output       io.Writer
	charsPerTick int // Number of characters to display per tick
	tickDuration time.Duration
	done         bool
}

func (m teletypeModel) Init() tea.Cmd {
	return m.tick()
}

func (m teletypeModel) View() string {
	return "" // Not used with WithoutRenderer
}

func (m teletypeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case teletypeTickMsg:
		if m.currentIndex >= len(m.text) {
			m.done = true
			return m, tea.Quit
		}

		// Display next batch of characters
		endIndex := m.currentIndex + m.charsPerTick
		if endIndex > len(m.text) {
			endIndex = len(m.text)
		}

		// Print the characters
		chunk := m.text[m.currentIndex:endIndex]
		fmt.Fprint(m.output, chunk)

		m.currentIndex = endIndex

		// Continue ticking if not done
		if m.currentIndex < len(m.text) {
			return m, m.tick()
		}

		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

// tick returns a command that waits for the configured duration and sends a tick message
func (m teletypeModel) tick() tea.Cmd {
	return tea.Tick(m.tickDuration, func(t time.Time) tea.Msg {
		return teletypeTickMsg(t)
	})
}

// TeletypeConfig holds configuration for the teletype effect
type TeletypeConfig struct {
	// CharsPerSecond controls typing speed (default: 120)
	CharsPerSecond int
	// Enabled controls whether animation is used (default: auto-detect)
	Enabled *bool
}

// DefaultTeletypeConfig returns the default teletype configuration
func DefaultTeletypeConfig() TeletypeConfig {
	return TeletypeConfig{
		CharsPerSecond: 120,
		Enabled:        nil, // Auto-detect
	}
}

// TeletypeWrite writes text to the output with a teletype effect.
// In non-TTY environments or when ACCESSIBLE is set, text is displayed instantly.
func TeletypeWrite(w io.Writer, text string) error {
	return TeletypeWriteConfig(w, text, DefaultTeletypeConfig())
}

// TeletypeWriteConfig writes text with custom teletype configuration
func TeletypeWriteConfig(w io.Writer, text string, config TeletypeConfig) error {
	if w == nil {
		w = os.Stderr
	}

	// Determine if teletype should be enabled
	enabled := true
	if config.Enabled != nil {
		enabled = *config.Enabled
	} else {
		// Auto-detect: enable only in TTY without ACCESSIBLE mode
		enabled = tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == ""
	}

	// If disabled or empty text, just write directly
	if !enabled || text == "" {
		_, err := fmt.Fprint(w, text)
		return err
	}

	// Configure typing speed
	charsPerSecond := config.CharsPerSecond
	if charsPerSecond <= 0 {
		charsPerSecond = 120
	}

	// Calculate tick parameters for smooth animation
	// We want to update frequently enough for smooth animation
	// but not so often that it's wasteful
	ticksPerSecond := 60 // 60 FPS for smooth animation
	charsPerTick := charsPerSecond / ticksPerSecond
	if charsPerTick < 1 {
		charsPerTick = 1
		ticksPerSecond = charsPerSecond
	}
	tickDuration := time.Second / time.Duration(ticksPerSecond)

	// Create the model
	model := teletypeModel{
		text:         text,
		currentIndex: 0,
		output:       w,
		charsPerTick: charsPerTick,
		tickDuration: tickDuration,
	}

	// Run the program without the default renderer
	program := tea.NewProgram(model, tea.WithOutput(w), tea.WithoutRenderer())
	_, err := program.Run()
	return err
}

// TeletypeWriteln writes text with a teletype effect followed by a newline
func TeletypeWriteln(w io.Writer, text string) error {
	return TeletypeWrite(w, text+"\n")
}

// TeletypeWritelnConfig writes text with a newline using custom configuration
func TeletypeWritelnConfig(w io.Writer, text string, config TeletypeConfig) error {
	return TeletypeWriteConfig(w, text+"\n", config)
}

// TeletypeWriteLines writes multiple lines with teletype effect and appropriate pauses
func TeletypeWriteLines(w io.Writer, lines ...string) error {
	for _, line := range lines {
		if err := TeletypeWriteln(w, line); err != nil {
			return err
		}
		// Small pause between lines for readability
		if tty.IsStderrTerminal() && os.Getenv("ACCESSIBLE") == "" {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}

// TeletypeSection writes a section of text with a header and content
// The header is displayed instantly, and the content uses teletype effect
func TeletypeSection(w io.Writer, header string, content ...string) error {
	if w == nil {
		w = os.Stderr
	}

	// Display header instantly
	fmt.Fprintln(w, header)

	// Display content with teletype effect
	fullContent := strings.Join(content, "\n")
	if fullContent != "" {
		if err := TeletypeWriteln(w, fullContent); err != nil {
			return err
		}
	}

	// Add blank line after section
	fmt.Fprintln(w, "")
	return nil
}
