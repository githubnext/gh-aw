// This file provides validation helper functions for agentic workflow compilation.
//
// This file contains reusable validation helpers for common validation patterns
// such as integer range validation, string validation, and list membership checks.
// These utilities are used across multiple workflow configuration validation functions.
//
// # Available Helper Functions
//
//   - validateIntRange() - Validates that an integer value is within a specified range
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

// formatList formats a list of strings as a comma-separated list with natural language conjunction
func formatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}
	return fmt.Sprintf("%s, and %s", formatList(items[:len(items)-1]), items[len(items)-1])
}

// validateTargetRepoSlug validates that a target-repo slug is not a wildcard.
// Returns true if the value is invalid (i.e., equals "*").
// This helper is used when the target-repo has already been parsed into a struct field.
func validateTargetRepoSlug(targetRepoSlug string, log *logger.Logger) bool {
	if targetRepoSlug == "*" {
		if log != nil {
			log.Print("Invalid target-repo: wildcard '*' is not allowed")
		}
		return true // Return true to indicate validation error
	}
	return false
}
