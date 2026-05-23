// This file provides helper functions for parsing safe output configurations.
//
// This file contains parsing utilities for extracting and validating configuration
// values from safe output config maps. These helpers are used across safe output
// processors to parse common configuration patterns.
//
// # Organization Rationale
//
// These parse functions are grouped in a helper file because they:
//   - Share a common purpose (safe output config parsing)
//   - Are used by multiple safe output modules (3+ callers)
//   - Provide stable, reusable parsing patterns
//   - Have clear domain focus (configuration extraction)
//
// This follows the helper file conventions documented in the developer instructions.
// See skills/developer/SKILL.md#helper-file-conventions for details.
//
// # Key Functions
//
// Configuration Array Parsing:
//   - ParseStringArrayFromConfig() - Generic string array extraction
//   - parseLabelsFromConfig() - Extract labels array
//
// Configuration String Parsing:
//   - extractStringFromMap() - Generic string extraction
//   - parseTargetRepoWithValidation() - Extract and validate target repo
//
// Configuration Integer Parsing:
//   - parseExpiresFromConfig() - Extract expiration time
//   - parseRelativeTimeSpec() - Parse relative time specifications
//
// Parser Scaffold:
//   - parseConfigScaffold() - Generic safe-output config parser scaffold

package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
	"github.com/goccy/go-yaml"
)

var configHelpersLog = logger.New("workflow:config_helpers")

// ParseStringArrayFromConfig is a generic helper that extracts and validates a string array from a map
// Returns a slice of strings, or nil if not present or invalid
// If log is provided, it will log the extracted values for debugging
func ParseStringArrayFromConfig(m map[string]any, key string, debugLog *logger.Logger) []string {
	if value, exists := m[key]; exists {
		if debugLog != nil {
			debugLog.Printf("Parsing %s from config", key)
		}
		if strings := parseStringSliceAny(value, debugLog); strings != nil {
			// Return the slice even if empty (to distinguish from not provided)
			if len(strings) == 0 && debugLog != nil {
				debugLog.Printf("No valid %s strings found, returning empty array", key)
			}
			if debugLog != nil {
				debugLog.Printf("Parsed %d %s from config", len(strings), key)
			}
			return strings
		}
	}
	return nil
}

// ParseStringArrayOrExprFromConfig is like ParseStringArrayFromConfig but also accepts a
// GitHub Actions expression string as a valid value.  When the raw value is an expression
// (starts with "${{" and ends with "}}") it is returned as []string{exprString} so that
// the handler config builder can later detect and re-emit it as a JSON string rather than
// a JSON array.
//
// Non-expression bare strings are treated as invalid and nil is returned.
func ParseStringArrayOrExprFromConfig(m map[string]any, key string, debugLog *logger.Logger) []string {
	if value, exists := m[key]; exists {
		if debugLog != nil {
			debugLog.Printf("Parsing %s from config", key)
		}
		// Accept a GitHub Actions expression string: wrap it in a single-element slice.
		if s, ok := value.(string); ok {
			if isExpression(s) {
				if debugLog != nil {
					debugLog.Printf("Field %s is a GitHub Actions expression, wrapping in single-element array", key)
				}
				return []string{s}
			}
			// Non-expression string is invalid for an array field.
			if debugLog != nil {
				debugLog.Printf("Field %q must be an array or a GitHub Actions expression, ignoring non-expression string: %q", key, s)
			}
			return nil
		}
		// Handle arrays (existing logic).
		if strings := parseStringSliceAny(value, debugLog); strings != nil {
			if len(strings) == 0 && debugLog != nil {
				debugLog.Printf("No valid %s strings found, returning empty array", key)
			}
			if debugLog != nil {
				debugLog.Printf("Parsed %d %s from config", len(strings), key)
			}
			return strings
		}
	}
	return nil
}

// extractStringFromMap is a generic helper that extracts and validates a string value from a map
// Returns the string value, or empty string if not present or invalid
// If log is provided, it will log the extracted value for debugging
func extractStringFromMap(m map[string]any, key string, debugLog *logger.Logger) string {
	if value, exists := m[key]; exists {
		if valueStr, ok := value.(string); ok {
			if debugLog != nil {
				debugLog.Printf("Parsed %s from config: %s", key, valueStr)
			}
			return valueStr
		}
	}
	return ""
}

// parseTargetRepoWithValidation extracts the target-repo value from a config map and validates it.
// Returns the target repository slug as a string, or empty string if not present or invalid.
// Returns an error (indicated by the second return value being true) if the value is "*" (wildcard),
// which is not allowed for safe output target repositories.
func parseTargetRepoWithValidation(configMap map[string]any) (string, bool) {
	targetRepoSlug := extractStringFromMap(configMap, "target-repo", configHelpersLog)
	// Validate that target-repo is not "*" - only definite strings are allowed
	if targetRepoSlug == "*" {
		configHelpersLog.Print("Invalid target-repo: wildcard '*' is not allowed")
		return "", true // Return true to indicate validation error
	}
	return targetRepoSlug, false
}

// NOTE: parseExpiresFromConfig and parseRelativeTimeSpec have been moved to time_delta.go
// to consolidate all time parsing logic in a single location. These functions are used
// for parsing expiration configurations in safe output jobs.

// ParseBoolFromConfig is a generic helper that extracts and validates a boolean value from a map.
// Returns the boolean value, or false if not present or invalid.
// If log is provided, it will log the extracted value for debugging.
func ParseBoolFromConfig(m map[string]any, key string, debugLog *logger.Logger) bool {
	if debugLog != nil {
		debugLog.Printf("Parsing %s from config", key)
	}
	result := typeutil.ParseBool(m, key)
	if debugLog != nil {
		debugLog.Printf("Parsed %s from config: %t", key, result)
	}
	return result
}

// unmarshalConfig unmarshals a config value from a map into a typed struct using YAML.
// This provides type-safe parsing by leveraging YAML struct tags on config types.
// Returns an error if the config key doesn't exist, the value can't be marshaled, or unmarshaling fails.
//
// Example usage:
//
//	var config CreateIssuesConfig
//	if err := unmarshalConfig(outputMap, "create-issue", &config, debugLog); err != nil {
//	    return nil, err
//	}
//
// This function:
// 1. Extracts the config value from the map using the provided key
// 2. Marshals it to YAML bytes (preserving structure)
// 3. Unmarshals the YAML into the typed struct (using struct tags for field mapping)
// 4. Validates that all fields are properly typed
func unmarshalConfig(m map[string]any, key string, target any, debugLog *logger.Logger) error {
	configData, exists := m[key]
	if !exists {
		return fmt.Errorf("config key %q not found", key)
	}

	// Handle nil config gracefully - unmarshal empty map
	if configData == nil {
		configData = map[string]any{}
	}

	if debugLog != nil {
		debugLog.Printf("Unmarshaling config for key %q into typed struct", key)
	}

	// Marshal the config data back to YAML bytes
	yamlBytes, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config for %q: %w", key, err)
	}

	// Unmarshal into the typed struct
	if err := yaml.Unmarshal(yamlBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal config for %q: %w", key, err)
	}

	if debugLog != nil {
		debugLog.Printf("Successfully unmarshaled config for key %q", key)
	}

	return nil
}

// parseConfigScaffold is the generic parser scaffold for safe-output handler config parsers.
// It implements the common four-step pattern shared across safe-output handlers:
//  1. Key existence check – returns nil immediately if the key is absent in outputMap
//  2. Entry log: "Parsing <key> configuration"
//  3. Typed unmarshal via unmarshalConfig into a zero-value T
//  4. On unmarshal error: delegate to the onError callback, which handles
//     additional logging and returns the appropriate fallback (or nil to disable)
//
// The caller is responsible for any preprocessing (e.g. preprocessIntFieldAsString)
// that must happen before YAML unmarshaling, and for any postprocessing (e.g. setting
// default max values or computing derived fields) after a successful parse.
//
// Example – empty-config fallback:
//
//	config := parseConfigScaffold(outputMap, "add-labels", addLabelsLog,
//	    func(err error) *AddLabelsConfig {
//	        addLabelsLog.Printf("Failed to unmarshal config: %v", err)
//	        addLabelsLog.Print("Using empty configuration (allows any labels)")
//	        return &AddLabelsConfig{}
//	    })
//
// Example – disable-on-error:
//
//	config := parseConfigScaffold(outputMap, "set-issue-type", setIssueTypeLog,
//	    func(err error) *SetIssueTypeConfig {
//	        setIssueTypeLog.Printf("Failed to unmarshal config, disabling handler: %v", err)
//	        return nil
//	    })
func parseConfigScaffold[T any](
	outputMap map[string]any,
	key string,
	debugLog *logger.Logger,
	onError func(err error) *T,
) *T {
	if _, exists := outputMap[key]; !exists {
		return nil
	}
	debugLog.Printf("Parsing %s configuration", key)
	var config T
	if err := unmarshalConfig(outputMap, key, &config, debugLog); err != nil {
		return onError(err)
	}
	return &config
}
