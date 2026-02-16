package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterErrorLog = logger.New("workflow:frontmatter_error")

// createFrontmatterError creates a detailed error for frontmatter parsing issues
// frontmatterLineOffset is the line number where the frontmatter content begins (1-based)
// Returns error in VSCode-compatible format: filename:line:column: error message
func (c *Compiler) createFrontmatterError(filePath, content string, err error, frontmatterLineOffset int) error {
	frontmatterErrorLog.Printf("Creating frontmatter error for file: %s, offset: %d", filePath, frontmatterLineOffset)

	errorStr := err.Error()

	// Check if error already contains formatted yaml.FormatError() output with source context
	// yaml.FormatError() produces output like "[line:col] message\n>  line | content..."
	if strings.Contains(errorStr, "failed to parse frontmatter:\n[") && (strings.Contains(errorStr, "\n>") || strings.Contains(errorStr, "|")) {
		// Extract line and column from the formatted error for VSCode compatibility
		// Pattern: [line:col] message
		lineColPattern := regexp.MustCompile(`\[(\d+):(\d+)\]\s*(.+)`)
		if matches := lineColPattern.FindStringSubmatch(errorStr); len(matches) >= 4 {
			line := matches[1]
			col := matches[2]
			message := matches[3]
			// Extract just the first line of the message (before newline)
			if idx := strings.Index(message, "\n"); idx != -1 {
				message = message[:idx]
			}

			// Format as: filename:line:column: error: message
			// This is compatible with VSCode's problem matcher
			vscodeFormat := fmt.Sprintf("%s:%s:%s: error: %s", filePath, line, col, message)

			// Return VSCode-compatible format on first line, followed by full context
			frontmatterErrorLog.Print("Formatting error for VSCode compatibility")
			return fmt.Errorf("%s\n%s", vscodeFormat, errorStr)
		}

		// Fallback if we can't parse the line/col
		frontmatterErrorLog.Print("Could not extract line/col from formatted error")
		return fmt.Errorf("%s: %v", filePath, err)
	}

	// Fallback: if not already formatted, return with filename prefix
	frontmatterErrorLog.Printf("Using fallback error message: %v", err)
	return fmt.Errorf("%s: failed to extract frontmatter: %w", filePath, err)
}
