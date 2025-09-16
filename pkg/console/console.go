package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// ErrorPosition represents a position in a source file
type ErrorPosition struct {
	File   string
	Line   int
	Column int
}

// CompilerError represents a structured compiler error with position information
type CompilerError struct {
	Position ErrorPosition
	Type     string // "error", "warning", "info"
	Message  string
	Context  []string // Source code lines for context
	Hint     string   // Optional hint for fixing the error
}

// Styles for different error types
var (
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5555"))

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFB86C"))

	infoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#8BE9FD"))

	filePathStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#BD93F9"))

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))

	contextLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2"))

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF5555")).
			Foreground(lipgloss.Color("#282A36"))

	hintStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#50FA7B"))
)

// isTTY checks if stdout is a terminal
func isTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// applyStyle conditionally applies styling based on TTY status
func applyStyle(style lipgloss.Style, text string) string {
	if isTTY() {
		return style.Render(text)
	}
	return text
}

// ToRelativePath converts an absolute path to a relative path from the current working directory
func ToRelativePath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}

	wd, err := os.Getwd()
	if err != nil {
		// If we can't get the working directory, return the original path
		return path
	}

	relPath, err := filepath.Rel(wd, path)
	if err != nil {
		// If we can't get a relative path, return the original path
		return path
	}

	return relPath
}

// FormatError formats a CompilerError with Rust-like rendering
func FormatError(err CompilerError) string {
	var output strings.Builder

	// Get style based on error type
	var typeStyle lipgloss.Style
	var prefix string
	switch err.Type {
	case "warning":
		typeStyle = warningStyle
		prefix = "warning"
	case "info":
		typeStyle = infoStyle
		prefix = "info"
	default:
		typeStyle = errorStyle
		prefix = "error"
	}

	// IDE-parseable format: file:line:column: type: message
	if err.Position.File != "" {
		relativePath := ToRelativePath(err.Position.File)
		location := fmt.Sprintf("%s:%d:%d:",
			relativePath,
			err.Position.Line,
			err.Position.Column)
		output.WriteString(applyStyle(filePathStyle, location))
		output.WriteString(" ")
	}

	// Error type and message
	output.WriteString(applyStyle(typeStyle, prefix+":"))
	output.WriteString(" ")
	output.WriteString(err.Message)
	output.WriteString("\n")

	// Context lines (Rust-like error rendering)
	if len(err.Context) > 0 && err.Position.Line > 0 {
		output.WriteString(renderContext(err))
	}

	// Remove hints as per requirements - hints are no longer displayed

	return output.String()
}

// renderContext renders source code context with line numbers and highlighting
func renderContext(err CompilerError) string {
	var output strings.Builder

	// Calculate line number width for padding
	maxLineNum := err.Position.Line + len(err.Context)/2
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	for i, line := range err.Context {
		// Calculate actual line number (context usually centers around error line)
		lineNum := err.Position.Line - len(err.Context)/2 + i
		if lineNum < 1 {
			continue
		}

		// Format line number with proper padding
		lineNumStr := fmt.Sprintf("%*d", lineNumWidth, lineNum)
		output.WriteString(applyStyle(lineNumberStyle, lineNumStr))
		output.WriteString(" | ")

		// Highlight the error line
		if lineNum == err.Position.Line {
			// For JSON validation errors (when Column is -1), or when highlighting entire line is needed,
			// highlight the entire line instead of specific column
			if err.Position.Column <= 0 || err.Position.Column > len(line) {
				// Highlight entire line if no specific column or invalid column
				output.WriteString(applyStyle(highlightStyle, line))
			} else {
				// Highlight the specific column if available and valid
				before := line[:err.Position.Column-1]
				errorChar := string(line[err.Position.Column-1])
				after := ""
				if err.Position.Column < len(line) {
					after = line[err.Position.Column:]
				}

				output.WriteString(applyStyle(contextLineStyle, before))
				output.WriteString(applyStyle(highlightStyle, errorChar))
				output.WriteString(applyStyle(contextLineStyle, after))
			}
		} else {
			output.WriteString(applyStyle(contextLineStyle, line))
		}
		output.WriteString("\n")

		// Add pointer to error position (only when highlighting specific column)
		if lineNum == err.Position.Line && err.Position.Column > 0 && err.Position.Column <= len(line) {
			// Create pointer line
			padding := strings.Repeat(" ", lineNumWidth+3+err.Position.Column-1)
			pointer := applyStyle(errorStyle, "^")
			output.WriteString(padding)
			output.WriteString(pointer)
			output.WriteString("\n")
		}
	}

	return output.String()
}

// FormatSuccessMessage formats a success message with styling
func FormatSuccessMessage(message string) string {
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#50FA7B"))

	return applyStyle(successStyle, "✓ ") + message
}

// FormatInfoMessage formats an informational message
func FormatInfoMessage(message string) string {
	return applyStyle(infoStyle, "ℹ ") + message
}

// FormatWarningMessage formats a warning message
func FormatWarningMessage(message string) string {
	return applyStyle(warningStyle, "⚠ ") + message
}

// Table rendering styles
var (
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#34374A"))

	tableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	tableBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4"))

	tableSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#44475A"))
)

// TableConfig represents configuration for table rendering
type TableConfig struct {
	Headers   []string
	Rows      [][]string
	Title     string
	ShowTotal bool
	TotalRow  []string
}

// RenderTable renders a formatted table using lipgloss
func RenderTable(config TableConfig) string {
	if len(config.Headers) == 0 {
		return ""
	}

	var output strings.Builder

	// Title
	if config.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#50FA7B"))
		output.WriteString(applyStyle(titleStyle, config.Title))
		output.WriteString("\n")
	}

	// Calculate column widths
	colWidths := make([]int, len(config.Headers))
	for i, header := range config.Headers {
		colWidths[i] = len(header)
	}

	// Check row data for max widths
	allRows := config.Rows
	if config.ShowTotal && len(config.TotalRow) > 0 {
		allRows = append(allRows, config.TotalRow)
	}

	for _, row := range allRows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Render header
	output.WriteString(renderTableRow(config.Headers, colWidths, tableHeaderStyle))
	output.WriteString("\n")

	// Header separator
	separatorChars := make([]string, len(config.Headers))
	for i, width := range colWidths {
		separatorChars[i] = strings.Repeat("-", width)
	}
	output.WriteString(applyStyle(tableSeparatorStyle, renderTableRow(separatorChars, colWidths, tableSeparatorStyle)))
	output.WriteString("\n")

	// Render data rows
	for _, row := range config.Rows {
		output.WriteString(renderTableRow(row, colWidths, tableCellStyle))
		output.WriteString("\n")
	}

	// Total row if specified
	if config.ShowTotal && len(config.TotalRow) > 0 {
		// Total separator
		output.WriteString(applyStyle(tableSeparatorStyle, renderTableRow(separatorChars, colWidths, tableSeparatorStyle)))
		output.WriteString("\n")

		// Total row with bold styling
		totalStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#50FA7B"))
		output.WriteString(renderTableRow(config.TotalRow, colWidths, totalStyle))
		output.WriteString("\n")
	}
	output.WriteString("\n")

	return output.String()
}

// renderTableRow renders a single table row with proper spacing
func renderTableRow(cells []string, colWidths []int, style lipgloss.Style) string {
	var row strings.Builder

	for i, cell := range cells {
		if i < len(colWidths) {
			// Pad cell to column width
			paddedCell := fmt.Sprintf("%-*s", colWidths[i], cell)
			row.WriteString(applyStyle(style, paddedCell))

			// Add separator between columns (except last)
			if i < len(cells)-1 {
				row.WriteString(applyStyle(tableBorderStyle, " | "))
			}
		}
	}

	return row.String()
}

// FormatLocationMessage formats a file/directory location message
func FormatLocationMessage(message string) string {
	locationStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFB86C"))

	return applyStyle(locationStyle, "📁 ") + message
}

// FormatCommandMessage formats a command execution message
func FormatCommandMessage(command string) string {
	commandStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#BD93F9"))

	return applyStyle(commandStyle, "⚡ ") + command
}

// FormatProgressMessage formats a progress/activity message
func FormatProgressMessage(message string) string {
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1FA8C"))

	return applyStyle(progressStyle, "🔨 ") + message
}

// FormatPromptMessage formats a user prompt message
func FormatPromptMessage(message string) string {
	promptStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#50FA7B"))

	return applyStyle(promptStyle, "❓ ") + message
}

// FormatCountMessage formats a count/numeric status message
func FormatCountMessage(message string) string {
	countStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#8BE9FD"))

	return applyStyle(countStyle, "📊 ") + message
}

// FormatVerboseMessage formats verbose debugging output
func FormatVerboseMessage(message string) string {
	verboseStyle := lipgloss.NewStyle().
		Italic(true).
		Foreground(lipgloss.Color("#6272A4"))

	return applyStyle(verboseStyle, "🔍 ") + message
}

// FormatListHeader formats a section header for lists
func FormatListHeader(header string) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Foreground(lipgloss.Color("#50FA7B"))

	return applyStyle(headerStyle, header)
}

// FormatListItem formats an item in a list
func FormatListItem(item string) string {
	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2"))

	return applyStyle(itemStyle, "  • "+item)
}

// FormatErrorMessage formats a simple error message (for stderr output)
func FormatErrorMessage(message string) string {
	return applyStyle(errorStyle, "✗ ") + message
}
