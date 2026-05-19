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
//   - preprocessProtectedFilesField() - Normalises the "protected-files" field from its object or
//     string form before downstream enum validation.

package workflow

import "github.com/github/gh-aw/pkg/logger"

// coerceStringOrArrayField converts configData[key] from a string to []string{value}
// so YAML unmarshaling into []string fields succeeds for single-value shorthand.
//
// When key is missing, nil, or already a non-string type, this function is a no-op.
// The log parameter is optional; pass nil to suppress debug output.
func coerceStringOrArrayField(configData map[string]any, key string, log *logger.Logger) {
	if configData == nil {
		return
	}

	if value, exists := configData[key]; exists {
		if stringValue, ok := value.(string); ok {
			configData[key] = []string{stringValue}
			if log != nil {
				log.Printf("Converted single %s string to array before unmarshaling", key)
			}
		}
	}
}

// coerceStringOrArrayFields applies coerceStringOrArrayField to multiple keys.
func coerceStringOrArrayFields(configData map[string]any, keys []string, log *logger.Logger) {
	for _, key := range keys {
		coerceStringOrArrayField(configData, key, log)
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
