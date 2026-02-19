package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterErrorLog = logger.New("workflow:frontmatter_error")

// Package-level compiled regex patterns for better performance
var (
	lineColPattern       = regexp.MustCompile(`\[(\d+):(\d+)\]\s*(.+)`)
	sourceContextPattern = regexp.MustCompile(`\n(\s+\d+\s*\|)`)
)

// yamlErrorTranslations maps raw goccy/go-yaml internal messages to user-friendly plain English.
// These messages are parser internals that are not helpful to end users.
var yamlErrorTranslations = []struct {
	pattern     string
	translation string
}{
	{
		"non-map value is specified",
		"Invalid YAML syntax: expected 'key: value' format (did you forget a colon after the key?)",
	},
	{
		"mapping values are not allowed",
		"Invalid YAML syntax: unexpected ':' — check your indentation",
	},
	{
		"did not find expected",
		"Invalid YAML syntax: check indentation or missing key",
	},
}

// translateYAMLMessage converts raw YAML parser messages to user-friendly plain English.
// This prevents internal library jargon from reaching the end user.
func translateYAMLMessage(message string) string {
	for _, t := range yamlErrorTranslations {
		if strings.Contains(message, t.pattern) {
			return t.translation
		}
	}
	return message
}

// createFrontmatterError creates a detailed error for frontmatter parsing issues
// frontmatterLineOffset is the line number where the frontmatter content begins (1-based)
// Returns error in VSCode-compatible format: filename:line:column: error message
func (c *Compiler) createFrontmatterError(filePath, content string, err error, frontmatterLineOffset int) error {
	frontmatterErrorLog.Printf("Creating frontmatter error for file: %s, offset: %d", filePath, frontmatterLineOffset)

	errorStr := err.Error()

	// Check if error already contains formatted yaml.FormatError() output with source context
	// yaml.FormatError() produces output like "failed to parse frontmatter:\n[line:col] message\n>  line | content..."
	if strings.Contains(errorStr, "failed to parse frontmatter:\n[") && (strings.Contains(errorStr, "\n>") || strings.Contains(errorStr, "|")) {
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
			// Translate raw YAML parser messages to user-friendly plain English
			message = translateYAMLMessage(message)

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
				return fmt.Errorf("%s\n%s", vscodeFormat, context)
			}

			// If we can't extract source context, return just the VSCode format
			return fmt.Errorf("%s", vscodeFormat)
		}

		// Fallback if we can't parse the line/col
		frontmatterErrorLog.Print("Could not extract line/col from formatted error")
		return fmt.Errorf("%s: %v", filePath, err)
	}

	// Fallback: if not already formatted, return with filename prefix
	frontmatterErrorLog.Printf("Using fallback error message: %v", err)
	return fmt.Errorf("%s: failed to extract frontmatter: %w", filePath, err)
}
