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

// formatCompilerError creates a formatted compiler error at line 1, column 1.
// Use this when the exact source position is unknown; IDE tooling can still navigate to the file.
// Use formatCompilerErrorWithPosition when a specific line/column is available.
//
// filePath: the file path to include in the error (typically markdownPath or lockFile)
// errType: the error type ("error" or "warning")
// message: the error message text
// cause: optional underlying error to wrap (use nil for validation errors)
func formatCompilerError(filePath string, errType string, message string, cause error) error {
	return formatCompilerErrorWithPosition(filePath, 1, 1, errType, message, cause)
}

// isFormattedCompilerError reports whether err is already a console-formatted compiler error
// produced by formatCompilerError or formatCompilerErrorWithPosition.  Use this instead of
// fragile string-contains checks to avoid double-wrapping.
func isFormattedCompilerError(err error) bool {
	var wce *wrappedCompilerError
	return errors.As(err, &wce)
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

	// Always return a *wrappedCompilerError so isFormattedCompilerError can detect it.
	// cause may be nil for validation errors that have no underlying cause.
	return &wrappedCompilerError{formatted: formattedErr, cause: cause}
}
