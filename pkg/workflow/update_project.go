package workflow

import (
	"fmt"
)

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
		} else if configData == nil {
			// null value means enable with defaults
			// Max already set to 10 above
		}

		return updateProjectConfig
	}
	return nil
}

// parseUpdateProjectConfig handles update-project configuration
func parseUpdateProjectConfig(outputMap map[string]interface{}) (*SafeOutputsConfig, error) {
	if configData, exists := outputMap["update-project"]; exists {
		updateProjectMap, ok := configData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("update-project configuration must be an object")
		}

		config := &UpdateProjectConfig{}

		// Parse max
		if maxVal, exists := updateProjectMap["max"]; exists {
			if maxInt, ok := maxVal.(int); ok {
				config.Max = maxInt
			} else if maxFloat, ok := maxVal.(float64); ok {
				config.Max = int(maxFloat)
			}
		}

		// Parse github_token
		if token, exists := updateProjectMap["github_token"]; exists {
			if tokenStr, ok := token.(string); ok {
				config.GitHubToken = tokenStr
			}
		}

		return &SafeOutputsConfig{UpdateProjects: config}, nil
	}

	return nil, nil
}
