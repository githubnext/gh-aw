package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var updateProjectLog = logger.New("workflow:update_project")

// UpdateProjectConfig holds configuration for unified project board management
type UpdateProjectConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
}

// parseUpdateProjectConfig handles update-project configuration
func (c *Compiler) parseUpdateProjectConfig(outputMap map[string]any) *UpdateProjectConfig {
	if configData, exists := outputMap["update-project"]; exists {
		updateProjectLog.Print("Parsing update-project configuration")
		updateProjectConfig := &UpdateProjectConfig{}
		updateProjectConfig.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &updateProjectConfig.BaseSafeOutputConfig, 10, false) // Runs in consolidated job

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					updateProjectConfig.GitHubToken = tokenStr
					updateProjectLog.Print("Using custom GitHub token for update-project")
				}
			}
		}

		updateProjectLog.Printf("Parsed update-project config: max=%d, hasCustomToken=%v",
			updateProjectConfig.Max, updateProjectConfig.GitHubToken != "")
		return updateProjectConfig
	}
	updateProjectLog.Print("No update-project configuration found")
	return nil
}
