package parser

import (
	"fmt"

	"github.com/github/gh-aw/pkg/validationerror"
)

// ValidationError represents an input validation error in parser package checks.
// It embeds validationerror.Payload so callers can uniformly detect structured
// validation errors from any package via errors.As(err, &validationerror.ValidationError).
type ValidationError struct {
	validationerror.Payload
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	header := fmt.Sprintf("Validation failed for field '%s'", e.Field)
	return validationerror.Format(header, e.Payload, false)
}

// NewValidationError creates a new parser validation error with context.
func NewValidationError(field, value, reason, suggestion string) *ValidationError {
	return &ValidationError{
		Payload: validationerror.Payload{
			Field:      field,
			Value:      value,
			Reason:     reason,
			Suggestion: suggestion,
		},
	}
}
