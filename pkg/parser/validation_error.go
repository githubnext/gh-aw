package parser

import (
	"fmt"
	"strings"
	"time"
)

// ValidationError represents an input validation error in parser package checks.
type ValidationError struct {
	Field      string
	Value      string
	Reason     string
	Suggestion string
	Timestamp  time.Time
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] Validation failed for field '%s'",
		e.Timestamp.Format(time.RFC3339), e.Field)

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
		Timestamp:  time.Now(),
	}
}
