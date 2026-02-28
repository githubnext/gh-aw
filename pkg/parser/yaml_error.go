package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var yamlErrorLog = logger.New("parser:yaml_error")

// Package-level compiled regex patterns for better performance
var (
	lineColPatternParser = regexp.MustCompile(`^\[(\d+):(\d+)\]`)
	definedAtPattern     = regexp.MustCompile(`already defined at \[(\d+):(\d+)\]`)
	sourceLinePattern    = regexp.MustCompile(`(?m)^(>?\s*)(\d+)(\s*\|)`)
)

// FormatYAMLError formats a YAML error with source code context using yaml.FormatError()
// frontmatterLineOffset is the line number where the frontmatter content begins in the document (1-based)
// Returns the formatted error string with line numbers adjusted for frontmatter position
func FormatYAMLError(err error, frontmatterLineOffset int, sourceYAML string) string {
	yamlErrorLog.Printf("Formatting YAML error with yaml.FormatError(): offset=%d", frontmatterLineOffset)

	// Use goccy/go-yaml's native FormatError for consistent formatting with source context
	// colored=false to avoid ANSI escape codes, inclSource=true to include source lines
	formatted := yaml.FormatError(err, false, true)

	// Adjust line numbers in the formatted output to account for frontmatter position
	if frontmatterLineOffset > 1 {
		formatted = adjustLineNumbersInFormattedError(formatted, frontmatterLineOffset-1)
	}

	return formatted
}

// adjustLineNumbersInFormattedError adjusts line numbers in yaml.FormatError() output
// by adding the specified offset to all line numbers
func adjustLineNumbersInFormattedError(formatted string, offset int) string {
	if offset == 0 {
		return formatted
	}

	// Pattern to match line numbers in the format:
	// [line:col] at the start
	// "   1 | content" in the source context
	// ">  2 | content" with the error marker

	// Adjust [line:col] format at the start
	formatted = lineColPatternParser.ReplaceAllStringFunc(formatted, func(match string) string {
		var line, col int
		if _, err := fmt.Sscanf(match, "[%d:%d]", &line, &col); err == nil {
			return fmt.Sprintf("[%d:%d]", line+offset, col)
		}
		return match
	})

	// Adjust line numbers in "already defined at [line:col]" references
	formatted = definedAtPattern.ReplaceAllStringFunc(formatted, func(match string) string {
		var line, col int
		if _, err := fmt.Sscanf(match, "already defined at [%d:%d]", &line, &col); err == nil {
			return fmt.Sprintf("already defined at [%d:%d]", line+offset, col)
		}
		return match
	})

	// Adjust line numbers in source context lines (both "   1 |" and ">  1 |" formats)
	formatted = sourceLinePattern.ReplaceAllStringFunc(formatted, func(match string) string {
		var line int
		if strings.Contains(match, ">") {
			if _, err := fmt.Sscanf(match, "> %d |", &line); err == nil {
				return fmt.Sprintf(">%3d |", line+offset)
			}
		} else {
			if _, err := fmt.Sscanf(match, "%d |", &line); err == nil {
				return fmt.Sprintf("%4d |", line+offset)
			}
		}
		// If we can't parse it, extract parts manually
		parts := strings.Split(match, "|")
		if len(parts) == 2 {
			prefix := strings.TrimRight(parts[0], "0123456789")
			lineStr := strings.Trim(parts[0][len(prefix):], " ")
			if n, err := fmt.Sscanf(lineStr, "%d", &line); err == nil && n == 1 {
				if strings.Contains(prefix, ">") {
					return fmt.Sprintf(">%3d |", line+offset)
				}
				return fmt.Sprintf("%4d |", line+offset)
			}
		}
		return match
	})

	return formatted
}

// ExtractYAMLError extracts line and column information from YAML parsing errors
// frontmatterLineOffset is the line number where the frontmatter content begins in the document (1-based)
// This allows proper line number reporting when frontmatter is not at the beginning of the document
//
// NOTE: This function is kept for backward compatibility. New code should use FormatYAMLError()
// which leverages yaml.FormatError() for better error messages with source context.

// extractFromGoccyFormat extracts line/column from goccy/go-yaml's [line:column] message format

// extractFromStringParsing provides fallback string parsing for other YAML libraries
