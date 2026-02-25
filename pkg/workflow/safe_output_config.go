package workflow

import "strings"

// parseBaseSafeOutputConfig parses common fields (max, github-token) from a config map.
// If defaultMax is provided (> 0), it will be set as the default value for config.Max
// before parsing the max field from configMap. Supports both integer values and GitHub
// Actions expression strings (e.g. "${{ inputs.max }}").
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	// Set default max if provided
	if defaultMax > 0 {
		config.Max = defaultIntStr(defaultMax)
	}

	// Parse max (this will override the default if present in configMap)
	if max, exists := configMap["max"]; exists {
		switch v := max.(type) {
		case string:
			// Accept GitHub Actions expression strings
			if strings.HasPrefix(v, "${{") && strings.HasSuffix(v, "}}") {
				config.Max = &v
			}
		default:
			// Convert integer/float64/etc to string via parseIntValue
			if maxInt, ok := parseIntValue(max); ok {
				s := defaultIntStr(maxInt)
				config.Max = s
			}
		}
	}

	// Parse github-token
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			config.GitHubToken = githubTokenStr
		}
	}
}
