package parser

import (
	"fmt"
	"strings"
)

// ValidationError represents an input validation error in parser package checks.
type ValidationError struct {
	Field      string
	Value      string
	Reason     string
	Suggestion string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Validation failed for field '%s'", e.Field)

	if e.Value != "" {
		fmt.Fprintf(&b, "\n\nValue: %s", e.Value)
	}

	fmt.Fprintf(&b, "\nReason: %s", e.Reason)

	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\nSuggestion: %s", e.Suggestion)
	}

	return b.String()
}

// NewValidationError creates a new parser validation error with context.
func NewValidationError(field, value, reason, suggestion string) *ValidationError {
	return &ValidationError{
		Field:      field,
		Value:      value,
		Reason:     reason,
		Suggestion: suggestion,
	}
}
