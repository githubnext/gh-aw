package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var createProjectLog = logger.New("workflow:create_project")

// CreateProjectsConfig holds configuration for creating GitHub Projects V2
type CreateProjectsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
	TargetOwner          string `yaml:"target-owner,omitempty"` // Default target owner (org/user) for the new project
	TitlePrefix          string `yaml:"title-prefix,omitempty"` // Default prefix for auto-generated project titles
}

// parseCreateProjectsConfig handles create-project configuration
func (c *Compiler) parseCreateProjectsConfig(outputMap map[string]any) *CreateProjectsConfig {
	if configData, exists := outputMap["create-project"]; exists {
		createProjectLog.Print("Parsing create-project configuration")
		createProjectsConfig := &CreateProjectsConfig{}
		createProjectsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &createProjectsConfig.BaseSafeOutputConfig, 1)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					createProjectsConfig.GitHubToken = tokenStr
					createProjectLog.Print("Using custom GitHub token for create-project")
				}
			}

			// Parse target-owner if specified
			if targetOwner, exists := configMap["target-owner"]; exists {
				if targetOwnerStr, ok := targetOwner.(string); ok {
					createProjectsConfig.TargetOwner = targetOwnerStr
					createProjectLog.Printf("Default target owner configured: %s", targetOwnerStr)
				}
			}

			// Parse title-prefix if specified
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					createProjectsConfig.TitlePrefix = titlePrefixStr
					createProjectLog.Printf("Title prefix configured: %s", titlePrefixStr)
				}
			}
		}

		createProjectLog.Printf("Parsed create-project config: max=%d, hasCustomToken=%v, hasTargetOwner=%v, hasTitlePrefix=%v",
			createProjectsConfig.Max, createProjectsConfig.GitHubToken != "", createProjectsConfig.TargetOwner != "", createProjectsConfig.TitlePrefix != "")
		return createProjectsConfig
	}
	createProjectLog.Print("No create-project configuration found")
	return nil
}
