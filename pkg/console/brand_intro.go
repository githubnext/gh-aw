package console

import (
	"fmt"
	"io"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/tty"
)

const brandIntroFrameDelay = 65 * time.Millisecond

var brandLogoStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(styles.ColorPurple)

var brandLogoFrames = [][]string{
	{
		"+-----+",
		"|     |",
		"+--+--+",
		"   |",
		"   +----+",
		"        +-----+",
		"        |     |",
		"        +-----+",
	},
	{
		"+-----+       .",
		"|     |",
		"+--+--+",
		"   |",
		"   +----+",
		"        +-----+",
		"        |     |",
		"        +-----+",
	},
	{
		"+-----+       *",
		"|     |      *+*",
		"+--+--+       *",
		"   |",
		"   +----+",
		"        +-----+",
		"        |     |",
		"        +-----+",
	},
	{
		"+-----+       +",
		"|     |      +++",
		"+--+--+       +",
		"   |",
		"   +----+",
		"        +-----+",
		"        |     |",
		"        +-----+",
	},
}

// ShowAnimatedBrandIntro clears the screen and displays a compact animated
// GitHub Agentic Workflows mark beside the supplied title and description.
// Animation is disabled outside a TTY and in accessible mode.
func ShowAnimatedBrandIntro(title, description string) {
	ClearScreen()
	out := stderrWriter()
	animate := tty.IsStderrTerminal() && !IsAccessibleMode()
	lastFrame := len(brandLogoFrames) - 1

	if animate {
		for frame := range lastFrame {
			printBrandIntroFrame(out, frame, "", "", true)
			time.Sleep(brandIntroFrameDelay)
			moveCursorToBrandIntroStart(out)
		}
	}

	printBrandIntroFrame(out, lastFrame, title, description, animate)
	fmt.Fprintln(out)
}

func printBrandIntroFrame(out io.Writer, frame int, title, description string, clearLines bool) {
	lines := formatBrandIntroFrame(frame, title, description)
	for _, line := range lines {
		if clearLines {
			fmt.Fprint(out, ansiClearLine)
		}
		fmt.Fprintln(out, line)
	}
}

func moveCursorToBrandIntroStart(out io.Writer) {
	fmt.Fprintf(out, "\033[%dA\r", len(brandLogoFrames[0]))
}

func formatBrandIntroFrame(frame int, title, description string) []string {
	logo := brandLogoFrames[frame]
	lines := make([]string, len(logo))
	for index, logoLine := range logo {
		visibleWidth := len([]rune(logoLine))
		if tty.IsStderrTerminal() {
			logoLine = brandLogoStyle.Render(logoLine)
		}
		lines[index] = padBrandLogoLine(logoLine, visibleWidth, 21)
	}

	lines[1] += title
	lines[3] += description
	return lines
}

func padBrandLogoLine(value string, visibleWidth, width int) string {
	padding := max(1, width-visibleWidth)
	return value + strings.Repeat(" ", padding)
}
