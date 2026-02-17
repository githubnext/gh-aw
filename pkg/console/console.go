//go:build !js && !wasm

package console

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/tty"
)

var consoleLog = logger.New("console:console")

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

	return output.String()
}

// renderContext renders source code context with line numbers and highlighting
func renderContext(err CompilerError) string {
	var output strings.Builder

	maxLineNum := err.Position.Line + len(err.Context)/2
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	for i, line := range err.Context {
		lineNum := err.Position.Line - len(err.Context)/2 + i
		if lineNum < 1 {
			continue
		}

		lineNumStr := fmt.Sprintf("%*d", lineNumWidth, lineNum)
		output.WriteString(applyStyle(styles.LineNumber, lineNumStr))
		output.WriteString(" | ")

		if lineNum == err.Position.Line {
			if err.Position.Column > 0 && err.Position.Column <= len(line) {
				before := line[:err.Position.Column-1]
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
				output.WriteString(applyStyle(styles.Highlight, line))
			}
		} else {
			output.WriteString(applyStyle(styles.ContextLine, line))
		}
		output.WriteString("\n")

		if lineNum == err.Position.Line && err.Position.Column > 0 && err.Position.Column <= len(line) {
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

// RenderTable renders a formatted table using lipgloss/table package
func RenderTable(config TableConfig) string {
	if len(config.Headers) == 0 {
		consoleLog.Print("No headers provided for table rendering")
		return ""
	}

	consoleLog.Printf("Rendering table: title=%s, columns=%d, rows=%d", config.Title, len(config.Headers), len(config.Rows))
	var output strings.Builder

	if config.Title != "" {
		output.WriteString(applyStyle(styles.TableTitle, config.Title))
		output.WriteString("\n")
	}

	allRows := config.Rows
	if config.ShowTotal && len(config.TotalRow) > 0 {
		allRows = append(allRows, config.TotalRow)
	}

	dataRowCount := len(config.Rows)

	styleFunc := func(row, col int) lipgloss.Style {
		if !isTTY() {
			return lipgloss.NewStyle()
		}
		if row == table.HeaderRow {
			headerStyle := styles.TableHeader
			return headerStyle.PaddingLeft(1).PaddingRight(1)
		}
		if config.ShowTotal && len(config.TotalRow) > 0 && row == dataRowCount {
			totalStyle := styles.TableTotal
			return totalStyle.PaddingLeft(1).PaddingRight(1)
		}
		if row%2 == 0 {
			cellStyle := styles.TableCell
			return cellStyle.PaddingLeft(1).PaddingRight(1)
		}
		return lipgloss.NewStyle().
			Foreground(styles.ColorForeground).
			Background(styles.ColorTableAltRow).
			PaddingLeft(1).
			PaddingRight(1)
	}

	t := table.New().
		Headers(config.Headers...).
		Rows(allRows...).
		Border(styles.RoundedBorder).
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

// FormatSectionHeader formats a section header with proper styling
func FormatSectionHeader(header string) string {
	if isTTY() {
		return applyStyle(styles.Header, header)
	}
	return header
}

// RenderTitleBox renders a title with a double border box in TTY mode
func RenderTitleBox(title string, width int) []string {
	if tty.IsStderrTerminal() {
		box := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorInfo).
			Border(lipgloss.DoubleBorder(), true, false).
			Padding(0, 2).
			Width(width).
			Align(lipgloss.Center).
			Render(title)
		return []string{box}
	}

	separator := strings.Repeat("━", width)
	return []string{separator, "  " + title, separator}
}

// RenderErrorBox renders an error/warning message with a rounded border box
func RenderErrorBox(title string) []string {
	if tty.IsStderrTerminal() {
		box := lipgloss.NewStyle().
			Border(styles.RoundedBorder).
			BorderForeground(styles.ColorError).
			Padding(1, 2).
			Bold(true).
			Render(title)
		return []string{box}
	}

	return []string{
		FormatErrorMessage(title),
	}
}

// RenderInfoSection renders an info section with left border emphasis
func RenderInfoSection(content string) []string {
	if tty.IsStderrTerminal() {
		section := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(styles.ColorInfo).
			PaddingLeft(2).
			Render(content)
		return []string{section}
	}

	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "  " + line
	}
	return result
}

// RenderComposedSections composes and outputs a slice of sections to stderr
func RenderComposedSections(sections []string) {
	if tty.IsStderrTerminal() {
		plan := lipgloss.JoinVertical(lipgloss.Left, sections...)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, plan)
		fmt.Fprintln(os.Stderr, "")
	} else {
		fmt.Fprintln(os.Stderr, "")
		for _, section := range sections {
			fmt.Fprintln(os.Stderr, section)
		}
		fmt.Fprintln(os.Stderr, "")
	}
}

// RenderTree renders a hierarchical tree structure using lipgloss/tree package
func RenderTree(root TreeNode) string {
	if !isTTY() {
		return renderTreeSimple(root, "", true)
	}

	lipglossTree := buildLipglossTree(root)
	return lipglossTree.String()
}

// buildLipglossTree converts our TreeNode structure to lipgloss/tree format
func buildLipglossTree(node TreeNode) *tree.Tree {
	t := tree.Root(node.Value).
		EnumeratorStyle(styles.TreeEnumerator).
		ItemStyle(styles.TreeNode)

	if len(node.Children) > 0 {
		children := make([]any, len(node.Children))
		for i, child := range node.Children {
			if len(child.Children) > 0 {
				children[i] = buildLipglossTree(child)
			} else {
				children[i] = child.Value
			}
		}
		t.Child(children...)
	}

	return t
}
