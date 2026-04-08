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
//   - parseAllowedLabelsFromConfig() - Extract allowed labels array
//
// Configuration String Parsing:
//   - extractStringFromMap() - Generic string extraction
//   - parseTitlePrefixFromConfig() - Extract title prefix
//   - parseTargetRepoFromConfig() - Extract target repository
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
func ParseStringArrayFromConfig(m map[string]any, key string, log *logger.Logger) []string {
	if value, exists := m[key]; exists {
		if log != nil {
			log.Printf("Parsing %s from config", key)
		}
		if arrayValue, ok := value.([]any); ok {
			var strings []string
			for _, item := range arrayValue {
				if strVal, ok := item.(string); ok {
					strings = append(strings, strVal)
				}
			}
			// Return the slice even if empty (to distinguish from not provided)
			if strings == nil {
				if log != nil {
					log.Printf("No valid %s strings found, returning empty array", key)
				}
				return []string{}
			}
			if log != nil {
				log.Printf("Parsed %d %s from config", len(strings), key)
			}
			return strings
		}
	}
	return nil
}

// parseLabelsFromConfig extracts and validates labels from a config map
// Returns a slice of label strings, or nil if labels is not present or invalid
func parseLabelsFromConfig(configMap map[string]any) []string {
	return ParseStringArrayFromConfig(configMap, "labels", configHelpersLog)
}

// extractStringFromMap is a generic helper that extracts and validates a string value from a map
// Returns the string value, or empty string if not present or invalid
// If log is provided, it will log the extracted value for debugging
func extractStringFromMap(m map[string]any, key string, log *logger.Logger) string {
	if value, exists := m[key]; exists {
		if valueStr, ok := value.(string); ok {
			if log != nil {
				log.Printf("Parsed %s from config: %s", key, valueStr)
			}
			return valueStr
		}
	}
	return ""
}

// parseTitlePrefixFromConfig extracts and validates title-prefix from a config map
// Returns the title prefix string, or empty string if not present or invalid
func parseTitlePrefixFromConfig(configMap map[string]any) string {
	return extractStringFromMap(configMap, "title-prefix", configHelpersLog)
}

// parseTargetRepoFromConfig extracts the target-repo value from a config map.
// Returns the target repository slug as a string, or empty string if not present or invalid.
// This function does not perform any special handling or validation for wildcard values ("*");
// callers are responsible for validating the returned value as needed.
func parseTargetRepoFromConfig(configMap map[string]any) string {
	return extractStringFromMap(configMap, "target-repo", configHelpersLog)
}

// parseTargetRepoWithValidation extracts the target-repo value from a config map and validates it.
// Returns the target repository slug as a string, or empty string if not present or invalid.
// Returns an error (indicated by the second return value being true) if the value is "*" (wildcard),
// which is not allowed for safe output target repositories.
func parseTargetRepoWithValidation(configMap map[string]any) (string, bool) {
	targetRepoSlug := parseTargetRepoFromConfig(configMap)
	// Validate that target-repo is not "*" - only definite strings are allowed
	if targetRepoSlug == "*" {
		configHelpersLog.Print("Invalid target-repo: wildcard '*' is not allowed")
		return "", true // Return true to indicate validation error
	}
	return targetRepoSlug, false
}

// parseAllowedLabelsFromConfig extracts and validates allowed-labels from a config map.
// Returns a slice of label strings, or nil if not present or invalid.
func parseAllowedLabelsFromConfig(configMap map[string]any) []string {
	return ParseStringArrayFromConfig(configMap, "allowed-labels", configHelpersLog)
}

// parseAllowedReposFromConfig extracts and validates allowed-repos from a config map.
// Returns a slice of repository slugs in "owner/repo" format.
// Returns nil when the key is not present or the value is not a valid array type.
// Returns an empty slice when the key exists but contains no valid strings.
func parseAllowedReposFromConfig(configMap map[string]any) []string {
	return ParseStringArrayFromConfig(configMap, "allowed-repos", configHelpersLog)
}

// NOTE: parseExpiresFromConfig and parseRelativeTimeSpec have been moved to time_delta.go
// to consolidate all time parsing logic in a single location. These functions are used
// for parsing expiration configurations in safe output jobs.

// preprocessExpiresField handles the common expires field preprocessing pattern.
// This function:
//  1. Parses the expires value through parseExpiresFromConfig (handles integers, strings, and boolean false)
//  2. Handles explicit disablement when expires=false (returns -1)
//  3. Normalizes the value to hours and updates configData["expires"] in place
//  4. Logs the parsed value with the provided logger
//
// Returns true if expires was explicitly disabled with false, false otherwise.
// This helper consolidates duplicate preprocessing logic used in parseIssuesConfig and parseDiscussionsConfig.
func preprocessExpiresField(configData map[string]any, log *logger.Logger) bool {
	expiresDisabled := false
	if configData != nil {
		if expires, exists := configData["expires"]; exists {
			// Always parse the expires value through parseExpiresFromConfig
			// This handles: integers (days), strings (time specs like "48h"), and boolean false
			expiresInt := parseExpiresFromConfig(configData)
			if expiresInt == -1 {
				// Explicitly disabled with false
				expiresDisabled = true
				configData["expires"] = 0
			} else if expiresInt > 0 {
				configData["expires"] = expiresInt
			} else {
				// Invalid or missing - set to 0
				configData["expires"] = 0
			}
			if log != nil {
				log.Printf("Parsed expires value %v to %d hours (disabled=%t)", expires, expiresInt, expiresDisabled)
			}
		}
	}
	return expiresDisabled
}

// ParseBoolFromConfig is a generic helper that extracts and validates a boolean value from a map.
// Returns the boolean value, or false if not present or invalid.
// If log is provided, it will log the extracted value for debugging.
func ParseBoolFromConfig(m map[string]any, key string, log *logger.Logger) bool {
	if log != nil {
		log.Printf("Parsing %s from config", key)
	}
	result := typeutil.ParseBool(m, key)
	if log != nil {
		log.Printf("Parsed %s from config: %t", key, result)
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
//	if err := unmarshalConfig(outputMap, "create-issue", &config, log); err != nil {
//	    return nil, err
//	}
//
// This function:
// 1. Extracts the config value from the map using the provided key
// 2. Marshals it to YAML bytes (preserving structure)
// 3. Unmarshals the YAML into the typed struct (using struct tags for field mapping)
// 4. Validates that all fields are properly typed
func unmarshalConfig(m map[string]any, key string, target any, log *logger.Logger) error {
	configData, exists := m[key]
	if !exists {
		return fmt.Errorf("config key %q not found", key)
	}

	// Handle nil config gracefully - unmarshal empty map
	if configData == nil {
		configData = map[string]any{}
	}

	if log != nil {
		log.Printf("Unmarshaling config for key %q into typed struct", key)
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

	if log != nil {
		log.Printf("Successfully unmarshaled config for key %q", key)
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
	log *logger.Logger,
	onError func(err error) *T,
) *T {
	if _, exists := outputMap[key]; !exists {
		return nil
	}
	log.Printf("Parsing %s configuration", key)
	var config T
	if err := unmarshalConfig(outputMap, key, &config, log); err != nil {
		return onError(err)
	}
	return &config
}
