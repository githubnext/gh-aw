package workflow

// extractStringValue extracts a string value from the frontmatter map
func extractStringValue(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}

	if strValue, ok := value.(string); ok {
		return strValue
	}

	return ""
}

// parseIntValue safely parses various numeric types to int
// This is a common utility used across multiple parsing functions
func parseIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// filterMapKeys creates a new map excluding the specified keys
func filterMapKeys(original map[string]any, excludeKeys ...string) map[string]any {
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
