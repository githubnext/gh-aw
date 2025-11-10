package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var configLog = logger.New("workflow:config")

// parseLabelsFromConfig extracts and validates labels from a config map
// Returns a slice of label strings, or nil if labels is not present or invalid
func parseLabelsFromConfig(configMap map[string]any) []string {
	if labels, exists := configMap["labels"]; exists {
		configLog.Print("Parsing labels from config")
		if labelsArray, ok := labels.([]any); ok {
			var labelStrings []string
			for _, label := range labelsArray {
				if labelStr, ok := label.(string); ok {
					labelStrings = append(labelStrings, labelStr)
				}
			}
			// Return the slice even if empty (to distinguish from not provided)
			if labelStrings == nil {
				configLog.Print("No valid label strings found, returning empty array")
				return []string{}
			}
			configLog.Printf("Parsed %d labels from config", len(labelStrings))
			return labelStrings
		}
	}
	return nil
}

// parseStringFromConfig is a generic helper that extracts and validates a string value from a config map
// Returns the string value, or empty string if not present or invalid
func parseStringFromConfig(configMap map[string]any, key string) string {
	if value, exists := configMap[key]; exists {
		if valueStr, ok := value.(string); ok {
			configLog.Printf("Parsed %s from config: %s", key, valueStr)
			return valueStr
		}
	}
	return ""
}

// parseTitlePrefixFromConfig extracts and validates title-prefix from a config map
// Returns the title prefix string, or empty string if not present or invalid
func parseTitlePrefixFromConfig(configMap map[string]any) string {
	return parseStringFromConfig(configMap, "title-prefix")
}

// parseTargetRepoFromConfig extracts the target-repo value from a config map.
// Returns the target repository slug as a string, or empty string if not present or invalid.
// This function does not perform any special handling or validation for wildcard values ("*");
// callers are responsible for validating the returned value as needed.
func parseTargetRepoFromConfig(configMap map[string]any) string {
	return parseStringFromConfig(configMap, "target-repo")
}
