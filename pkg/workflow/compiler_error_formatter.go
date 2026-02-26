package workflow

import (
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerErrorLog = logger.New("workflow:compiler_error_formatter")

// formatCompilerError creates a formatted compiler error message with optional error wrapping
// filePath: the file path to include in the error (typically markdownPath or lockFile)
// errType: the error type ("error" or "warning")
// message: the error message text
// cause: optional underlying error to wrap (use nil for validation errors)
func formatCompilerError(filePath string, errType string, message string, cause error) error {
	compilerErrorLog.Printf("Formatting compiler error: file=%s, type=%s, message=%s", filePath, errType, message)
	formattedErr := console.FormatError(console.CompilerError{
		Position: console.ErrorPosition{
			File:   filePath,
			Line:   1,
			Column: 1,
		},
		Type:    errType,
		Message: message,
	})

	// Wrap the underlying error if provided (preserves error chain)
	if cause != nil {
		return fmt.Errorf("%s: %w", formattedErr, cause)
	}

	// Create new error for validation errors (no underlying cause)
	return errors.New(formattedErr)
}

// formatCompilerErrorWithPosition creates a formatted compiler error with specific line/column position
// filePath: the file path to include in the error
// line: the line number where the error occurred
// column: the column number where the error occurred
// errType: the error type ("error" or "warning")
// message: the error message text
// cause: optional underlying error to wrap (use nil for validation errors)
func formatCompilerErrorWithPosition(filePath string, line int, column int, errType string, message string, cause error) error {
	compilerErrorLog.Printf("Formatting compiler error: file=%s, line=%d, column=%d, type=%s, message=%s", filePath, line, column, errType, message)
	formattedErr := console.FormatError(console.CompilerError{
		Position: console.ErrorPosition{
			File:   filePath,
			Line:   line,
			Column: column,
		},
		Type:    errType,
		Message: message,
	})

	// Wrap the underlying error if provided (preserves error chain)
	if cause != nil {
		return fmt.Errorf("%s: %w", formattedErr, cause)
	}

	// Create new error for validation errors (no underlying cause)
	return errors.New(formattedErr)
}
