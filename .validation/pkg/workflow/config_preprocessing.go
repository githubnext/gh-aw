package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

// preprocessBoolFieldAsString converts the value of a boolean config field
// to a string before YAML unmarshaling. This lets struct fields typed as
// *string accept both literal boolean values (true/false) and GitHub Actions
// expression strings (e.g. "${{ inputs.draft-prs }}").
//
// If the value is a bool it is converted to "true" or "false".
// If the value is a string it must be a GitHub Actions expression (starts
// with "${{" and ends with "}}"); any other free-form string is rejected
// and an error is returned.
func preprocessBoolFieldAsString(configData map[string]any, fieldName string, debugLog *logger.Logger) error {
	if configData == nil {
		return nil
	}
	if val, exists := configData[fieldName]; exists {
		switch v := val.(type) {
		case bool:
			if v {
				configData[fieldName] = "true"
			} else {
				configData[fieldName] = "false"
			}
			if debugLog != nil {
				debugLog.Printf("Converted %s bool to string before unmarshaling", fieldName)
			}
		case string:
			if !isExpression(v) {
				return fmt.Errorf("field %q must be a boolean or a GitHub Actions expression (e.g. '${{ inputs.flag }}'), got string %q", fieldName, v)
			}
		}
	}
	return nil
}

// preprocessIntFieldAsString converts the value of an integer config field
// to a string before YAML unmarshaling. This lets struct fields typed as
// *string accept both literal integer values and GitHub Actions expression
// strings (e.g. "${{ inputs.max-issues }}").
//
// If the value is an int, int64, float64, or uint64 it is converted to its
// decimal string representation.
// If the value is a string it must be a GitHub Actions expression (starts
// with "${{" and ends with "}}"); any other free-form string is rejected
// and an error is returned.
func preprocessIntFieldAsString(configData map[string]any, fieldName string, debugLog *logger.Logger) error {
	if configData == nil {
		return nil
	}
	if val, exists := configData[fieldName]; exists {
		switch v := val.(type) {
		case int:
			configData[fieldName] = strconv.Itoa(v)
			if debugLog != nil {
				debugLog.Printf("Converted %s int to string before unmarshaling", fieldName)
			}
		case int64:
			configData[fieldName] = strconv.FormatInt(v, 10)
			if debugLog != nil {
				debugLog.Printf("Converted %s int64 to string before unmarshaling", fieldName)
			}
		case float64:
			configData[fieldName] = strconv.Itoa(int(v))
			if debugLog != nil {
				debugLog.Printf("Converted %s float64 to string before unmarshaling", fieldName)
			}
		case uint64:
			configData[fieldName] = strconv.FormatUint(v, 10)
			if debugLog != nil {
				debugLog.Printf("Converted %s uint64 to string before unmarshaling", fieldName)
			}
		case string:
			if !isExpression(v) {
				return fmt.Errorf("field %q must be an integer or a GitHub Actions expression (e.g. '${{ inputs.max }}'), got string %q", fieldName, v)
			}
		}
	}
	return nil
}

// preprocessStringArrayFieldAsTemplatable handles a string-array config field that also
// accepts a GitHub Actions expression string (e.g. "${{ inputs.labels }}").
//
// When the field value is an expression string it is wrapped in a single-element []string
// so that existing YAML struct-unmarshal code (which expects []string) continues to work
// unchanged. The handler config builder then detects this single-element expression slice
// and stores it as a JSON string rather than a JSON array, allowing GitHub Actions to
// evaluate the expression at runtime before the config.json file is written.
//
// Free-form strings that are not GitHub Actions expressions are rejected with an error.
// Array values ([]string, []any) are left untouched for the normal YAML unmarshal path.
func preprocessStringArrayFieldAsTemplatable(configData map[string]any, fieldName string, debugLog *logger.Logger) error {
	if configData == nil {
		return nil
	}
	if val, exists := configData[fieldName]; exists {
		if s, ok := val.(string); ok {
			if !isExpression(s) {
				var exampleExpr string
				if strings.Contains(fieldName, "-") {
					exampleExpr = fmt.Sprintf("${{ inputs['%s'] }}", fieldName)
				} else {
					exampleExpr = fmt.Sprintf("${{ inputs.%s }}", fieldName)
				}
				return fmt.Errorf("field %q must be an array of strings or a GitHub Actions expression (e.g. '%s'), got string %q", fieldName, exampleExpr, s)
			}
			configData[fieldName] = []string{s}
			if debugLog != nil {
				debugLog.Printf("Wrapped %s expression string in single-element array before unmarshaling", fieldName)
			}
		}
	}
	return nil
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
// The debugLog parameter is optional; pass nil to suppress debug output.
func preprocessProtectedFilesField(configData map[string]any, debugLog *logger.Logger) []string {
	if configData == nil {
		return nil
	}
	raw, exists := configData["protected-files"]
	if !exists || raw == nil {
		return nil
	}
	pfMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if policy, ok := pfMap["policy"].(string); ok && policy != "" {
		configData["protected-files"] = policy
		if debugLog != nil {
			debugLog.Printf("protected-files object form: policy=%s", policy)
		}
	} else {
		delete(configData, "protected-files")
		if debugLog != nil {
			debugLog.Print("protected-files object form: no policy, using default")
		}
	}
	return parseStringSliceAny(pfMap["exclude"], debugLog)
}

// preprocessExpiresField handles the common expires field preprocessing pattern.
// This function:
//  1. Parses the expires value through parseExpiresFromConfig (handles integers, strings, and boolean false)
//  2. Handles explicit disablement when expires=false (returns -1)
//  3. Normalizes the value to hours and updates configData["expires"] in place
//  4. Logs the parsed value with the provided logger
//
// Returns true if expires was explicitly disabled with false, false otherwise.
// This helper consolidates duplicate preprocessing logic used in parseCreateIssuesConfig and parseCreateDiscussionsConfig.
func preprocessExpiresField(configData map[string]any, debugLog *logger.Logger) bool {
	expiresDisabled := false
	if configData != nil {
		if expires, exists := configData["expires"]; exists {
			expiresInt := parseExpiresFromConfig(configData)
			if expiresInt == -1 {
				expiresDisabled = true
				configData["expires"] = 0
			} else if expiresInt > 0 {
				configData["expires"] = expiresInt
			} else {
				configData["expires"] = 0
			}
			if debugLog != nil {
				debugLog.Printf("Parsed expires value %v to %d hours (disabled=%t)", expires, expiresInt, expiresDisabled)
			}
		}
	}
	return expiresDisabled
}
