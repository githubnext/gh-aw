package console

import (
	"encoding/json"
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

	// ANSI escape sequences for terminal control
	clearScreenSequence = "\033[2J\033[H" // Clear screen and move cursor to home position
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

// findWordEnd finds the end of a word starting at the given position
// A word ends at whitespace, punctuation, or end of line
func findWordEnd(line string, start int) int {
	if start >= len(line) {
		return len(line)
	}

	end := start
	for end < len(line) {
		char := line[end]
		// Stop at whitespace or common punctuation that would end a YAML key/value
		if char == ' ' || char == '\t' || char == ':' || char == '\n' || char == '\r' {
			break
		}
		end++
	}

	return end
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
			// For JSON validation errors, highlight from column to end of word
			if err.Position.Column > 0 && err.Position.Column <= len(line) {
				before := line[:err.Position.Column-1]

				// Find the end of the word starting at the column position
				wordEnd := findWordEnd(line, err.Position.Column-1)
				highlightedPart := line[err.Position.Column-1 : wordEnd]
				after := ""
				if wordEnd < len(line) {
					after = line[wordEnd:]
				}

				output.WriteString(applyStyle(contextLineStyle, before))
				output.WriteString(applyStyle(highlightStyle, highlightedPart))
				output.WriteString(applyStyle(contextLineStyle, after))
			} else {
				// Highlight entire line if no specific column or invalid column
				output.WriteString(applyStyle(highlightStyle, line))
			}
		} else {
			output.WriteString(applyStyle(contextLineStyle, line))
		}
		output.WriteString("\n")

		// Add pointer to error position (only when highlighting specific column)
		if lineNum == err.Position.Line && err.Position.Column > 0 && err.Position.Column <= len(line) {
			// Create pointer line that spans the highlighted word
			wordEnd := findWordEnd(line, err.Position.Column-1)
			wordLength := wordEnd - (err.Position.Column - 1)

			padding := strings.Repeat(" ", lineNumWidth+3+err.Position.Column-1)
			pointer := applyStyle(errorStyle, strings.Repeat("^", wordLength))
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
				Foreground(lipgloss.Color("#6272A4"))

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

// FormatNestedError formats an error message with nested errors shown on separate lines
// with visual separators similar to Python's traceback format.
// Uses a robust splitting strategy that avoids breaking on colons within URLs, file paths, etc.
//
// Example input:  "failed to compile: failed to parse: invalid syntax"
// Example output:
//
//	✗ failed to compile
//	  ╰─▶ failed to parse
//	      ╰─▶ invalid syntax
func FormatNestedError(message string) string {
	if message == "" {
		return ""
	}

	// Split the error message intelligently to avoid breaking on colons in URLs, file paths, etc.
	parts := splitNestedErrors(message)
	if len(parts) <= 1 {
		// No nested errors, return simple formatted message
		return FormatErrorMessage(message)
	}

	var output strings.Builder
	for i, part := range parts {
		if i == 0 {
			// First part gets the error icon
			output.WriteString(FormatErrorMessage(part))
		} else {
			// Nested parts with visual arrow indicators
			output.WriteString("\n")
			// Add indentation spaces for nesting depth (2 spaces per level)
			output.WriteString(strings.Repeat("  ", i))
			// Add visual arrow separator
			output.WriteString("╰─▶ ")
			output.WriteString(part)
		}
	}

	return output.String()
}

// splitNestedErrors intelligently splits an error message into nested error parts.
// It uses heuristics to identify real error boundaries vs. colons in content like URLs or file paths.
func splitNestedErrors(message string) []string {
	// First, try simple split
	candidateParts := strings.Split(message, ": ")
	if len(candidateParts) <= 1 {
		return candidateParts
	}

	// Error-indicating prefixes that suggest a new error boundary
	errorPrefixes := []string{
		"failed to ",
		"could not ",
		"unable to ",
		"cannot ",
		"error ",
		"failed ",
	}

	// Check if a string looks like it starts a new error message
	startsNewError := func(s string) bool {
		trimmed := strings.TrimSpace(s)
		lower := strings.ToLower(trimmed)

		// Check for common error prefixes
		for _, prefix := range errorPrefixes {
			if strings.HasPrefix(lower, prefix) {
				return true
			}
		}

		// Single word errors like "error", "warning"
		if len(strings.Fields(lower)) == 1 && (lower == "error" || lower == "warning") {
			return true
		}

		// Short phrases that look like error messages (not URLs or file paths)
		// These typically don't contain "/" or "=" and are reasonable length
		if !strings.Contains(trimmed, "/") &&
			!strings.Contains(trimmed, "=") &&
			!strings.Contains(trimmed, "http") &&
			len(trimmed) > 0 && len(trimmed) < 200 {
			// If it doesn't look like a URL or file path, it's likely an error message
			return true
		}

		return false
	}

	// Merge parts that look like they're part of URLs or file paths
	var result []string
	var current strings.Builder

	for i, part := range candidateParts {
		if i == 0 {
			// First part is always the start
			current.WriteString(part)
		} else {
			prevPart := candidateParts[i-1]

			// Check if the previous part looks like it's incomplete (URL, file path)
			// Common patterns: "https", "http", "ftp", file paths ending without extension
			prevLower := strings.ToLower(strings.TrimSpace(prevPart))
			isIncompleteURL := strings.HasSuffix(prevLower, "https") ||
				strings.HasSuffix(prevLower, "http") ||
				strings.HasSuffix(prevLower, "ftp")

			// If previous looks incomplete, merge this part
			if isIncompleteURL {
				current.WriteString(": ")
				current.WriteString(part)
			} else if startsNewError(part) {
				// This looks like a new error boundary
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
				current.WriteString(part)
			} else {
				// Not clearly a new error, merge with previous
				current.WriteString(": ")
				current.WriteString(part)
			}
		}
	}

	// Add the last part
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// FormatErrorWithSuggestions formats an error message with actionable suggestions
func FormatErrorWithSuggestions(message string, suggestions []string) string {
	var output strings.Builder
	output.WriteString(FormatErrorMessage(message))

	if len(suggestions) > 0 {
		output.WriteString("\n\nSuggestions:\n")
		for _, suggestion := range suggestions {
			output.WriteString("  • " + suggestion + "\n")
		}
	}

	return output.String()
}

// RenderTableAsJSON renders a table configuration as JSON
// This converts the table structure to a JSON array of objects
func RenderTableAsJSON(config TableConfig) (string, error) {
	if len(config.Headers) == 0 {
		return "[]", nil
	}

	// Create array of objects, where each object has header names as keys
	var result []map[string]string
	for _, row := range config.Rows {
		obj := make(map[string]string)
		for i, cell := range row {
			if i < len(config.Headers) {
				// Convert header to lowercase with underscores for JSON keys
				key := strings.ToLower(strings.ReplaceAll(config.Headers[i], " ", "_"))
				obj[key] = cell
			}
		}
		result = append(result, obj)
	}

	// Marshal to JSON with indentation
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal table to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// ClearScreen clears the terminal screen if stdout is a TTY
// Uses ANSI escape codes for cross-platform compatibility
func ClearScreen() {
	if isTTY() {
		fmt.Print(clearScreenSequence)
	}
}
