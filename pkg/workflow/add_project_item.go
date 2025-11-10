package workflow

// AddProjectItemsConfig holds configuration for adding items to GitHub Projects v2 boards
type AddProjectItemsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseAddProjectItemsConfig handles add-project-item configuration
func (c *Compiler) parseAddProjectItemsConfig(outputMap map[string]any) *AddProjectItemsConfig {
	if configData, exists := outputMap["add-project-item"]; exists {
		config := &AddProjectItemsConfig{}
		config.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base configuration (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)
		}

		return config
	}
	return nil
}
