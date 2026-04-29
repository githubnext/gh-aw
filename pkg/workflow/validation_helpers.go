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
// # Type Conversion Helpers (any → []string)
//
//   - parseStringSliceAny() - Canonical coercion of []string/[]any to []string; skips non-string items.
//     For GitHub Actions fields where a bare string is valid shorthand for a single-element list
//     (e.g. `needs: job-name`, `state: failure`), handle the string case explicitly at the call site.
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

// preprocessProtectedFilesField preprocesses the "protected-files" field in configData,
// handling both the legacy string-enum form and the new object form.
//
// String form (unchanged): "blocked", "allowed", or "fallback-to-issue".
// Object form: { policy: "blocked", exclude: ["AGENTS.md"] }
//   - policy is optional; when missing or empty, this preprocessing step treats it as absent
//     and leaves downstream default handling to apply (the "protected-files" key is deleted)
//   - exclude is a list of filenames/path-prefixes to remove from the default protected set
//
// When the object form is encountered the field is normalised in-place:
//   - "protected-files" is replaced with the extracted policy string, or deleted when policy is absent/empty
//   - The extracted exclude slice is returned so callers can store it in the config struct
//
// When the string form is encountered the field is left unchanged and nil is returned.
// The log parameter is optional; pass nil to suppress debug output.
func preprocessProtectedFilesField(configData map[string]any, log *logger.Logger) []string {
	if configData == nil {
		return nil
	}
	raw, exists := configData["protected-files"]
	if !exists || raw == nil {
		return nil
	}
	pfMap, ok := raw.(map[string]any)
	if !ok {
		// String form — left for validateStringEnumField to handle
		return nil
	}
	// Object form: extract policy and exclude
	if policy, ok := pfMap["policy"].(string); ok && policy != "" {
		configData["protected-files"] = policy
		if log != nil {
			log.Printf("protected-files object form: policy=%s", policy)
		}
	} else {
		delete(configData, "protected-files")
		if log != nil {
			log.Print("protected-files object form: no policy, using default")
		}
	}
	return parseStringSliceAny(pfMap["exclude"], log)
}

// parseStringSliceAny coerces a raw any value into a []string.
// It accepts a []string (returned as-is), []any (string elements extracted),
// or nil (returns nil). Non-string elements inside a []any are skipped.
// The log parameter is optional; pass nil to suppress debug output about skipped items.
//
// Bare string scalars are intentionally NOT wrapped — this preserves the existing
// contract for callers (e.g. ParseStringArrayFromConfig) that treat a scalar string
// as a type error rather than a single-element list.
//
// When GitHub Actions syntax allows a scalar as shorthand for a single-element list
// (e.g. `needs: "job-name"`, `state: "failure"`), handle the string case explicitly
// before calling this function:
//
//	if s, ok := raw.(string); ok { return []string{s} }
//	return parseStringSliceAny(raw, log)
func parseStringSliceAny(raw any, log *logger.Logger) []string {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		// Already the right type — return directly without copying.
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else if log != nil {
				log.Printf("parseStringSliceAny: skipping non-string item: %T", item)
			}
		}
		return result
	default:
		if log != nil {
			log.Printf("parseStringSliceAny: unexpected type %T, ignoring", raw)
		}
		return nil
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
