package workflow

// UpdateProjectConfig holds configuration for unified project board management
type UpdateProjectConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
}

// parseUpdateProjectConfig handles update-project configuration
func (c *Compiler) parseUpdateProjectConfig(outputMap map[string]any) *UpdateProjectConfig {
	if configData, exists := outputMap["update-project"]; exists {
		updateProjectConfig := &UpdateProjectConfig{}
		updateProjectConfig.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &updateProjectConfig.BaseSafeOutputConfig)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					updateProjectConfig.GitHubToken = tokenStr
				}
			}
		}

		return updateProjectConfig
	}
	return nil
}
