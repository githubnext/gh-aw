package workflow

// UpdateProjectItemsConfig holds configuration for updating items in GitHub Projects v2 boards
type UpdateProjectItemsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseUpdateProjectItemsConfig handles update-project-item configuration
func (c *Compiler) parseUpdateProjectItemsConfig(outputMap map[string]any) *UpdateProjectItemsConfig {
	if configData, exists := outputMap["update-project-item"]; exists {
		config := &UpdateProjectItemsConfig{}
		config.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base configuration (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)
		}

		return config
	}
	return nil
}
