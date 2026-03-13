package workflow

import (
	"errors"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerErrorLog = logger.New("workflow:compiler_error_formatter")

// wrappedCompilerError carries the formatted diagnostic string (returned by Error())
// and the original underlying error (returned by Unwrap()), preserving the error chain
// for errors.Is/As callers while keeping the displayed string free of duplication.
type wrappedCompilerError struct {
	formatted string
	cause     error
}

func (e *wrappedCompilerError) Error() string { return e.formatted }
func (e *wrappedCompilerError) Unwrap() error { return e.cause }

// formatCompilerError creates a formatted compiler error message with optional error wrapping.
// It always uses line:1, column:1 so IDE tooling can navigate to the file even when a
// specific source position is unavailable.
//
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

	// Preserve the error chain via Unwrap() so errors.Is/As continue to work on the
	// returned error, while Error() returns only the clean formatted string (no duplication).
	if cause != nil {
		return &wrappedCompilerError{formatted: formattedErr, cause: cause}
	}

	// Create new error for validation errors (no underlying cause)
	return errors.New(formattedErr)
}

// formatCompilerErrorWithPosition creates a formatted compiler error with specific line/column position.
//
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

	// Preserve the error chain via Unwrap() so errors.Is/As continue to work on the
	// returned error, while Error() returns only the clean formatted string (no duplication).
	if cause != nil {
		return &wrappedCompilerError{formatted: formattedErr, cause: cause}
	}

	// Create new error for validation errors (no underlying cause)
	return errors.New(formattedErr)
}
