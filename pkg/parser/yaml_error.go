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
func ExtractYAMLError(err error, frontmatterLineOffset int) (line int, column int, message string) {
	yamlErrorLog.Printf("Extracting YAML error information: offset=%d", frontmatterLineOffset)
	errStr := err.Error()

	// First try to extract from goccy/go-yaml's [line:column] format
	line, column, message = extractFromGoccyFormat(errStr, frontmatterLineOffset)
	if line > 0 || column > 0 {
		yamlErrorLog.Printf("Extracted error location from goccy format: line=%d, column=%d", line, column)
		return line, column, message
	}

	// Fallback to standard YAML error string parsing for other libraries
	yamlErrorLog.Print("Falling back to string parsing for error location")
	return extractFromStringParsing(errStr, frontmatterLineOffset)
}

// extractFromGoccyFormat extracts line/column from goccy/go-yaml's [line:column] message format
func extractFromGoccyFormat(errStr string, frontmatterLineOffset int) (line int, column int, message string) {
	// Look for goccy format like "[5:10] mapping value is not allowed in this context"
	if strings.Contains(errStr, "[") && strings.Contains(errStr, "]") {
		start := strings.Index(errStr, "[")
		end := strings.Index(errStr, "]")
		if start >= 0 && end > start {
			locationPart := errStr[start+1 : end]
			messagePart := strings.TrimSpace(errStr[end+1:])

			// Parse line:column format
			if strings.Contains(locationPart, ":") {
				parts := strings.Split(locationPart, ":")
				if len(parts) == 2 {
					lineStr := strings.TrimSpace(parts[0])
					columnStr := strings.TrimSpace(parts[1])

					// Parse line and column numbers
					if _, parseErr := fmt.Sscanf(lineStr, "%d", &line); parseErr == nil {
						if _, parseErr := fmt.Sscanf(columnStr, "%d", &column); parseErr == nil {
							// Adjust line number to account for frontmatter position in file
							if line > 0 {
								line += frontmatterLineOffset - 1 // -1 because line numbers in YAML errors are 1-based relative to YAML content
							}

							// Only return valid positions - avoid returning 1,1 when location is unknown
							if line <= frontmatterLineOffset && column <= 1 {
								return 0, 0, messagePart
							}

							return line, column, messagePart
						}
					}
				}
			}
		}
	}

	return 0, 0, ""
}

// extractFromStringParsing provides fallback string parsing for other YAML libraries
func extractFromStringParsing(errStr string, frontmatterLineOffset int) (line int, column int, message string) {
	// Parse "yaml: line X: column Y: message" format (enhanced parsers that provide column info)
	if strings.Contains(errStr, "yaml: line ") && strings.Contains(errStr, "column ") {
		parts := strings.SplitN(errStr, "yaml: line ", 2)
		if len(parts) > 1 {
			lineInfo := parts[1]

			// Look for column information
			colonIndex := strings.Index(lineInfo, ":")
			if colonIndex > 0 {
				lineStr := lineInfo[:colonIndex]

				// Parse line number
				if _, parseErr := fmt.Sscanf(lineStr, "%d", &line); parseErr == nil {
					// Look for column part
					remaining := lineInfo[colonIndex+1:]
					if strings.Contains(remaining, "column ") {
						columnParts := strings.SplitN(remaining, "column ", 2)
						if len(columnParts) > 1 {
							columnInfo := columnParts[1]
							colonIndex2 := strings.Index(columnInfo, ":")
							if colonIndex2 > 0 {
								columnStr := columnInfo[:colonIndex2]
								message = strings.TrimSpace(columnInfo[colonIndex2+1:])

								// Parse column number
								if _, parseErr := fmt.Sscanf(columnStr, "%d", &column); parseErr == nil {
									// Adjust line number to account for frontmatter position in file
									line += frontmatterLineOffset - 1 // -1 because line numbers in YAML errors are 1-based relative to YAML content
									return
								}
							}
						}
					}
				}
			}
		}
	}

	// Parse "yaml: line X: message" format (standard format without column info)
	if strings.Contains(errStr, "yaml: line ") {
		parts := strings.SplitN(errStr, "yaml: line ", 2)
		if len(parts) > 1 {
			lineInfo := parts[1]
			colonIndex := strings.Index(lineInfo, ":")
			if colonIndex > 0 {
				lineStr := lineInfo[:colonIndex]
				message = strings.TrimSpace(lineInfo[colonIndex+1:])

				// Parse line number
				if _, parseErr := fmt.Sscanf(lineStr, "%d", &line); parseErr == nil {
					// Adjust line number to account for frontmatter position in file
					line += frontmatterLineOffset - 1 // -1 because line numbers in YAML errors are 1-based relative to YAML content
					// Don't default to column 1 when not provided - return 0 instead
					column = 0
					return
				}
			}
		}
	}

	// Parse "yaml: unmarshal errors: line X: message" format (multiline errors)
	if strings.Contains(errStr, "yaml: unmarshal errors:") && strings.Contains(errStr, "line ") {
		lines := strings.SplitSeq(errStr, "\n")
		for errorLine := range lines {
			errorLine = strings.TrimSpace(errorLine)
			if strings.Contains(errorLine, "line ") && strings.Contains(errorLine, ":") {
				// Extract the first line number found in the error
				parts := strings.SplitN(errorLine, "line ", 2)
				if len(parts) > 1 {
					colonIndex := strings.Index(parts[1], ":")
					if colonIndex > 0 {
						lineStr := parts[1][:colonIndex]
						restOfMessage := strings.TrimSpace(parts[1][colonIndex+1:])

						// Parse line number
						if _, parseErr := fmt.Sscanf(lineStr, "%d", &line); parseErr == nil {
							// Adjust line number to account for frontmatter position in file
							line += frontmatterLineOffset - 1 // -1 because line numbers in YAML errors are 1-based relative to YAML content
							column = 0                        // Don't default to column 1
							message = restOfMessage
							return
						}
					}
				}
			}
		}
	}

	// Fallback: return original error message with no location
	return 0, 0, errStr
}
