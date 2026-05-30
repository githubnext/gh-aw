package console

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/tty"
)

// ANSI escape sequences for terminal control
const (
	// ansiClearScreen clears the screen and moves cursor to home position
	ansiClearScreen = "\033[H\033[2J"

	// ansiClearLine clears from cursor to end of line
	ansiClearLine = "\033[K"

	// ansiCarriageReturn moves cursor to start of current line
	ansiCarriageReturn = "\r"
)

// ClearScreen clears the terminal screen if stderr is a TTY
// Uses ANSI escape codes for cross-platform compatibility
func ClearScreen() {
	if tty.IsStderrTerminal() {
		fmt.Fprint(os.Stderr, ansiClearScreen)
	}
}

// ClearLine clears the current line in the terminal if stderr is a TTY
// Uses ANSI escape codes: \r moves cursor to start, \033[K clears to end of line
func ClearLine() {
	if tty.IsStderrTerminal() {
		fmt.Fprintf(os.Stderr, "%s%s", ansiCarriageReturn, ansiClearLine)
	}
}

// ShowWelcomeBanner clears the screen and displays the welcome banner for interactive commands.
// Use this at the start of interactive commands (add, trial, init) for a consistent experience.
func ShowWelcomeBanner(description string) {
	ClearScreen()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "🚀 Welcome to GitHub Agentic Workflows!")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, description)
	fmt.Fprintln(os.Stderr, "")
}
