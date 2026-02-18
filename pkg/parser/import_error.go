package parser

import (
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

// Error returns the error message
func (e *ImportError) Error() string {
	return fmt.Sprintf("failed to resolve import '%s': %v", e.ImportPath, e.Cause)
}

// Unwrap returns the underlying error
func (e *ImportError) Unwrap() error {
	return e.Cause
}

// Error returns the error message for ImportCycleError
func (e *ImportCycleError) Error() string {
	if len(e.Chain) == 0 {
		return "circular import detected"
	}
	return fmt.Sprintf("circular import detected: %s", strings.Join(e.Chain, " → "))
}

// FormatImportCycleError formats an import cycle error with a delightful multiline indented display
func FormatImportCycleError(err *ImportCycleError) error {
	importErrorLog.Printf("Formatting import cycle error: chain=%v, workflow=%s", err.Chain, err.WorkflowFile)

	if len(err.Chain) < 2 {
		return fmt.Errorf("circular import detected (invalid chain)")
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

	return fmt.Errorf("%s", messageBuilder.String())
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
	return fmt.Errorf("%s", formattedErr)
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
			return i + 1, col                          // +1 for 1-based line indexing
		}
	}
	// Default to line 1, column 1 if not found
	return 1, 1
}

// findImportItemLocation finds the line and column number of a specific import item in YAML content
func findImportItemLocation(yamlContent string, importPath string) (line int, column int) {
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
				return i + 1, col                          // +1 for 1-based line indexing
			}
		}
	}

	// Fallback to imports field location
	return findImportsFieldLocation(yamlContent)
}
