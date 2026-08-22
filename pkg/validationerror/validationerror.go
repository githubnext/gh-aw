// Package validationerror provides a shared, dependency-neutral payload and
// message formatting for structured field/value/reason/suggestion validation
// errors. Packages such as parser and workflow embed Payload in their own
// error types so that:
//   - the message formatting logic (including value truncation) lives in one
//     place and future fixes don't need to be duplicated per package, and
//   - callers can uniformly detect any structured validation error with
//     errors.As(err, &validationerror.ValidationError) regardless of which
//     package produced it.
package validationerror

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

// MaxValueLength is the maximum number of characters kept for the Value field
// when formatting a message with value truncation enabled.
const MaxValueLength = 100

// Payload holds the common field/value/reason/suggestion data shared by
// structured validation errors across packages.
type Payload struct {
	Field      string
	Value      string
	Reason     string
	Suggestion string
}

// ValidationField returns the name of the field that failed validation.
func (p Payload) ValidationField() string { return p.Field }

// ValidationValue returns the offending value, if any.
func (p Payload) ValidationValue() string { return p.Value }

// ValidationReason returns the reason validation failed.
func (p Payload) ValidationReason() string { return p.Reason }

// ValidationSuggestion returns actionable remediation guidance, if any.
func (p Payload) ValidationSuggestion() string { return p.Suggestion }

// ValidationError is the common interface implemented by structured
// field/value/reason/suggestion validation errors across packages (for
// example parser.ValidationError and workflow.WorkflowValidationError).
// Use errors.As(err, &target) with a variable of this interface type to
// uniformly extract validation details regardless of the concrete type.
type ValidationError interface {
	error
	ValidationField() string
	ValidationValue() string
	ValidationReason() string
	ValidationSuggestion() string
}

// Format renders header followed by the payload's value/reason/suggestion in
// a deterministic multi-line shape. When truncateValue is true, Value is
// truncated to MaxValueLength before being included.
func Format(header string, p Payload, truncateValue bool) string {
	var b strings.Builder
	b.WriteString(header)

	if p.Value != "" {
		value := p.Value
		if truncateValue {
			value = stringutil.Truncate(value, MaxValueLength)
		}
		fmt.Fprintf(&b, "\n\nValue: %s", value)
	}

	fmt.Fprintf(&b, "\nReason: %s", p.Reason)

	if p.Suggestion != "" {
		fmt.Fprintf(&b, "\nSuggestion: %s", p.Suggestion)
	}

	return b.String()
}
