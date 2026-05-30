package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var frontmatterErrorLog = logger.New("workflow:frontmatter_error")

// frontmatterParseErrPrefix is the string prefix that ExtractFrontmatterFromContent
// prepends to the formatted yaml.FormatError() output when a YAML syntax error occurs.
// It is used as a sentinel to detect whether a frontmatter error already carries
// formatted YAML position information.
const frontmatterParseErrPrefix = "failed to parse frontmatter:\n"

// Package-level compiled regex patterns for better performance
var (
	lineColPattern       = regexp.MustCompile(`\[(\d+):(\d+)\]\s*(.+)`)
	sourceContextPattern = regexp.MustCompile(`\n(\s+\d+\s*\|)`)
)

// readSourceContextLines extracts source lines around a target line (±3 lines)
// from the given file content for Rust-style error rendering.
// The returned slice is suitable for console.CompilerError.Context.
func readSourceContextLines(content []byte, targetLine int) []string {
	allLines := strings.Split(string(content), "\n")
	contextSize := 7 // ±3 lines around the error

	// Calculate the expected first line of the context window
	expectedFirstLine := targetLine - contextSize/2
	fileStart := max(0, expectedFirstLine-1) // 0-indexed, clamped to file start

	var contextLines []string

	// Pad with empty strings for lines that are before the file
	for lineNum := expectedFirstLine; lineNum < 1; lineNum++ {
		contextLines = append(contextLines, "")
	}

	// Add real lines from the file
	fileEnd := min(len(allLines), fileStart+contextSize-len(contextLines))
	for i := fileStart; i < fileEnd; i++ {
		contextLines = append(contextLines, allLines[i])
	}

	return contextLines
}

// findFrontmatterFieldLine searches frontmatterLines for a line whose first
// non-space key matches fieldName (e.g., "engine") and returns the 1-based
// document line number.  frontmatterStart is the 1-based line number of the
// first frontmatter line (i.e., the line immediately after the opening "---").
// Returns 0 if the field is not found.
//
// Only top-level (non-indented) keys are matched.  Nested values that happen
// to contain the field name are ignored.
func findFrontmatterFieldLine(frontmatterLines []string, frontmatterStart int, fieldName string) int {
	prefix := fieldName + ":"
	for i, line := range frontmatterLines {
		// Match only non-indented lines so nested YAML values are not confused
		// with top-level keys (e.g. "  engine: ..." inside a mapping is ignored).
		if strings.HasPrefix(line, prefix) {
			return frontmatterStart + i
		}
	}
	return 0
}

// createFrontmatterError creates a detailed error for frontmatter parsing issues
// frontmatterLineOffset is the line number where the frontmatter content begins (1-based)
// Returns error in VSCode-compatible format: filename:line:column: error message
func (c *Compiler) createFrontmatterError(filePath, content string, err error, frontmatterLineOffset int) error {
	frontmatterErrorLog.Printf("Creating frontmatter error for file: %s, offset: %d", filePath, frontmatterLineOffset)

	errorStr := err.Error()

	// Check if error already contains formatted yaml.FormatError() output with source context
	// yaml.FormatError() produces output like "failed to parse frontmatter:\n[line:col] message\n>  line | content..."
	if strings.Contains(errorStr, frontmatterParseErrPrefix+"[") && (strings.Contains(errorStr, "\n>") || strings.Contains(errorStr, "|")) {
		// Extract line and column from the formatted error for VSCode compatibility
		// Pattern: [line:col] message
		if matches := lineColPattern.FindStringSubmatch(errorStr); len(matches) >= 4 {
			line := matches[1]
			col := matches[2]
			message := matches[3]
			// Extract just the first line of the message (before newline)
			if idx := strings.Index(message, "\n"); idx != -1 {
				message = message[:idx]
			}
			// Translate raw YAML parser messages to user-friendly plain English.
			// Uses the shared translation table from pkg/parser to keep both code paths in sync.
			message = parser.TranslateYAMLMessage(message)

			// Format as: filename:line:column: error: message
			// This is compatible with VSCode's problem matcher
			vscodeFormat := fmt.Sprintf("%s:%s:%s: error: %s", filePath, line, col, message)

			// Extract just the source context lines (skip the [line:col] message line to avoid duplication)
			// Find the first line that starts with whitespace + digit + | (source context line)
			if loc := sourceContextPattern.FindStringIndex(errorStr); loc != nil {
				// Extract from the first source context line to the end
				context := errorStr[loc[0]+1:] // +1 to skip the leading newline
				// Return VSCode-compatible format on first line, followed by source context only
				frontmatterErrorLog.Print("Formatting error for VSCode compatibility")
				return parser.NewFormattedParserError(fmt.Sprintf("%s\n%s", vscodeFormat, context))
			}

			// If we can't extract source context, return just the VSCode format
			return parser.NewFormattedParserError(vscodeFormat)
		}

		// Fallback if we can't parse the line/col: emit an IDE-compatible error
		// pointing to the frontmatter start so the developer is at least brought to
		// the right section rather than the useless line 1, col 1.
		frontmatterErrorLog.Print("Could not extract line/col from formatted error, falling back to frontmatter start")
		fallbackMsg := "failed to parse YAML frontmatter"
		// Try to surface a single-line description from the raw error text.
		if _, rest, found := strings.Cut(errorStr, frontmatterParseErrPrefix); found {
			firstLine, _, _ := strings.Cut(rest, "\n")
			if translated := parser.TranslateYAMLMessage(strings.TrimSpace(firstLine)); translated != "" {
				fallbackMsg = "failed to parse YAML frontmatter: " + translated
			}
		}
		fallbackFmt := fmt.Sprintf("%s:%d:1: error: %s", filePath, frontmatterLineOffset, fallbackMsg)
		return parser.NewFormattedParserError(fallbackFmt)
	}

	// Fallback: if not already formatted, create a FormattedParserError pointing to the
	// frontmatter start so the IDE navigates to the right file and section rather than
	// defaulting to line 1, col 1.
	frontmatterErrorLog.Printf("Using fallback error message: %v", err)
	fallbackFmt := fmt.Sprintf("%s:%d:1: error: %s", filePath, frontmatterLineOffset, err.Error())
	return parser.NewFormattedParserError(fallbackFmt)
}
