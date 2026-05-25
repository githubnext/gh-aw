// This file provides parse helper functions for agentic workflow compilation.
//
// These helpers handle coercion and preprocessing of raw configuration data
// before it is passed to validation or generation code.
//
// # Available Helper Functions
//
//   - parseStringSliceAny() - Canonical coercion of []string/[]any to []string; skips non-string items.
//     For GitHub Actions fields where a bare string is valid shorthand for a single-element list
//     (e.g. `needs: job-name`, `state: failure`), handle the string case explicitly at the call site.
//   - coerceStringOrArrayField() - Converts a single string scalar field into a one-element []string
//     for fields that accept either a single value or an array in workflow YAML.
//
// Config normalization helpers such as preprocessProtectedFilesField now live in
// config_preprocessing.go.

package workflow

import "github.com/github/gh-aw/pkg/logger"

// coerceStringOrArrayField converts configData[key] from a string to []string{value}
// so YAML unmarshaling into []string fields succeeds for single-value shorthand.
//
// When key is missing, nil, or already a non-string type, this function is a no-op.
// The debugLog parameter is optional; pass nil to suppress debug output.
func coerceStringOrArrayField(configData map[string]any, key string, debugLog *logger.Logger) {
	if configData == nil {
		return
	}

	if value, exists := configData[key]; exists {
		if stringValue, ok := value.(string); ok {
			configData[key] = []string{stringValue}
			if debugLog != nil {
				debugLog.Printf("Converted single %s string to array before unmarshaling", key)
			}
		}
	}
}

// coerceStringOrArrayFields applies coerceStringOrArrayField to multiple keys.
func coerceStringOrArrayFields(configData map[string]any, keys []string, debugLog *logger.Logger) {
	for _, key := range keys {
		coerceStringOrArrayField(configData, key, debugLog)
	}
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
//	return parseStringSliceAny(raw, debugLog)
func parseStringSliceAny(raw any, debugLog *logger.Logger) []string {
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
			} else if debugLog != nil {
				debugLog.Printf("parseStringSliceAny: skipping non-string item: %T", item)
			}
		}
		return result
	default:
		if debugLog != nil {
			debugLog.Printf("parseStringSliceAny: unexpected type %T, ignoring", raw)
		}
		return nil
	}
}
