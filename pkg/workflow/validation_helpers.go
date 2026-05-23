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
// Parse/coercion helpers live in parse_helpers.go and config_preprocessing.go.

package workflow

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var validationHelpersLog = logger.New("workflow:validation_helpers")

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
		validationHelpersLog.Printf("Invalid mount format: %q (expected 3 colon-separated parts, got %d)", mount, len(parts))
		return "", "", "", errors.New("must follow 'source:destination:mode' format with exactly 3 colon-separated parts")
	}
	mode = parts[2]
	if mode != "ro" && mode != "rw" {
		validationHelpersLog.Printf("Invalid mount mode: %q in %q (must be 'ro' or 'rw')", mode, mount)
		return parts[0], parts[1], parts[2], fmt.Errorf("mode must be 'ro' or 'rw', got %q", mode)
	}
	validationHelpersLog.Printf("Valid mount: source=%s, dest=%s, mode=%s", parts[0], parts[1], mode)
	return parts[0], parts[1], parts[2], nil
}

// mountValidationKind classifies the result of parsing and validating a mount entry.
// Callers use it to translate shared parsing results into context-specific errors
// without re-implementing format, mode, and empty-path branching.
type mountValidationKind int

const (
	// mountValidationOK indicates the mount parsed successfully and both paths are non-empty.
	mountValidationOK mountValidationKind = iota
	// mountValidationFormatError indicates the mount did not have exactly three colon-separated parts.
	mountValidationFormatError
	// mountValidationModeError indicates the mount mode was present but not one of "ro" or "rw".
	mountValidationModeError
	// mountValidationEmptySource indicates the mount source path was empty after successful parsing.
	mountValidationEmptySource
	// mountValidationEmptyDestination indicates the mount destination path was empty after successful parsing.
	mountValidationEmptyDestination
)

// mountParts contains the parsed components of a mount string.
// Fields may still be populated when validation fails after parsing, such as for
// invalid modes or empty source/destination paths.
type mountParts struct {
	source string
	dest   string
	mode   string
}

// parseMountEntry parses a mount string and classifies the validation result.
// It returns the parsed parts together with a mountValidationKind so callers can
// map the shared result to their own error constructors. On format errors the
// returned parts are empty; on mode and empty-path errors any successfully parsed
// fields are preserved in mountParts.
func parseMountEntry(mount string) (mountParts, mountValidationKind) {
	source, dest, mode, err := validateMountStringFormat(mount)
	if err != nil {
		if source == "" && dest == "" && mode == "" {
			return mountParts{}, mountValidationFormatError
		}
		return mountParts{source: source, dest: dest, mode: mode}, mountValidationModeError
	}
	if source == "" {
		return mountParts{source: source, dest: dest, mode: mode}, mountValidationEmptySource
	}
	if dest == "" {
		return mountParts{source: source, dest: dest, mode: mode}, mountValidationEmptyDestination
	}
	return mountParts{source: source, dest: dest, mode: mode}, mountValidationOK
}

// validateStringEnumField checks that a config field, if present, contains one
// of the allowed string values. Non-string values and unrecognised strings are
// removed from the map (treated as absent) and a warning is logged. Use this
// for fields that are pure string enums with no boolean shorthand.
//
// GitHub Actions expression strings (e.g. "${{ inputs.policy }}") are accepted
// without enum validation and passed through unchanged; the resolved value is
// validated at runtime by the safe-output handler.
func validateStringEnumField(configData map[string]any, fieldName string, allowed []string, debugLog *logger.Logger) {
	if configData == nil {
		return
	}
	val, exists := configData[fieldName]
	if !exists || val == nil {
		return
	}
	strVal, ok := val.(string)
	if !ok {
		if debugLog != nil {
			debugLog.Printf("Invalid %s value %v (must be one of %v), ignoring", fieldName, val, allowed)
		}
		delete(configData, fieldName)
		return
	}
	// GitHub Actions expressions are validated at runtime by the handler.
	if containsExpression(strVal) {
		if debugLog != nil {
			debugLog.Printf("%s value is a GitHub Actions expression, skipping compile-time enum validation", fieldName)
		}
		return
	}
	if !slices.Contains(allowed, strVal) {
		if debugLog != nil {
			debugLog.Printf("Invalid %s value %v (must be one of %v), ignoring", fieldName, val, allowed)
		}
		delete(configData, fieldName)
	}
}

// validateNoDuplicateIDs checks that all items have unique IDs extracted by idFunc.
// The onDuplicate callback creates the error to return when a duplicate is found.
func validateNoDuplicateIDs[T any](items []T, idFunc func(T) string, onDuplicate func(string) error) error {
	seen := make(map[string]struct{})
	for _, item := range items {
		id := idFunc(item)
		if _, ok := seen[id]; ok {
			return onDuplicate(id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// containsTrigger reports whether the given 'on:' section value includes
// the named trigger. It handles the three GitHub Actions forms:
//   - string:          "on: <triggerName>"
//   - []any:           "on: [push, <triggerName>]"
//   - map[string]any:  "on:\n  <triggerName>: ..."
func containsTrigger(onSection any, triggerName string) bool {
	found := false
	switch on := onSection.(type) {
	case string:
		found = on == triggerName
	case []any:
		for _, trigger := range on {
			if triggerStr, ok := trigger.(string); ok && triggerStr == triggerName {
				found = true
				break
			}
		}
	case map[string]any:
		_, found = on[triggerName]
	}
	validationHelpersLog.Printf("containsTrigger: trigger=%s, found=%t", triggerName, found)
	return found
}
