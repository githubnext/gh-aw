package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var configHelpersLog = logger.New("workflow:config_helpers")

// parseLabelsFromConfig extracts and validates labels from a config map
// Returns a slice of label strings, or nil if labels is not present or invalid
func parseLabelsFromConfig(configMap map[string]any) []string {
	if labels, exists := configMap["labels"]; exists {
		configHelpersLog.Print("Parsing labels from config")
		if labelsArray, ok := labels.([]any); ok {
			var labelStrings []string
			for _, label := range labelsArray {
				if labelStr, ok := label.(string); ok {
					labelStrings = append(labelStrings, labelStr)
				}
			}
			// Return the slice even if empty (to distinguish from not provided)
			if labelStrings == nil {
				configHelpersLog.Print("No valid label strings found, returning empty array")
				return []string{}
			}
			configHelpersLog.Printf("Parsed %d labels from config", len(labelStrings))
			return labelStrings
		}
	}
	return nil
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

// parseParticipantsFromConfig extracts and validates participants (assignees/reviewers) from a config map.
// Supports both string (single participant) and array (multiple participants) formats.
// Returns a slice of participant usernames, or nil if not present or invalid.
// The participantKey parameter specifies which key to look for (e.g., "assignees" or "reviewers").
func parseParticipantsFromConfig(configMap map[string]any, participantKey string) []string {
	if participants, exists := configMap[participantKey]; exists {
		configHelpersLog.Printf("Parsing %s from config", participantKey)

		// Handle single string format
		if participantStr, ok := participants.(string); ok {
			configHelpersLog.Printf("Parsed single %s: %s", participantKey, participantStr)
			return []string{participantStr}
		}

		// Handle array format
		if participantsArray, ok := participants.([]any); ok {
			var participantStrings []string
			for _, participant := range participantsArray {
				if participantStr, ok := participant.(string); ok {
					participantStrings = append(participantStrings, participantStr)
				}
			}
			// Return the slice even if empty (to distinguish from not provided)
			if participantStrings == nil {
				configHelpersLog.Printf("No valid %s strings found, returning empty array", participantKey)
				return []string{}
			}
			configHelpersLog.Printf("Parsed %d %s from config", len(participantStrings), participantKey)
			return participantStrings
		}
	}
	return nil
}

// parseAllowedReposFromConfig extracts and validates allowed-repos from a config map.
// Returns a slice of repository slugs (owner/repo format), or nil if not present or invalid.
func parseAllowedReposFromConfig(configMap map[string]any) []string {
	if allowedRepos, exists := configMap["allowed-repos"]; exists {
		configHelpersLog.Print("Parsing allowed-repos from config")
		if reposArray, ok := allowedRepos.([]any); ok {
			var repoStrings []string
			for _, repo := range reposArray {
				if repoStr, ok := repo.(string); ok {
					repoStrings = append(repoStrings, repoStr)
				}
			}
			// Return the slice even if empty (to distinguish from not provided)
			if repoStrings == nil {
				configHelpersLog.Print("No valid allowed-repos strings found, returning empty array")
				return []string{}
			}
			configHelpersLog.Printf("Parsed %d allowed-repos from config", len(repoStrings))
			return repoStrings
		}
	}
	return nil
}

// parseAllowedLabelsFromConfig extracts and validates allowed-labels from a config map.
// Returns a slice of label strings, or nil if not present or invalid.
func parseAllowedLabelsFromConfig(configMap map[string]any) []string {
	if allowedLabels, exists := configMap["allowed-labels"]; exists {
		configHelpersLog.Print("Parsing allowed-labels from config")
		if labelsArray, ok := allowedLabels.([]any); ok {
			var labelStrings []string
			for _, label := range labelsArray {
				if labelStr, ok := label.(string); ok {
					labelStrings = append(labelStrings, labelStr)
				}
			}
			// Return the slice even if empty (to distinguish from not provided)
			if labelStrings == nil {
				configHelpersLog.Print("No valid allowed-labels strings found, returning empty array")
				return []string{}
			}
			configHelpersLog.Printf("Parsed %d allowed-labels from config", len(labelStrings))
			return labelStrings
		}
	}
	return nil
}

// parseExpiresFromConfig parses expires value from config map
// Supports both integer (days) and string formats like "7d", "2w", "1m", "1y"
// Returns the number of days, or 0 if invalid or not present
func parseExpiresFromConfig(configMap map[string]any) int {
	configHelpersLog.Printf("DEBUG: parseExpiresFromConfig called with configMap: %+v", configMap)
	if expires, exists := configMap["expires"]; exists {
		// Try numeric types first
		switch v := expires.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case uint64:
			return int(v)
		case string:
			// Parse relative time specification like "7d", "2w", "1m", "1y"
			return parseRelativeTimeSpec(v)
		}
	}
	return 0
}

// parseRelativeTimeSpec parses a relative time specification string
// Supports: d (days), w (weeks), m (months ~30 days), y (years ~365 days)
// Examples: "7d" = 7 days, "2w" = 14 days, "1m" = 30 days, "1y" = 365 days
// Returns 0 if the format is invalid
func parseRelativeTimeSpec(spec string) int {
	configHelpersLog.Printf("DEBUG: parseRelativeTimeSpec called with spec: %s", spec)
	if spec == "" {
		return 0
	}

	// Get the last character (unit)
	unit := spec[len(spec)-1:]
	// Get the number part
	numStr := spec[:len(spec)-1]

	// Parse the number
	var num int
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil || num <= 0 {
		configHelpersLog.Printf("Invalid expires time spec number: %s", spec)
		return 0
	}

	// Convert to days based on unit
	switch unit {
	case "d", "D":
		return num // days
	case "w", "W":
		return num * 7 // weeks to days
	case "m", "M":
		return num * 30 // months to days (approximate)
	case "y", "Y":
		return num * 365 // years to days (approximate)
	default:
		configHelpersLog.Printf("Invalid expires time spec unit: %s", spec)
		return 0
	}
}
