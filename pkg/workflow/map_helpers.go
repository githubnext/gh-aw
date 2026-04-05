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
//   - parseIntValue() - Strictly parse numeric types to int; returns (value, ok). Use when
//     the caller needs to distinguish "missing/invalid" from a zero value, or when string
//     inputs are not expected (e.g. YAML config field parsing).
//   - safeUint64ToInt() - Convert uint64 to int, returning 0 on overflow
//   - safeUintToInt() - Convert uint to int, returning 0 on overflow (thin wrapper around safeUint64ToInt)
//   - ConvertToInt() - Leniently convert any value (int/int64/float64/string) to int, returning 0
//     on failure. Use when the input may come from heterogeneous sources such as JSON metrics,
//     log parsing, or user-provided strings where a zero default on failure is acceptable.
//   - ConvertToFloat() - Safely convert any value (float64/int/int64/string) to float64
//
// Map Operations:
//   - excludeMapKeys() - Create new map excluding specified keys
//   - sortedMapKeys() - Return sorted keys of a map[string]string
//
// These utilities handle common type conversion and map manipulation patterns that
// occur frequently during YAML-to-struct parsing and configuration processing.

package workflow

import (
	"math"
	"sort"
	"strconv"

	"github.com/github/gh-aw/pkg/logger"
)

var mapHelpersLog = logger.New("workflow:map_helpers")

// parseIntValue strictly parses numeric types to int, returning (value, true) on success
// and (0, false) for any unrecognized or non-numeric type.
//
// Use this when the caller needs to distinguish a missing/invalid value from a legitimate
// zero, or when string inputs are not expected (e.g. YAML config field parsing where the
// YAML library has already produced a typed numeric value).
//
// For lenient conversion that also handles string inputs and returns 0 on failure, use
// ConvertToInt instead.
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

// safeUint64ToInt safely converts uint64 to int, returning 0 if overflow would occur
func safeUint64ToInt(u uint64) int {
	if u > math.MaxInt {
		return 0 // Return 0 (engine default) if value would overflow
	}
	return int(u)
}

// safeUintToInt safely converts uint to int, returning 0 if overflow would occur.
// This is a thin wrapper around safeUint64ToInt that widens the uint argument first.
func safeUintToInt(u uint) int { return safeUint64ToInt(uint64(u)) }

// excludeMapKeys creates a new map excluding the specified keys
func excludeMapKeys(original map[string]any, excludeKeys ...string) map[string]any {
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

// sortedMapKeys returns the keys of a map[string]string in sorted order.
// Used to produce deterministic output when writing environment variables.
func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ConvertToInt leniently converts any value to int, returning 0 on failure.
//
// Unlike parseIntValue, this function also handles string inputs via strconv.Atoi,
// making it suitable for heterogeneous sources such as JSON metrics, log-parsed data,
// or user-provided configuration where a zero default on failure is acceptable and
// the caller does not need to distinguish "invalid" from a genuine zero.
//
// For strict numeric-only parsing where the caller must distinguish missing/invalid
// values from zero, use parseIntValue instead.
func ConvertToInt(val any) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		intVal := int(v)
		// Warn if truncation occurs (value has fractional part)
		if v != float64(intVal) {
			mapHelpersLog.Printf("Float value %.2f truncated to integer %d", v, intVal)
		}
		return intVal
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// ConvertToFloat safely converts any to float64
func ConvertToFloat(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}
