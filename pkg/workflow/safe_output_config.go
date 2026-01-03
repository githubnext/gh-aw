package workflow

// parseBaseSafeOutputConfig parses common fields (max) from a config map.
// If defaultMax is provided (>= 0), it will be set as the default value for config.Max
// before parsing the max field from configMap.
// Note: github-token is not parsed here as individual safe output types using the
// unified handler should not have per-type tokens. They use the global safe-outputs.github-token.
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
}
