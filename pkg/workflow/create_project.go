package workflow

// CreateProjectsConfig holds configuration for creating GitHub Projects v2 boards
type CreateProjectsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseCreateProjectsConfig handles create-project configuration
func (c *Compiler) parseCreateProjectsConfig(outputMap map[string]any) *CreateProjectsConfig {
	if configData, exists := outputMap["create-project"]; exists {
		config := &CreateProjectsConfig{}
		config.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base configuration (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)
		}

		return config
	}
	return nil
}
