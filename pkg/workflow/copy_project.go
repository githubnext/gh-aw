package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var copyProjectLog = logger.New("workflow:copy_project")

// CopyProjectsConfig holds configuration for copying GitHub Projects V2
type CopyProjectsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
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
		}

		copyProjectLog.Printf("Parsed copy-project config: max=%d, hasCustomToken=%v",
			copyProjectsConfig.Max, copyProjectsConfig.GitHubToken != "")
		return copyProjectsConfig
	}
	copyProjectLog.Print("No copy-project configuration found")
	return nil
}
