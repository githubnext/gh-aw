package console

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/styles"
	"github.com/githubnext/gh-aw/pkg/tty"
)

var consoleLog = logger.New("console:console")

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

// ANSI escape sequences for terminal control
var clearScreenSequence = "\033[2J\033[H" // Clear screen and move cursor to home position

// isTTY checks if stdout is a terminal
func isTTY() bool {
	return tty.IsStdoutTerminal()
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
	consoleLog.Printf("Formatting error: type=%s, file=%s, line=%d", err.Type, err.Position.File, err.Position.Line)
	var output strings.Builder

	// Get style based on error type
	var typeStyle lipgloss.Style
	var prefix string
	switch err.Type {
	case "warning":
		typeStyle = styles.Warning
		prefix = "warning"
	case "info":
		typeStyle = styles.Info
		prefix = "info"
	default:
		typeStyle = styles.Error
		prefix = "error"
	}

	// IDE-parseable format: file:line:column: type: message
	if err.Position.File != "" {
		relativePath := ToRelativePath(err.Position.File)
		location := fmt.Sprintf("%s:%d:%d:",
			relativePath,
			err.Position.Line,
			err.Position.Column)
		output.WriteString(applyStyle(styles.FilePath, location))
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
		output.WriteString(applyStyle(styles.LineNumber, lineNumStr))
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

				output.WriteString(applyStyle(styles.ContextLine, before))
				output.WriteString(applyStyle(styles.Highlight, highlightedPart))
				output.WriteString(applyStyle(styles.ContextLine, after))
			} else {
				// Highlight entire line if no specific column or invalid column
				output.WriteString(applyStyle(styles.Highlight, line))
			}
		} else {
			output.WriteString(applyStyle(styles.ContextLine, line))
		}
		output.WriteString("\n")

		// Add pointer to error position (only when highlighting specific column)
		if lineNum == err.Position.Line && err.Position.Column > 0 && err.Position.Column <= len(line) {
			// Create pointer line that spans the highlighted word
			wordEnd := findWordEnd(line, err.Position.Column-1)
			wordLength := wordEnd - (err.Position.Column - 1)

			padding := strings.Repeat(" ", lineNumWidth+3+err.Position.Column-1)
			pointer := applyStyle(styles.Error, strings.Repeat("^", wordLength))
			output.WriteString(padding)
			output.WriteString(pointer)
			output.WriteString("\n")
		}
	}

	return output.String()
}

// FormatSuccessMessage formats a success message with styling
func FormatSuccessMessage(message string) string {
	return applyStyle(styles.Success, "✓ ") + message
}

// FormatInfoMessage formats an informational message
func FormatInfoMessage(message string) string {
	return applyStyle(styles.Info, "ℹ ") + message
}

// FormatWarningMessage formats a warning message
func FormatWarningMessage(message string) string {
	return applyStyle(styles.Warning, "⚠ ") + message
}

// TableConfig represents configuration for table rendering
type TableConfig struct {
	Headers   []string
	Rows      [][]string
	Title     string
	ShowTotal bool
	TotalRow  []string
}

// RenderTable renders a formatted table using lipgloss/table package
func RenderTable(config TableConfig) string {
	if len(config.Headers) == 0 {
		consoleLog.Print("No headers provided for table rendering")
		return ""
	}

	consoleLog.Printf("Rendering table: title=%s, columns=%d, rows=%d", config.Title, len(config.Headers), len(config.Rows))
	var output strings.Builder

	// Title
	if config.Title != "" {
		output.WriteString(applyStyle(styles.TableTitle, config.Title))
		output.WriteString("\n")
	}

	// Build rows including total row if specified
	allRows := config.Rows
	if config.ShowTotal && len(config.TotalRow) > 0 {
		allRows = append(allRows, config.TotalRow)
	}

	// Determine row count for styling purposes
	dataRowCount := len(config.Rows)

	// Create style function that applies different styles based on row type
	styleFunc := func(row, col int) lipgloss.Style {
		if !isTTY() {
			return lipgloss.NewStyle()
		}
		if row == table.HeaderRow {
			return styles.TableHeader
		}
		// If we have a total row and this is the last row
		if config.ShowTotal && len(config.TotalRow) > 0 && row == dataRowCount {
			return styles.TableTotal
		}
		return styles.TableCell
	}

	// Create table with lipgloss/table package
	t := table.New().
		Headers(config.Headers...).
		Rows(allRows...).
		Border(styles.NormalBorder).
		BorderStyle(styles.TableBorder).
		StyleFunc(styleFunc)

	output.WriteString(t.String())
	output.WriteString("\n")

	return output.String()
}

// FormatLocationMessage formats a file/directory location message
func FormatLocationMessage(message string) string {
	return applyStyle(styles.Location, "📁 ") + message
}

// FormatCommandMessage formats a command execution message
func FormatCommandMessage(command string) string {
	return applyStyle(styles.Command, "⚡ ") + command
}

// FormatProgressMessage formats a progress/activity message
func FormatProgressMessage(message string) string {
	return applyStyle(styles.Progress, "🔨 ") + message
}

// FormatPromptMessage formats a user prompt message
func FormatPromptMessage(message string) string {
	return applyStyle(styles.Prompt, "❓ ") + message
}

// FormatCountMessage formats a count/numeric status message
func FormatCountMessage(message string) string {
	return applyStyle(styles.Count, "📊 ") + message
}

// FormatVerboseMessage formats verbose debugging output
func FormatVerboseMessage(message string) string {
	return applyStyle(styles.Verbose, "🔍 ") + message
}

// FormatListHeader formats a section header for lists
func FormatListHeader(header string) string {
	return applyStyle(styles.ListHeader, header)
}

// FormatListItem formats an item in a list
func FormatListItem(item string) string {
	return applyStyle(styles.ListItem, "  • "+item)
}

// FormatErrorMessage formats a simple error message (for stderr output)
func FormatErrorMessage(message string) string {
	return applyStyle(styles.Error, "✗ ") + message
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

// RenderList renders a simple list with the specified enumerator
// Enumerator options: "bullet", "dash", "asterisk", "arabic", "roman", "alphabet"
// If TTY is not detected, returns plain text without styling
func RenderList(items []string, enumerator string) string {
	if len(items) == 0 {
		return ""
	}

	consoleLog.Printf("Rendering list: enumerator=%s, items=%d", enumerator, len(items))

	// Convert strings to any for lipgloss/list
	listItems := make([]any, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create the list
	l := list.New(listItems...)

	// Set enumerator based on type
	switch enumerator {
	case "bullet":
		l = l.Enumerator(list.Bullet)
	case "dash":
		l = l.Enumerator(list.Dash)
	case "asterisk":
		l = l.Enumerator(list.Asterisk)
	case "arabic":
		l = l.Enumerator(list.Arabic)
	case "roman":
		l = l.Enumerator(list.Roman)
	case "alphabet":
		l = l.Enumerator(list.Alphabet)
	default:
		// Default to bullet
		l = l.Enumerator(list.Bullet)
	}

	// Apply styling if TTY
	if isTTY() {
		l = l.EnumeratorStyle(styles.ListEnumerator).
			ItemStyle(styles.ListItem)
	}

	return l.String()
}

// RenderNestedList renders a hierarchical list where each key has nested items
// If TTY is not detected, returns plain text without styling
func RenderNestedList(sections map[string][]string) string {
	if len(sections) == 0 {
		return ""
	}

	consoleLog.Printf("Rendering nested list: sections=%d", len(sections))

	var result strings.Builder

	// Iterate over sections (order not guaranteed in maps, but that's okay for this use case)
	for sectionTitle, items := range sections {
		// Add section header
		if isTTY() {
			result.WriteString(styles.ListHeader.Render(sectionTitle))
		} else {
			result.WriteString(sectionTitle)
		}
		result.WriteString("\n")

		// Create nested list for items
		if len(items) > 0 {
			listItems := make([]any, len(items))
			for i, item := range items {
				listItems[i] = item
			}

			nestedList := list.New(listItems...).
				Enumerator(list.Bullet)

			// Apply styling if TTY
			if isTTY() {
				nestedList = nestedList.EnumeratorStyle(styles.ListEnumerator).
					ItemStyle(styles.ListItem)
			}

			result.WriteString(nestedList.String())
			result.WriteString("\n")
		}
	}

	return result.String()
}
