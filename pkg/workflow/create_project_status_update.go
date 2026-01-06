package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var createProjectStatusUpdateLog = logger.New("workflow:create_project_status_update")

// CreateProjectStatusUpdateConfig holds configuration for creating GitHub project status updates
type CreateProjectStatusUpdateConfig struct {
	BaseSafeOutputConfig
	GitHubToken string `yaml:"github-token,omitempty"` // Optional custom GitHub token for project status updates
}

// parseCreateProjectStatusUpdateConfig handles create-project-status-update configuration
func (c *Compiler) parseCreateProjectStatusUpdateConfig(outputMap map[string]any) *CreateProjectStatusUpdateConfig {
	if configData, exists := outputMap["create-project-status-update"]; exists {
		createProjectStatusUpdateLog.Print("Parsing create-project-status-update configuration")
		config := &CreateProjectStatusUpdateConfig{}
		config.Max = 10 // Default max is 10

		if configMap, ok := configData.(map[string]any); ok {
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 10)

			// Parse custom GitHub token
			if token, ok := configMap["github-token"]; ok {
				if tokenStr, ok := token.(string); ok {
					config.GitHubToken = tokenStr
					createProjectStatusUpdateLog.Print("Using custom GitHub token for create-project-status-update")
				}
			}
		}

		createProjectStatusUpdateLog.Printf("Parsed create-project-status-update config: max=%d, hasCustomToken=%v",
			config.Max, config.GitHubToken != "")
		return config
	}
	createProjectStatusUpdateLog.Print("No create-project-status-update configuration found")
	return nil
}
