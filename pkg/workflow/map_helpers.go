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
// Type Conversion (delegated to pkg/typeutil for general-purpose reuse):
//   - parseIntValue() - Strictly parse numeric types to int; returns (value, ok). Use when
//     the caller needs to distinguish "missing/invalid" from a zero value, or when string
//     inputs are not expected (e.g. YAML config field parsing). Delegates to typeutil.ParseIntValue.
//   - safeUint64ToInt() - Convert uint64 to int, returning 0 on overflow. Delegates to typeutil.SafeUint64ToInt.
//   - safeUintToInt() - Convert uint to int, returning 0 on overflow. Delegates to typeutil.SafeUintToInt.
//   - ConvertToInt() - Leniently convert any value (int/int64/float64/string) to int, returning 0
//     on failure. Use when the input may come from heterogeneous sources such as JSON metrics,
//     log parsing, or user-provided strings where a zero default on failure is acceptable.
//     Delegates to typeutil.ConvertToInt.
//   - ConvertToFloat() - Safely convert any value (float64/int/int64/string) to float64.
//     Delegates to typeutil.ConvertToFloat.
//
// Map Operations:
//   - excludeMapKeys() - Create new map excluding specified keys
//   - sortedMapKeys() - Return sorted keys of a map[string]string
//
// These utilities handle common type conversion and map manipulation patterns that
// occur frequently during YAML-to-struct parsing and configuration processing.

package workflow

import (
	"sort"

	"github.com/github/gh-aw/pkg/typeutil"
)

// parseIntValue strictly parses numeric types to int, returning (value, true) on success
// and (0, false) for any unrecognized or non-numeric type.
//
// Use this when the caller needs to distinguish a missing/invalid value from a legitimate
// zero, or when string inputs are not expected (e.g. YAML config field parsing where the
// YAML library has already produced a typed numeric value).
//
// For lenient conversion that also handles string inputs and returns 0 on failure, use
// ConvertToInt instead.
//
// This is a package-private alias for typeutil.ParseIntValue.
func parseIntValue(value any) (int, bool) { return typeutil.ParseIntValue(value) }

// safeUint64ToInt safely converts uint64 to int, returning 0 if overflow would occur.
// This is a package-private alias for typeutil.SafeUint64ToInt.
func safeUint64ToInt(u uint64) int { return typeutil.SafeUint64ToInt(u) }

// safeUintToInt safely converts uint to int, returning 0 if overflow would occur.
// This is a package-private alias for typeutil.SafeUintToInt.
func safeUintToInt(u uint) int { return typeutil.SafeUintToInt(u) }

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
//
// This is a workflow-package alias for typeutil.ConvertToInt. For new code outside
// this package, prefer using typeutil.ConvertToInt directly.
func ConvertToInt(val any) int { return typeutil.ConvertToInt(val) }

// ConvertToFloat safely converts any value to float64, returning 0 on failure.
//
// Supported input types: float64, int, int64, and string (parsed via strconv.ParseFloat).
// Returns 0 for any other type or for strings that cannot be parsed as a float.
//
// This is a workflow-package alias for typeutil.ConvertToFloat. For new code outside
// this package, prefer using typeutil.ConvertToFloat directly.
func ConvertToFloat(val any) float64 { return typeutil.ConvertToFloat(val) }
