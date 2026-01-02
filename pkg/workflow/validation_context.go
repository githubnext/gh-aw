// Package workflow provides unified validation context for error aggregation.
//
// # ValidationContext
//
// ValidationContext collects validation errors and warnings across the entire
// validation pipeline, enabling developers to see all validation issues in one
// compilation run instead of fixing one issue at a time.
//
// # Validation Phases
//
// The validation process is organized into four distinct phases:
//   - ParseTime: Validation during markdown parsing (frontmatter, syntax)
//   - PreCompile: Validation before YAML generation (configuration, features)
//   - PostYAMLGeneration: Validation after YAML is generated (expression sizes, schema)
//   - PreEmit: Final validation before writing lock file (file size, completeness)
//
// # Usage Pattern
//
// Create a ValidationContext at the start of compilation, pass it to all
// validators, and check for errors before proceeding to the next phase:
//
//	ctx := NewValidationContext(markdownPath, workflowData)
//	ctx.SetPhase(PhasePreCompile)
//
//	// Run all pre-compile validators
//	validateFeatures(ctx, workflowData)
//	validateSandboxConfig(ctx, workflowData)
//	validateStrictMode(ctx, workflowData)
//
//	// Check for errors before continuing
//	if ctx.HasErrors() {
//		return ctx.Error()
//	}
//
// # Error Reporting
//
// ValidationContext uses console formatting for multi-error reports that are
// IDE-parseable and human-readable:
//
//	file.md:1:1: error: feature flag validation failed
//	  Invalid feature: unknown-feature
//	file.md:1:1: error: sandbox configuration invalid
//	  Mount path must be absolute: ./relative/path
//
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var validationContextLog = logger.New("workflow:validation_context")

// ValidationPhase represents a distinct phase in the validation pipeline
type ValidationPhase int

const (
	// PhaseParseTime validates during markdown parsing (frontmatter, syntax)
	PhaseParseTime ValidationPhase = iota
	// PhasePreCompile validates before YAML generation (configuration, features)
	PhasePreCompile
	// PhasePostYAMLGeneration validates after YAML is generated (expression sizes, schema)
	PhasePostYAMLGeneration
	// PreEmit validates before writing lock file (file size, completeness)
	PhasePreEmit
)

// String returns the human-readable name of the validation phase
func (p ValidationPhase) String() string {
	switch p {
	case PhaseParseTime:
		return "ParseTime"
	case PhasePreCompile:
		return "PreCompile"
	case PhasePostYAMLGeneration:
		return "PostYAMLGeneration"
	case PhasePreEmit:
		return "PreEmit"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}

// ValidationError represents a single validation error with metadata
type ValidationError struct {
	Validator string // Name of the validator that produced this error
	Message   string // Error message
	File      string // Source file path
	Line      int    // Line number (0 if unknown)
	Column    int    // Column number (0 if unknown)
}

// ValidationWarning represents a single validation warning with metadata
type ValidationWarning struct {
	Validator string // Name of the validator that produced this warning
	Message   string // Warning message
	File      string // Source file path
	Line      int    // Line number (0 if unknown)
	Column    int    // Column number (0 if unknown)
}

// ValidationContext collects validation errors and warnings across the validation pipeline
type ValidationContext struct {
	phase           ValidationPhase
	errors          []ValidationError
	warnings        []ValidationWarning
	markdownPath    string
	workflowData    *WorkflowData
	yamlContent     string
	verbose         bool
	skipExternalAPI bool
}

// NewValidationContext creates a new validation context for a workflow
func NewValidationContext(markdownPath string, workflowData *WorkflowData) *ValidationContext {
	validationContextLog.Printf("Creating validation context: path=%s", markdownPath)
	return &ValidationContext{
		phase:        PhaseParseTime,
		errors:       make([]ValidationError, 0),
		warnings:     make([]ValidationWarning, 0),
		markdownPath: markdownPath,
		workflowData: workflowData,
		verbose:      false,
	}
}

// SetPhase sets the current validation phase
func (vc *ValidationContext) SetPhase(phase ValidationPhase) {
	validationContextLog.Printf("Setting validation phase: %s -> %s", vc.phase, phase)
	vc.phase = phase
}

// GetPhase returns the current validation phase
func (vc *ValidationContext) GetPhase() ValidationPhase {
	return vc.phase
}

// SetYAMLContent sets the generated YAML content for post-generation validation
func (vc *ValidationContext) SetYAMLContent(yamlContent string) {
	validationContextLog.Printf("Setting YAML content: length=%d", len(yamlContent))
	vc.yamlContent = yamlContent
}

// GetYAMLContent returns the generated YAML content
func (vc *ValidationContext) GetYAMLContent() string {
	return vc.yamlContent
}

// SetVerbose enables verbose validation logging
func (vc *ValidationContext) SetVerbose(verbose bool) {
	vc.verbose = verbose
}

// IsVerbose returns whether verbose logging is enabled
func (vc *ValidationContext) IsVerbose() bool {
	return vc.verbose
}

// SetSkipExternalAPI controls whether external API validation is skipped
func (vc *ValidationContext) SetSkipExternalAPI(skip bool) {
	vc.skipExternalAPI = skip
}

// ShouldSkipExternalAPI returns whether external API validation should be skipped
func (vc *ValidationContext) ShouldSkipExternalAPI() bool {
	return vc.skipExternalAPI
}

// GetMarkdownPath returns the markdown file path being validated
func (vc *ValidationContext) GetMarkdownPath() string {
	return vc.markdownPath
}

// GetWorkflowData returns the workflow data being validated
func (vc *ValidationContext) GetWorkflowData() *WorkflowData {
	return vc.workflowData
}

// AddError adds a validation error to the context
func (vc *ValidationContext) AddError(validator string, err error) {
	if err == nil {
		return
	}

	validationContextLog.Printf("Adding error: validator=%s, phase=%s, message=%s", validator, vc.phase, err.Error())
	vc.errors = append(vc.errors, ValidationError{
		Validator: validator,
		Message:   err.Error(),
		File:      vc.markdownPath,
		Line:      1, // Default to line 1 for now, can be enhanced later
		Column:    1,
	})
}

// AddErrorWithPosition adds a validation error with specific position information
func (vc *ValidationContext) AddErrorWithPosition(validator string, message string, line, column int) {
	validationContextLog.Printf("Adding error with position: validator=%s, phase=%s, line=%d, col=%d", validator, vc.phase, line, column)
	vc.errors = append(vc.errors, ValidationError{
		Validator: validator,
		Message:   message,
		File:      vc.markdownPath,
		Line:      line,
		Column:    column,
	})
}

// AddWarning adds a validation warning to the context
func (vc *ValidationContext) AddWarning(validator string, message string) {
	validationContextLog.Printf("Adding warning: validator=%s, phase=%s, message=%s", validator, vc.phase, message)
	vc.warnings = append(vc.warnings, ValidationWarning{
		Validator: validator,
		Message:   message,
		File:      vc.markdownPath,
		Line:      1, // Default to line 1 for now
		Column:    1,
	})
}

// AddWarningWithPosition adds a validation warning with specific position information
func (vc *ValidationContext) AddWarningWithPosition(validator string, message string, line, column int) {
	validationContextLog.Printf("Adding warning with position: validator=%s, phase=%s, line=%d, col=%d", validator, vc.phase, line, column)
	vc.warnings = append(vc.warnings, ValidationWarning{
		Validator: validator,
		Message:   message,
		File:      vc.markdownPath,
		Line:      line,
		Column:    column,
	})
}

// HasErrors returns true if any validation errors have been collected
func (vc *ValidationContext) HasErrors() bool {
	return len(vc.errors) > 0
}

// HasWarnings returns true if any validation warnings have been collected
func (vc *ValidationContext) HasWarnings() bool {
	return len(vc.warnings) > 0
}

// ErrorCount returns the number of validation errors
func (vc *ValidationContext) ErrorCount() int {
	return len(vc.errors)
}

// WarningCount returns the number of validation warnings
func (vc *ValidationContext) WarningCount() int {
	return len(vc.warnings)
}

// GetErrors returns all collected validation errors
func (vc *ValidationContext) GetErrors() []ValidationError {
	return vc.errors
}

// GetWarnings returns all collected validation warnings
func (vc *ValidationContext) GetWarnings() []ValidationWarning {
	return vc.warnings
}

// Error returns a formatted error message containing all validation errors
// This method satisfies the error interface
func (vc *ValidationContext) Error() string {
	if !vc.HasErrors() {
		return ""
	}
	return vc.FormatReport()
}

// FormatReport generates a multi-error report with console formatting
func (vc *ValidationContext) FormatReport() string {
	var output strings.Builder

	// Format errors
	if len(vc.errors) > 0 {
		if len(vc.errors) == 1 {
			// Single error - use simple format
			err := vc.errors[0]
			formatted := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   err.File,
					Line:   err.Line,
					Column: err.Column,
				},
				Type:    "error",
				Message: err.Message,
			})
			output.WriteString(formatted)
		} else {
			// Multiple errors - format each one
			output.WriteString(fmt.Sprintf("Found %d validation errors:\n\n", len(vc.errors)))
			for i, err := range vc.errors {
				formatted := console.FormatError(console.CompilerError{
					Position: console.ErrorPosition{
						File:   err.File,
						Line:   err.Line,
						Column: err.Column,
					},
					Type:    "error",
					Message: err.Message,
				})
				output.WriteString(formatted)
				if i < len(vc.errors)-1 {
					output.WriteString("\n")
				}
			}
		}
	}

	// Format warnings (if verbose or if there are no errors)
	if (vc.verbose || !vc.HasErrors()) && len(vc.warnings) > 0 {
		if len(vc.errors) > 0 {
			output.WriteString("\n\n")
		}
		output.WriteString(fmt.Sprintf("Found %d validation warnings:\n\n", len(vc.warnings)))
		for i, warn := range vc.warnings {
			formatted := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   warn.File,
					Line:   warn.Line,
					Column: warn.Column,
				},
				Type:    "warning",
				Message: warn.Message,
			})
			output.WriteString(formatted)
			if i < len(vc.warnings)-1 {
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

// Clear clears all collected errors and warnings
func (vc *ValidationContext) Clear() {
	validationContextLog.Printf("Clearing validation context")
	vc.errors = make([]ValidationError, 0)
	vc.warnings = make([]ValidationWarning, 0)
}

// Summary returns a summary of validation results
func (vc *ValidationContext) Summary() string {
	if !vc.HasErrors() && !vc.HasWarnings() {
		return "No validation issues found"
	}

	var parts []string
	if vc.HasErrors() {
		parts = append(parts, fmt.Sprintf("%d error(s)", len(vc.errors)))
	}
	if vc.HasWarnings() {
		parts = append(parts, fmt.Sprintf("%d warning(s)", len(vc.warnings)))
	}

	return strings.Join(parts, ", ")
}
