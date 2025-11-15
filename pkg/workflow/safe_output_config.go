package workflow

// parseBaseSafeOutputConfig parses common fields (max, github-token) from a config map.
// If defaultMax is provided (>= 0), it will be set as the default value for config.Max
// before parsing the max field from configMap.
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	// Set default max if provided
	if defaultMax >= 0 {
		config.Max = defaultMax
	}

	// Parse max (this will override the default if present in configMap)
	if max, exists := configMap["max"]; exists {
		if maxInt, ok := parseIntValue(max); ok {
			config.Max = maxInt
		}
	}

	// Parse github-token
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			config.GitHubToken = githubTokenStr
		}
	}
}
