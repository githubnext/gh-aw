// This file provides validation helper functions for agentic workflow compilation.
//
// This file contains reusable validation helpers for common validation patterns
// such as integer range validation, string validation, and list membership checks.
// These utilities are used across multiple workflow configuration validation functions.
//
// # Available Helper Functions
//
//   - newValidationLogger() - Creates a standardized logger for a validation domain
//   - validateIntRange() - Validates that an integer value is within a specified range
//   - validateMountStringFormat() - Parses and validates a "source:dest:mode" mount string
//   - containsTrigger() - Reports whether an 'on:' section includes a named trigger
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
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

// newValidationLogger creates a standardized logger for a validation domain.
// It follows the naming convention "workflow:<domain>_validation" used across
// all *_validation.go files.
//
// Example:
//
//	var engineValidationLog = newValidationLogger("engine")
//	// produces logger named "workflow:engine_validation"
func newValidationLogger(domain string) *logger.Logger {
	return logger.New("workflow:" + domain + "_validation")
}

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

// validateStringEnumField checks that a config field, if present, contains one
// of the allowed string values. Non-string values and unrecognised strings are
// removed from the map (treated as absent) and a warning is logged. Use this
// for fields that are pure string enums with no boolean shorthand.
func validateStringEnumField(configData map[string]any, fieldName string, allowed []string, log *logger.Logger) {
	if configData == nil {
		return
	}
	val, exists := configData[fieldName]
	if !exists || val == nil {
		return
	}
	strVal, ok := val.(string)
	if !ok || !slices.Contains(allowed, strVal) {
		if log != nil {
			log.Printf("Invalid %s value %v (must be one of %v), ignoring", fieldName, val, allowed)
		}
		delete(configData, fieldName)
	}
}

// validateNoDuplicateIDs checks that all items have unique IDs extracted by idFunc.
// The onDuplicate callback creates the error to return when a duplicate is found.
func validateNoDuplicateIDs[T any](items []T, idFunc func(T) string, onDuplicate func(string) error) error {
	seen := make(map[string]bool)
	for _, item := range items {
		id := idFunc(item)
		if seen[id] {
			return onDuplicate(id)
		}
		seen[id] = true
	}
	return nil
}

// containsTrigger reports whether the given 'on:' section value includes
// the named trigger. It handles the three GitHub Actions forms:
//   - string:          "on: <triggerName>"
//   - []any:           "on: [push, <triggerName>]"
//   - map[string]any:  "on:\n  <triggerName>: ..."
func containsTrigger(onSection any, triggerName string) bool {
	switch on := onSection.(type) {
	case string:
		return on == triggerName
	case []any:
		for _, trigger := range on {
			if triggerStr, ok := trigger.(string); ok && triggerStr == triggerName {
				return true
			}
		}
	case map[string]any:
		_, ok := on[triggerName]
		return ok
	}
	return false
}
