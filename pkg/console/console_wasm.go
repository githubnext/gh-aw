//go:build js || wasm

package console

import (
	"fmt"
	"os"
	"strings"
)

func isTTY() bool {
	return false
}

func FormatError(err CompilerError) string {
	var output strings.Builder

	var prefix string
	switch err.Type {
	case "warning":
		prefix = "warning"
	case "info":
		prefix = "info"
	default:
		prefix = "error"
	}

	if err.Position.File != "" {
		relativePath := ToRelativePath(err.Position.File)
		location := fmt.Sprintf("%s:%d:%d:", relativePath, err.Position.Line, err.Position.Column)
		output.WriteString(location)
		output.WriteString(" ")
	}

	output.WriteString(prefix + ": ")
	output.WriteString(err.Message)
	output.WriteString("\n")

	if len(err.Context) > 0 && err.Position.Line > 0 {
		maxLineNum := err.Position.Line + len(err.Context)/2
		lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))
		for i, line := range err.Context {
			lineNum := err.Position.Line - len(err.Context)/2 + i
			if lineNum < 1 {
				continue
			}
			lineNumStr := fmt.Sprintf("%*d", lineNumWidth, lineNum)
			output.WriteString(lineNumStr)
			output.WriteString(" | ")
			output.WriteString(line)
			output.WriteString("\n")
			if lineNum == err.Position.Line && err.Position.Column > 0 && err.Position.Column <= len(line) {
				wordEnd := findWordEnd(line, err.Position.Column-1)
				wordLength := wordEnd - (err.Position.Column - 1)
				padding := strings.Repeat(" ", lineNumWidth+3+err.Position.Column-1)
				pointer := strings.Repeat("^", wordLength)
				output.WriteString(padding)
				output.WriteString(pointer)
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

func FormatSuccessMessage(message string) string  { return "✓ " + message }
func FormatInfoMessage(message string) string     { return "ℹ " + message }
func FormatWarningMessage(message string) string  { return "⚠ " + message }
func FormatErrorMessage(message string) string    { return "✗ " + message }
func FormatLocationMessage(message string) string { return "📁 " + message }
func FormatCommandMessage(command string) string  { return "⚡ " + command }
func FormatProgressMessage(message string) string { return "🔨 " + message }
func FormatPromptMessage(message string) string   { return "❓ " + message }
func FormatCountMessage(message string) string    { return "📊 " + message }
func FormatVerboseMessage(message string) string  { return "🔍 " + message }
func FormatListHeader(header string) string       { return header }
func FormatListItem(item string) string           { return "  • " + item }
func FormatSectionHeader(header string) string    { return header }

func RenderTable(config TableConfig) string {
	if len(config.Headers) == 0 {
		return ""
	}
	var output strings.Builder
	if config.Title != "" {
		output.WriteString(config.Title)
		output.WriteString("\n")
	}
	output.WriteString(strings.Join(config.Headers, "\t"))
	output.WriteString("\n")
	allRows := config.Rows
	if config.ShowTotal && len(config.TotalRow) > 0 {
		allRows = append(allRows, config.TotalRow)
	}
	for _, row := range allRows {
		output.WriteString(strings.Join(row, "\t"))
		output.WriteString("\n")
	}
	return output.String()
}

func RenderTitleBox(title string, width int) []string {
	separator := strings.Repeat("=", width)
	return []string{separator, "  " + title, separator}
}

func RenderErrorBox(title string) []string {
	return []string{FormatErrorMessage(title)}
}

func RenderInfoSection(content string) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "  " + line
	}
	return result
}

func RenderComposedSections(sections []string) {
	fmt.Fprintln(os.Stderr, "")
	for _, section := range sections {
		fmt.Fprintln(os.Stderr, section)
	}
	fmt.Fprintln(os.Stderr, "")
}

func RenderTree(root TreeNode) string {
	return renderTreeSimple(root, "", true)
}
