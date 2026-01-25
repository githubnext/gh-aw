// Package workflow provides validation helper functions for agentic workflow compilation.
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
//   - isEmptyOrNil() - Checks if a value is empty, nil, or zero
//   - getMapFieldAsString() - Safely extracts a string field from a map[string]any
//   - getMapFieldAsMap() - Safely extracts a nested map from a map[string]any
//   - getMapFieldAsBool() - Safely extracts a boolean field from a map[string]any
//   - getMapFieldAsInt() - Safely extracts an integer field from a map[string]any
//   - fileExists() - Checks if a file exists at the given path
//   - dirExists() - Checks if a directory exists at the given path
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
	"fmt"
	"os"
	"strings"
)

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
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

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
			fmt.Sprintf("%d", value),
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
			fmt.Sprintf("%d", value),
			"value must be a non-negative integer",
			fmt.Sprintf("Provide a non-negative integer value for '%s'", field),
		)
	}
	return nil
}

// isEmptyOrNil checks if a value is considered empty or nil.
// Returns true if the value is nil, an empty string, zero, or false.
//
// This helper consolidates common empty-checking patterns across validation files.
//
// Example:
//
//	if isEmptyOrNil(config.Name) {
//	    return NewValidationError("config.name", "", "name cannot be empty", "Provide a valid name")
//	}
func isEmptyOrNil(value any) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == ""
	case int, int32, int64:
		return v == 0
	case bool:
		return !v
	case []string:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

// getMapFieldAsString safely extracts a string field from a map[string]any.
// Returns the string value and a boolean indicating success.
//
// This helper consolidates common type assertion patterns for string fields.
//
// Example:
//
//	if name, ok := getMapFieldAsString(config, "name"); ok {
//	    // Use name
//	}
func getMapFieldAsString(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}

	value, exists := m[key]
	if !exists {
		return "", false
	}

	str, ok := value.(string)
	return str, ok
}

// getMapFieldAsMap safely extracts a nested map from a map[string]any.
// Returns the nested map and a boolean indicating success.
//
// This helper consolidates common type assertion patterns for nested maps.
//
// Example:
//
//	if config, ok := getMapFieldAsMap(frontmatter, "tools"); ok {
//	    // Process tools config
//	}
func getMapFieldAsMap(m map[string]any, key string) (map[string]any, bool) {
	if m == nil {
		return nil, false
	}

	value, exists := m[key]
	if !exists {
		return nil, false
	}

	nested, ok := value.(map[string]any)
	return nested, ok
}

// getMapFieldAsBool safely extracts a boolean field from a map[string]any.
// Returns the boolean value and a boolean indicating success.
//
// This helper consolidates common type assertion patterns for boolean fields.
//
// Example:
//
//	if enabled, ok := getMapFieldAsBool(config, "enabled"); ok && enabled {
//	    // Feature is enabled
//	}
func getMapFieldAsBool(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}

	value, exists := m[key]
	if !exists {
		return false, false
	}

	b, ok := value.(bool)
	return b, ok
}

// getMapFieldAsInt safely extracts an integer field from a map[string]any.
// Returns the integer value and a boolean indicating success.
// Handles both int and float64 types (common in JSON unmarshaling).
//
// This helper consolidates common type assertion patterns for integer fields.
//
// Example:
//
//	if count, ok := getMapFieldAsInt(config, "max_count"); ok {
//	    // Use count
//	}
func getMapFieldAsInt(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}

	value, exists := m[key]
	if !exists {
		return 0, false
	}

	// Handle int type
	if i, ok := value.(int); ok {
		return i, true
	}

	// Handle float64 (common in JSON unmarshaling)
	if f, ok := value.(float64); ok {
		return int(f), true
	}

	return 0, false
}

// fileExists checks if a file exists at the given path.
// Returns true if the file exists and is accessible, false otherwise.
//
// This helper consolidates common file existence checking patterns.
//
// Example:
//
//	if !fileExists(agentPath) {
//	    return NewValidationError("agent.file", agentPath, "file does not exist", "...")
//	}
func fileExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// dirExists checks if a directory exists at the given path.
// Returns true if the directory exists and is accessible, false otherwise.
//
// This helper is useful for validating directory paths in configuration.
//
// Example:
//
//	if !dirExists(workspaceDir) {
//	    return NewValidationError("workspace", workspaceDir, "directory does not exist", "...")
//	}
func dirExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
