package workflow

// CreateProjectConfig holds configuration for creating GitHub Projects v2
type CreateProjectConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
}

// parseCreateProjectConfig handles create-project configuration
func (c *Compiler) parseCreateProjectConfig(outputMap map[string]any) *CreateProjectConfig {
	if configData, exists := outputMap["create-project"]; exists {
		createProjectConfig := &CreateProjectConfig{}
		createProjectConfig.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &createProjectConfig.BaseSafeOutputConfig, 10)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					createProjectConfig.GitHubToken = tokenStr
				}
			}
		}

		return createProjectConfig
	}
	return nil
}
