package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var importErrorLog = logger.New("parser:import_error")

// ImportError represents an error that occurred during import resolution
type ImportError struct {
	ImportPath string // The import path that failed (e.g., "nonexistent.md")
	FilePath   string // The workflow file containing the import
	Line       int    // Line number where the import is defined
	Column     int    // Column number where the import is defined
	Cause      error  // The underlying error
}

// ImportCycleError represents a circular import dependency
type ImportCycleError struct {
	Chain        []string // Full import chain showing the cycle (e.g., ["a.md", "b.md", "c.md", "d.md", "b.md"])
	WorkflowFile string   // The main workflow file being compiled
}

// Error returns the error message for ImportCycleError
func (e *ImportCycleError) Error() string {
	if len(e.Chain) == 0 {
		return "circular import detected"
	}
	return "circular import detected: " + strings.Join(e.Chain, " → ")
}

// FormatImportCycleError formats an import cycle error with a delightful multiline indented display
func FormatImportCycleError(err *ImportCycleError) error {
	importErrorLog.Printf("Formatting import cycle error: chain=%v, workflow=%s", err.Chain, err.WorkflowFile)

	if len(err.Chain) < 2 {
		return errors.New("circular import detected (invalid chain)")
	}

	// Build a multiline, indented representation of the import chain
	var messageBuilder strings.Builder
	messageBuilder.WriteString("Import cycle detected\n\n")
	messageBuilder.WriteString("The following import chain creates a circular dependency:\n\n")

	// Show each step in the chain with indentation to emphasize the flow
	for i, file := range err.Chain {
		indent := strings.Repeat("  ", i)
		if i == 0 {
			messageBuilder.WriteString(fmt.Sprintf("%s%s (starting point)\n", indent, file))
		} else if i == len(err.Chain)-1 {
			// Last item is the back-edge - highlight it
			messageBuilder.WriteString(fmt.Sprintf("%s↳ %s ⚠️  cycles back to %s\n", indent, file, err.Chain[0]))
		} else {
			messageBuilder.WriteString(fmt.Sprintf("%s↳ imports %s\n", indent, file))
		}
	}

	messageBuilder.WriteString("\nTo fix this issue:\n")
	messageBuilder.WriteString("1. Review the import dependencies in the files listed above\n")
	messageBuilder.WriteString("2. Remove one of the imports to break the cycle\n")
	messageBuilder.WriteString("3. Consider restructuring your workflow imports to avoid circular dependencies\n")

	return &FormattedParserError{formatted: messageBuilder.String(), cause: err}
}

// FormattedParserError is a sentinel error type returned by FormatImportError (and similar
// parser-level formatters) to signal that the error message is already console-formatted
// with source location.  Callers that detect this type must NOT re-wrap it, otherwise the
// user would see a double-formatted error message (e.g. one location at "engine:" wrapping
// another at the actual import line).
//
// This type is intentionally exported so that the workflow package's isFormattedCompilerError
// helper can detect it via errors.As without creating a circular import.
type FormattedParserError struct {
	formatted string // The complete console-formatted error string ready for display.
	cause     error  // The underlying error (e.g. ImportError.Cause) for errors.Is/As traversal.
}

func (e *FormattedParserError) Error() string { return e.formatted }
func (e *FormattedParserError) Unwrap() error { return e.cause }

// NewFormattedParserError creates a FormattedParserError with the given pre-formatted
// message string. Use this in external packages (e.g. pkg/workflow) to return an error
// that isFormattedCompilerError can detect without double-wrapping.
func NewFormattedParserError(formatted string) *FormattedParserError {
	return &FormattedParserError{formatted: formatted}
}

// FormatImportError formats an import error as a compilation error with source location
func FormatImportError(err *ImportError, yamlContent string) error {
	importErrorLog.Printf("Formatting import error: path=%s, file=%s, line=%d", err.ImportPath, err.FilePath, err.Line)

	lines := strings.Split(yamlContent, "\n")

	// Create context lines around the error
	var context []string
	startLine := max(1, err.Line-2)
	endLine := min(len(lines), err.Line+2)

	for i := startLine; i <= endLine; i++ {
		if i-1 < len(lines) {
			context = append(context, lines[i-1])
		}
	}

	// Determine the error message based on the cause
	message := "failed to resolve import"
	if err.Cause != nil {
		causeMsg := err.Cause.Error()
		if strings.Contains(causeMsg, "file not found") {
			message = "import file not found"
		} else if strings.Contains(causeMsg, "failed to download") {
			message = "failed to download import file"
		} else if strings.Contains(causeMsg, "failed to resolve ref") {
			message = "failed to resolve import reference"
		} else if strings.Contains(causeMsg, "invalid workflowspec") {
			message = "invalid import specification"
		} else {
			message = causeMsg
		}
	}

	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{
			File:   err.FilePath,
			Line:   err.Line,
			Column: err.Column,
		},
		Type:    "error",
		Message: message,
		Context: context,
	}

	formattedErr := console.FormatError(compilerErr)
	// Return a FormattedParserError so callers can detect that this error is already
	// console-formatted and must not be re-wrapped with additional location context.
	return &FormattedParserError{formatted: formattedErr, cause: err.Cause}
}

// findImportsFieldLocation finds the line and column number of the imports field in YAML content
func findImportsFieldLocation(yamlContent string) (line int, column int) {
	lines := strings.Split(yamlContent, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for "imports:" at the start of a line (accounting for indentation)
		if strings.HasPrefix(trimmed, "imports:") {
			// Find the column where "imports:" starts
			col := strings.Index(line, "imports:") + 1 // +1 for 1-based indexing
			importErrorLog.Printf("Found imports field at line=%d, col=%d", i+1, col)
			return i + 1, col // +1 for 1-based line indexing
		}
	}
	// Default to line 1, column 1 if not found
	importErrorLog.Print("imports field not found in YAML content, defaulting to line=1, col=1")
	return 1, 1
}

// findImportItemLocation finds the line and column number of a specific import item in YAML content
func findImportItemLocation(yamlContent string, importPath string) (line int, column int) {
	importErrorLog.Printf("Locating import item in YAML: path=%s", importPath)
	lines := strings.Split(yamlContent, "\n")
	inImportsSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're entering the imports section
		if strings.HasPrefix(trimmed, "imports:") {
			inImportsSection = true
			continue
		}

		// If we're in the imports section and find a line with our import path
		if inImportsSection {
			// Check if this line exits the imports section (new top-level key)
			if len(line) > 0 && line[0] != ' ' && line[0] != '-' && line[0] != '\t' {
				break
			}

			// Check for the import path in this line
			if strings.Contains(line, importPath) {
				// Find the column where the import path starts
				col := strings.Index(line, importPath) + 1 // +1 for 1-based indexing
				importErrorLog.Printf("Located import item at line=%d, col=%d", i+1, col)
				return i + 1, col // +1 for 1-based line indexing
			}
		}
	}

	// Fallback to imports field location
	importErrorLog.Printf("Import item %q not found, falling back to imports field location", importPath)
	return findImportsFieldLocation(yamlContent)
}
