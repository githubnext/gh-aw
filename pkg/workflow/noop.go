package workflow

// NoOpConfig holds configuration for no-op safe output (logging only)
type NoOpConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseNoOpConfig handles noop configuration
func (c *Compiler) parseNoOpConfig(outputMap map[string]any) *NoOpConfig {
	if configData, exists := outputMap["noop"]; exists {
		// Handle the case where configData is false (explicitly disabled)
		if configBool, ok := configData.(bool); ok && !configBool {
			return nil
		}

		noopConfig := &NoOpConfig{}

		// Handle the case where configData is nil (noop: with no value)
		if configData == nil {
			// Set default max for noop messages
			noopConfig.Max = 1
			return noopConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &noopConfig.BaseSafeOutputConfig, 1, false) // Runs in consolidated job
		}

		return noopConfig
	}

	return nil
}
