package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterErrorLog = logger.New("workflow:frontmatter_error")

// createFrontmatterError creates a detailed error for frontmatter parsing issues
// frontmatterLineOffset is the line number where the frontmatter content begins (1-based)
func (c *Compiler) createFrontmatterError(filePath, content string, err error, frontmatterLineOffset int) error {
	frontmatterErrorLog.Printf("Creating frontmatter error for file: %s, offset: %d", filePath, frontmatterLineOffset)
	
	errorStr := err.Error()
	
	// Check if error already contains formatted yaml.FormatError() output with source context
	// yaml.FormatError() produces output like "[line:col] message\n>  line | content..."
	if strings.Contains(errorStr, "failed to parse frontmatter:\n[") && (strings.Contains(errorStr, "\n>") || strings.Contains(errorStr, "|")) {
		// This is already formatted by yaml.FormatError() with source context
		frontmatterErrorLog.Print("Detected yaml.FormatError() formatted output with source context")
		return fmt.Errorf("%s: %v", filePath, err)
	}
	
	// Fallback: if not already formatted, return with filename prefix
	frontmatterErrorLog.Printf("Using fallback error message: %v", err)
	return fmt.Errorf("%s: failed to extract frontmatter: %w", filePath, err)
}
