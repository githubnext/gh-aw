// This file provides generic map and type conversion utilities.
//
// This file contains low-level helper functions for working with map[string]any
// structures and type conversions. These utilities are used throughout the workflow
// compilation process to safely parse and manipulate configuration data.
//
// # Organization Rationale
//
// These functions are grouped in a helper file because they:
//   - Provide generic, reusable utilities (used by 10+ files)
//   - Have no specific domain focus (work with any map/type data)
//   - Are small, stable functions (< 50 lines each)
//   - Follow clear, single-purpose patterns
//
// This follows the helper file conventions documented in skills/developer/SKILL.md.
//
// # Key Functions
//
// Type Conversion:
//   - parseIntValue() - Safely parse numeric types to int with truncation warnings
//   - isEmptyOrNil() - Check if a value is empty, nil, or zero
//
// Map Operations:
//   - filterMapKeys() - Create new map excluding specified keys
//   - getMapFieldAsString() - Safely extract a string field from a map[string]any
//   - getMapFieldAsMap() - Safely extract a nested map from a map[string]any
//   - getMapFieldAsBool() - Safely extract a boolean field from a map[string]any
//   - getMapFieldAsInt() - Safely extract an integer field from a map[string]any
//
// These utilities handle common type conversion and map manipulation patterns that
// occur frequently during YAML-to-struct parsing and configuration processing.

package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var mapHelpersLog = logger.New("workflow:map_helpers")

// parseIntValue safely parses various numeric types to int
// This is a common utility used across multiple parsing functions
func parseIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		// Check for overflow before converting uint64 to int
		const maxInt = int(^uint(0) >> 1)
		if v > uint64(maxInt) {
			mapHelpersLog.Printf("uint64 value %d exceeds max int value, returning 0", v)
			return 0, false
		}
		return int(v), true
	case float64:
		intVal := int(v)
		// Warn if truncation occurs (value has fractional part)
		if v != float64(intVal) {
			mapHelpersLog.Printf("Float value %.2f truncated to integer %d", v, intVal)
		}
		return intVal, true
	default:
		return 0, false
	}
}

// filterMapKeys creates a new map excluding the specified keys
func filterMapKeys(original map[string]any, excludeKeys ...string) map[string]any {
	excludeSet := make(map[string]bool)
	for _, key := range excludeKeys {
		excludeSet[key] = true
	}

	result := make(map[string]any)
	for key, value := range original {
		if !excludeSet[key] {
			result[key] = value
		}
	}
	return result
}

// isEmptyOrNil evaluates whether a value represents an empty or absent state.
// This consolidates various emptiness checks across the codebase into a single
// reusable function. The function handles multiple value types with appropriate
// emptiness semantics for each.
//
// Returns true when encountering:
//   - nil values (representing absence)
//   - strings that are empty or contain only whitespace
//   - numeric types equal to zero
//   - boolean false
//   - collections (slices, maps) with no elements
//
// Usage pattern:
//
//	if isEmptyOrNil(configValue) {
//	    return NewValidationError("fieldName", "", "required field missing", "provide a value")
//	}
func isEmptyOrNil(candidate any) bool {
	// Handle nil case first
	if candidate == nil {
		return true
	}

	// Type-specific emptiness checks using reflection-free approach
	switch typedValue := candidate.(type) {
	case string:
		// String is empty if blank after trimming whitespace
		return len(strings.TrimSpace(typedValue)) == 0
	case int:
		return typedValue == 0
	case int8:
		return typedValue == 0
	case int16:
		return typedValue == 0
	case int32:
		return typedValue == 0
	case int64:
		return typedValue == 0
	case uint:
		return typedValue == 0
	case uint8:
		return typedValue == 0
	case uint16:
		return typedValue == 0
	case uint32:
		return typedValue == 0
	case uint64:
		return typedValue == 0
	case float32:
		return typedValue == 0.0
	case float64:
		return typedValue == 0.0
	case bool:
		// false represents empty boolean state
		return !typedValue
	case []any:
		return len(typedValue) == 0
	case map[string]any:
		return len(typedValue) == 0
	}

	// Non-nil values of unrecognized types are considered non-empty
	return false
}

// getMapFieldAsString retrieves a string value from a configuration map with safe type handling.
// This function wraps the common pattern of extracting string fields from map[string]any structures
// that result from YAML parsing, providing consistent error behavior and logging.
//
// The function returns the fallback value in these scenarios:
//   - Source map is nil
//   - Requested key doesn't exist in map
//   - Value at key is not a string type
//
// Parameters:
//   - source: The configuration map to query
//   - fieldKey: The key to look up in the map
//   - fallback: Value returned when extraction fails
//
// Example usage:
//
//	titleValue := getMapFieldAsString(frontmatter, "title", "")
//	if titleValue == "" {
//	    return NewValidationError("title", "", "title required", "provide a title")
//	}
func getMapFieldAsString(source map[string]any, fieldKey string, fallback string) string {
	// Early return for nil map
	if source == nil {
		return fallback
	}

	// Attempt to retrieve value
	retrievedValue, keyFound := source[fieldKey]
	if !keyFound {
		return fallback
	}

	// Verify type before returning
	stringValue, isString := retrievedValue.(string)
	if !isString {
		mapHelpersLog.Printf("Type mismatch for key %q: expected string, found %T", fieldKey, retrievedValue)
		return fallback
	}

	return stringValue
}

// getMapFieldAsMap retrieves a nested map value from a configuration map with safe type handling.
// This consolidates the pattern of extracting nested configuration sections while handling
// type mismatches gracefully. Returns nil when the field cannot be extracted as a map.
//
// Parameters:
//   - source: The parent configuration map
//   - fieldKey: The key identifying the nested map
//
// Example usage:
//
//	toolsSection := getMapFieldAsMap(config, "tools")
//	if toolsSection != nil {
//	    playwrightConfig := getMapFieldAsMap(toolsSection, "playwright")
//	}
func getMapFieldAsMap(source map[string]any, fieldKey string) map[string]any {
	// Guard against nil source
	if source == nil {
		return nil
	}

	// Look up the field
	retrievedValue, keyFound := source[fieldKey]
	if !keyFound {
		return nil
	}

	// Type assert to nested map
	mapValue, isMap := retrievedValue.(map[string]any)
	if !isMap {
		mapHelpersLog.Printf("Type mismatch for key %q: expected map[string]any, found %T", fieldKey, retrievedValue)
		return nil
	}

	return mapValue
}

// getMapFieldAsBool retrieves a boolean value from a configuration map with safe type handling.
// This wraps the pattern of extracting boolean configuration flags while providing consistent
// fallback behavior when the value is missing or has an unexpected type.
//
// Parameters:
//   - source: The configuration map to query
//   - fieldKey: The key to look up
//   - fallback: Value returned when extraction fails
//
// Example usage:
//
//	sandboxEnabled := getMapFieldAsBool(config, "sandbox", false)
//	if sandboxEnabled {
//	    // Enable sandbox mode
//	}
func getMapFieldAsBool(source map[string]any, fieldKey string, fallback bool) bool {
	// Handle nil source
	if source == nil {
		return fallback
	}

	// Retrieve value from map
	retrievedValue, keyFound := source[fieldKey]
	if !keyFound {
		return fallback
	}

	// Verify boolean type
	booleanValue, isBoolean := retrievedValue.(bool)
	if !isBoolean {
		mapHelpersLog.Printf("Type mismatch for key %q: expected bool, found %T", fieldKey, retrievedValue)
		return fallback
	}

	return booleanValue
}

// getMapFieldAsInt retrieves an integer value from a configuration map with automatic numeric type conversion.
// This function handles the common pattern of extracting numeric config values that may be represented
// as various numeric types in YAML (int, int64, float64, uint64). It delegates to parseIntValue for
// the actual type conversion logic.
//
// Parameters:
//   - source: The configuration map to query
//   - fieldKey: The key to look up
//   - fallback: Value returned when extraction or conversion fails
//
// Example usage:
//
//	retentionDays := getMapFieldAsInt(config, "retention-days", 30)
//	if err := validateIntRange(retentionDays, 1, 90, "retention-days"); err != nil {
//	    return err
//	}
func getMapFieldAsInt(source map[string]any, fieldKey string, fallback int) int {
	// Guard against nil source
	if source == nil {
		return fallback
	}

	// Look up the value
	retrievedValue, keyFound := source[fieldKey]
	if !keyFound {
		return fallback
	}

	// Attempt numeric conversion using existing utility
	convertedInt, conversionOk := parseIntValue(retrievedValue)
	if !conversionOk {
		mapHelpersLog.Printf("Failed to convert key %q to int: got %T", fieldKey, retrievedValue)
		return fallback
	}

	return convertedInt
}
