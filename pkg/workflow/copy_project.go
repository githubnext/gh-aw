package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var copyProjectLog = logger.New("workflow:copy_project")

// CopyProjectsConfig holds configuration for copying GitHub Projects V2
type CopyProjectsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
	SourceProject        string `yaml:"source-project,omitempty"` // Default source project URL to copy from
	TargetOwner          string `yaml:"target-owner,omitempty"`   // Default target owner (org/user) for the new project
}

// parseCopyProjectsConfig handles copy-project configuration
func (c *Compiler) parseCopyProjectsConfig(outputMap map[string]any) *CopyProjectsConfig {
	if configData, exists := outputMap["copy-project"]; exists {
		copyProjectLog.Print("Parsing copy-project configuration")
		copyProjectsConfig := &CopyProjectsConfig{}
		copyProjectsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &copyProjectsConfig.BaseSafeOutputConfig, 1)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					copyProjectsConfig.GitHubToken = tokenStr
					copyProjectLog.Print("Using custom GitHub token for copy-project")
				}
			}

			// Parse source-project if specified
			if sourceProject, exists := configMap["source-project"]; exists {
				if sourceProjectStr, ok := sourceProject.(string); ok {
					copyProjectsConfig.SourceProject = sourceProjectStr
					copyProjectLog.Printf("Default source project configured: %s", sourceProjectStr)
				}
			}

			// Parse target-owner if specified
			if targetOwner, exists := configMap["target-owner"]; exists {
				if targetOwnerStr, ok := targetOwner.(string); ok {
					copyProjectsConfig.TargetOwner = targetOwnerStr
					copyProjectLog.Printf("Default target owner configured: %s", targetOwnerStr)
				}
			}
		}

		copyProjectLog.Printf("Parsed copy-project config: max=%d, hasCustomToken=%v, hasSourceProject=%v, hasTargetOwner=%v",
			copyProjectsConfig.Max, copyProjectsConfig.GitHubToken != "", copyProjectsConfig.SourceProject != "", copyProjectsConfig.TargetOwner != "")
		return copyProjectsConfig
	}
	copyProjectLog.Print("No copy-project configuration found")
	return nil
}
