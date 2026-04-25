// This file provides generic map utilities.
//
// This file contains low-level helper functions for working with map[string]any
// structures. These utilities are used throughout the workflow compilation process
// to safely parse and manipulate configuration data.
//
// # Organization Rationale
//
// These functions are grouped in a helper file because they:
//   - Provide generic, reusable utilities (used by 10+ files)
//   - Have no specific domain focus (work with any map data)
//   - Are small, stable functions (< 50 lines each)
//   - Follow clear, single-purpose patterns
//
// This follows the helper file conventions documented in skills/developer/SKILL.md.
//
// # Key Functions
//
// Map Operations:
//   - excludeMapKeys() - Create new map excluding specified keys
//   - sortedMapKeys() - Return sorted keys of a map[string]string
//
// For type conversion utilities, use pkg/typeutil directly:
//   - typeutil.ParseIntValue() - Strictly parse numeric types to int; returns (value, ok).
//   - typeutil.SafeUint64ToInt() - Convert uint64 to int, returning 0 on overflow.
//   - typeutil.SafeUintToInt() - Convert uint to int, returning 0 on overflow.
//   - typeutil.ConvertToInt() - Leniently convert any value to int, returning 0 on failure.
//   - typeutil.ConvertToFloat() - Safely convert any value to float64.
//   - typeutil.ParseBool() - Extract a bool from map[string]any by key.
//
// These utilities handle common map manipulation patterns that occur frequently
// during YAML-to-struct parsing and configuration processing.

package workflow

import (
	"maps"
	"slices"
)

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
	return slices.Sorted(maps.Keys(m))
}
