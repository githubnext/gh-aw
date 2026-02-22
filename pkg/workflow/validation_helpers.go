// This file provides validation helper functions for agentic workflow compilation.
//
// This file contains reusable validation helpers for common validation patterns
// such as integer range validation, string validation, and list membership checks.
// These utilities are used across multiple workflow configuration validation functions.
//
// # Available Helper Functions
//
//   - validateIntRange() - Validates that an integer value is within a specified range
//   - ValidateRequired() - Validates that a required field is not empty
//   - ValidateMaxLength() - Validates that a field does not exceed maximum length
//   - ValidateMinLength() - Validates that a field meets minimum length requirement
//   - ValidateInList() - Validates that a value is in an allowed list
//   - ValidatePositiveInt() - Validates that a value is a positive integer
//   - ValidateNonNegativeInt() - Validates that a value is a non-negative integer
//   - validateMountStringFormat() - Parses and validates a "source:dest:mode" mount string
//
// # Design Rationale
//
// These helpers consolidate 76+ duplicate validation patterns identified in the
// semantic function clustering analysis. By extracting common patterns, we:
//   - Reduce code duplication across 32 validation files
//   - Provide consistent validation behavior
//   - Make validation code more maintainable and testable
//   - Reduce cognitive overhead when writing new validators
//
// For the validation architecture overview, see validation.go.

package workflow

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var validationHelpersLog = logger.New("workflow:validation_helpers")

// validateIntRange validates that a value is within the specified inclusive range [min, max].
// It returns an error if the value is outside the range, with a descriptive message
// including the field name and the actual value.
//
// Parameters:
//   - value: The integer value to validate
//   - min: The minimum allowed value (inclusive)
//   - max: The maximum allowed value (inclusive)
//   - fieldName: A human-readable name for the field being validated (used in error messages)
//
// Returns:
//   - nil if the value is within range
//   - error with a descriptive message if the value is outside the range
//
// Example:
//
//	err := validateIntRange(port, 1, 65535, "port")
//	if err != nil {
//	    return err
//	}
func validateIntRange(value, min, max int, fieldName string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d, got %d",
			fieldName, min, max, value)
	}
	return nil
}

// ValidateRequired validates that a required field is not empty
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		validationHelpersLog.Printf("Required field validation failed: field=%s", field)
		return NewValidationError(
			field,
			value,
			"field is required and cannot be empty",
			fmt.Sprintf("Provide a non-empty value for '%s'", field),
		)
	}
	return nil
}

// ValidateMaxLength validates that a field does not exceed maximum length
func ValidateMaxLength(field, value string, maxLength int) error {
	if len(value) > maxLength {
		return NewValidationError(
			field,
			value,
			fmt.Sprintf("field exceeds maximum length of %d characters (actual: %d)", maxLength, len(value)),
			fmt.Sprintf("Shorten '%s' to %d characters or less", field, maxLength),
		)
	}
	return nil
}

// ValidateMinLength validates that a field meets minimum length requirement
func ValidateMinLength(field, value string, minLength int) error {
	if len(value) < minLength {
		return NewValidationError(
			field,
			value,
			fmt.Sprintf("field is shorter than minimum length of %d characters (actual: %d)", minLength, len(value)),
			fmt.Sprintf("Ensure '%s' is at least %d characters long", field, minLength),
		)
	}
	return nil
}

// ValidateInList validates that a value is in an allowed list
func ValidateInList(field, value string, allowedValues []string) error {
	if slices.Contains(allowedValues, value) {
		return nil
	}

	validationHelpersLog.Printf("List validation failed: field=%s, value=%s not in allowed list", field, value)
	return NewValidationError(
		field,
		value,
		fmt.Sprintf("value is not in allowed list: %v", allowedValues),
		fmt.Sprintf("Choose one of the allowed values for '%s': %s", field, strings.Join(allowedValues, ", ")),
	)
}

// ValidatePositiveInt validates that a value is a positive integer
func ValidatePositiveInt(field string, value int) error {
	if value <= 0 {
		return NewValidationError(
			field,
			strconv.Itoa(value),
			"value must be a positive integer",
			fmt.Sprintf("Provide a positive integer value for '%s'", field),
		)
	}
	return nil
}

// ValidateNonNegativeInt validates that a value is a non-negative integer
func ValidateNonNegativeInt(field string, value int) error {
	if value < 0 {
		return NewValidationError(
			field,
			strconv.Itoa(value),
			"value must be a non-negative integer",
			fmt.Sprintf("Provide a non-negative integer value for '%s'", field),
		)
	}
	return nil
}

// validateMountStringFormat parses a mount string and validates its basic format.
// Expected format: "source:destination:mode" where mode is "ro" or "rw".
// Returns (source, dest, mode, nil) on success, or ("", "", "", error) on failure.
// The error message describes which aspect of the format is invalid.
// Callers are responsible for wrapping the error with context-appropriate error types.
func validateMountStringFormat(mount string) (source, dest, mode string, err error) {
	parts := strings.Split(mount, ":")
	if len(parts) != 3 {
		return "", "", "", errors.New("must follow 'source:destination:mode' format with exactly 3 colon-separated parts")
	}
	mode = parts[2]
	if mode != "ro" && mode != "rw" {
		return parts[0], parts[1], parts[2], fmt.Errorf("mode must be 'ro' or 'rw', got %q", mode)
	}
	return parts[0], parts[1], parts[2], nil
}
