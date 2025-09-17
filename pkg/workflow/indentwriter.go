package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// IndentWriter is a utility for writing indented strings, particularly useful for generating YAML
type IndentWriter struct {
	indent           string
	excludeJSComments bool
	lines            []string
}

// NewIndentWriter creates a new IndentWriter with the specified indentation
func NewIndentWriter(indent string) *IndentWriter {
	return &IndentWriter{
		indent: indent,
		lines:  make([]string, 0),
	}
}

// NewIndentWriterWithSpaces creates a new IndentWriter with the specified number of spaces for indentation
func NewIndentWriterWithSpaces(spaces int) *IndentWriter {
	return NewIndentWriter(strings.Repeat(" ", spaces))
}

// ExcludeJSComments configures the writer to exclude JavaScript single-line comments
func (w *IndentWriter) ExcludeJSComments(exclude bool) *IndentWriter {
	w.excludeJSComments = exclude
	return w
}

// WriteString adds a string to the writer with proper indentation
// Empty lines are skipped, and existing indentation is preserved relative to the base indent
func (w *IndentWriter) WriteString(content string) *IndentWriter {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		w.WriteLine(line)
	}
	return w
}

// WriteStringf adds a formatted string to the writer with proper indentation
func (w *IndentWriter) WriteStringf(format string, args ...interface{}) *IndentWriter {
	content := fmt.Sprintf(format, args...)
	return w.WriteString(content)
}

// WriteLine adds a single line to the writer with proper indentation
// Empty/whitespace-only lines are skipped, and JavaScript comments are optionally excluded
func (w *IndentWriter) WriteLine(line string) *IndentWriter {
	// Skip empty or whitespace-only lines
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return w
	}

	// Skip JavaScript single-line comments if configured
	if w.excludeJSComments && w.isJavaScriptComment(trimmed) {
		return w
	}

	// Add the line with proper indentation
	indentedLine := w.indent + line
	w.lines = append(w.lines, indentedLine)
	return w
}

// WriteLinef adds a formatted single line to the writer with proper indentation
func (w *IndentWriter) WriteLinef(format string, args ...interface{}) *IndentWriter {
	line := fmt.Sprintf(format, args...)
	return w.WriteLine(line)
}

// Lines returns all the lines as a slice of strings, each ending with a newline
func (w *IndentWriter) Lines() []string {
	result := make([]string, len(w.lines))
	for i, line := range w.lines {
		result[i] = line + "\n"
	}
	return result
}

// String returns all the lines as a single string
func (w *IndentWriter) String() string {
	var builder strings.Builder
	for _, line := range w.lines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

// WriteToBuilder writes all lines to an existing strings.Builder
func (w *IndentWriter) WriteToBuilder(builder *strings.Builder) {
	for _, line := range w.lines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
}

// Clear removes all lines from the writer
func (w *IndentWriter) Clear() *IndentWriter {
	w.lines = w.lines[:0]
	return w
}

// LineCount returns the number of lines currently in the writer
func (w *IndentWriter) LineCount() int {
	return len(w.lines)
}

// isJavaScriptComment checks if a line is a JavaScript single-line comment
var jsCommentRegex = regexp.MustCompile(`^\s*//`)

func (w *IndentWriter) isJavaScriptComment(line string) bool {
	return jsCommentRegex.MatchString(line)
}