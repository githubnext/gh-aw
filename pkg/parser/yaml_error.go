package parser

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ExtractYAMLError extracts line and column information from YAML parsing errors
func ExtractYAMLError(err error, frontmatterStartLine int) (line int, column int, message string) {
	// First try to unwrap and get the original error from goccy/go-yaml
	originalErr := err
	for unwrapped := errors.Unwrap(originalErr); unwrapped != nil; unwrapped = errors.Unwrap(originalErr) {
		originalErr = unwrapped
	}

	// Try to extract information from goccy/go-yaml's native error structure
	line, column, message = extractFromGoccyError(originalErr, frontmatterStartLine)
	if line > 0 || column > 0 {
		return line, column, message
	}

	// Also try the wrapped error itself in case it's a different structure
	if originalErr != err {
		line, column, message = extractFromGoccyError(err, frontmatterStartLine)
		if line > 0 || column > 0 {
			return line, column, message
		}
	}

	// Fallback to string parsing for other YAML libraries or unknown error formats
	errStr := err.Error()
	return extractFromStringParsing(errStr, frontmatterStartLine)
}

// extractFromGoccyError extracts line/column from goccy/go-yaml error structure using reflection
func extractFromGoccyError(err error, frontmatterStartLine int) (line int, column int, message string) {
	errorValue := reflect.ValueOf(err)
	if errorValue.Kind() != reflect.Ptr || errorValue.IsNil() {
		return 0, 0, ""
	}

	errorValue = errorValue.Elem()

	// Try to get Message field
	messageField := errorValue.FieldByName("Message")
	if messageField.IsValid() && messageField.Kind() == reflect.String {
		message = messageField.String()
	}

	// Try to get Token field
	tokenField := errorValue.FieldByName("Token")
	if !tokenField.IsValid() || tokenField.Kind() != reflect.Ptr || tokenField.IsNil() {
		return 0, 0, message
	}

	tokenValue := tokenField.Elem()

	// Try to get Position field from token
	positionField := tokenValue.FieldByName("Position")
	if !positionField.IsValid() || positionField.Kind() != reflect.Ptr || positionField.IsNil() {
		return 0, 0, message
	}

	positionValue := positionField.Elem()

	// Extract line and column from position
	lineField := positionValue.FieldByName("Line")
	columnField := positionValue.FieldByName("Column")

	if lineField.IsValid() && lineField.Kind() == reflect.Int {
		line = int(lineField.Int())
	}

	if columnField.IsValid() && columnField.Kind() == reflect.Int {
		column = int(columnField.Int())
	}

	// Adjust line number to account for frontmatter position in file
	if line > 0 {
		line += frontmatterStartLine
	}

	// Only return valid positions - avoid returning 1,1 when location is unknown
	if line <= frontmatterStartLine && column <= 1 {
		return 0, 0, message
	}

	return line, column, message
}

// extractFromStringParsing provides fallback string parsing for other YAML libraries
func extractFromStringParsing(errStr string, frontmatterStartLine int) (line int, column int, message string) {
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
									line += frontmatterStartLine
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
					line += frontmatterStartLine
					// Don't default to column 1 when not provided - return 0 instead
					column = 0
					return
				}
			}
		}
	}

	// Parse "yaml: unmarshal errors: line X: message" format (multiline errors)
	if strings.Contains(errStr, "yaml: unmarshal errors:") && strings.Contains(errStr, "line ") {
		lines := strings.Split(errStr, "\n")
		for _, errorLine := range lines {
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
							line += frontmatterStartLine
							column = 0 // Don't default to column 1
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
