package workflow

import (
	"errors"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
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

type compilerErrorOpts struct {
	FilePath string
	Line     int
	Column   int
	ErrType  string
	Message  string
	Cause    error
	Context  []string
}

// formatCompilerError creates a formatted compiler error from options.
// Defaults line/column to 1:1 when they are not provided.
// When Cause is a *WorkflowValidationError with Line > 0, the error's own
// position (and file, when available) takes precedence.
func formatCompilerError(opts compilerErrorOpts) error {
	line := opts.Line
	column := opts.Column
	if line <= 0 {
		line = 1
	}
	if column <= 0 {
		column = 1
	}

	// Promote precise source location from WorkflowValidationError when available so that
	// the emitted "file:line:col: error:" prefix points directly at the problematic field
	// rather than always defaulting to line 1, column 1.
	var vErr *WorkflowValidationError
	if errors.As(opts.Cause, &vErr) && vErr.Line > 0 {
		line = vErr.Line
		if vErr.Column > 0 {
			column = vErr.Column
		}
		if vErr.File != "" {
			opts.FilePath = vErr.File
		}
	}
	compilerErrorLog.Printf("Formatting compiler error: file=%s, line=%d, column=%d, type=%s, context=%d lines", opts.FilePath, line, column, opts.ErrType, len(opts.Context))
	formattedErr := console.FormatError(console.CompilerError{
		Position: console.ErrorPosition{
			File:   opts.FilePath,
			Line:   line,
			Column: column,
		},
		Type:    opts.ErrType,
		Message: opts.Message,
		Context: opts.Context,
	})

	return &wrappedCompilerError{formatted: formattedErr, cause: opts.Cause}
}

// isFormattedCompilerError reports whether err is already a console-formatted compiler error
// produced by formatCompilerError or parser.FormatImportError.
// Use this instead of fragile string-contains checks to avoid double-wrapping.
func isFormattedCompilerError(err error) bool {
	var wce *wrappedCompilerError
	if errors.As(err, &wce) {
		return true
	}
	// Also detect errors from the parser package (e.g. FormatImportError) which are already
	// console-formatted with source location and must not be re-wrapped.
	var fpe *parser.FormattedParserError
	return errors.As(err, &fpe)
}

// formatCompilerMessage creates a formatted compiler message string (for warnings printed to stderr)
// filePath: the file path to include in the message (typically markdownPath or lockFile)
// msgType: the message type ("error" or "warning")
// message: the message text
func formatCompilerMessage(filePath string, msgType string, message string) string {
	return console.FormatError(console.CompilerError{
		Position: console.ErrorPosition{
			File:   filePath,
			Line:   0,
			Column: 0,
		},
		Type:    msgType,
		Message: message,
	})
}
