package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var listProjectItemsWithUrlsLog = logger.New("workflow:list_project_items_with_urls")

// ListProjectItemsWithUrlsConfig holds configuration for listing project items with URLs
type ListProjectItemsWithUrlsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	GitHubToken          string `yaml:"github-token,omitempty"`
}

// parseListProjectItemsWithUrlsConfig handles list-project-items-with-urls configuration
func (c *Compiler) parseListProjectItemsWithUrlsConfig(outputMap map[string]any) *ListProjectItemsWithUrlsConfig {
	if configData, exists := outputMap["list-project-items-with-urls"]; exists {
		listProjectItemsWithUrlsLog.Print("Parsing list-project-items-with-urls configuration")
		config := &ListProjectItemsWithUrlsConfig{}
		config.Max = 5 // Default max is 5

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 5)

			// Parse github-token override if specified
			if token, exists := configMap["github-token"]; exists {
				if tokenStr, ok := token.(string); ok {
					config.GitHubToken = tokenStr
					listProjectItemsWithUrlsLog.Print("Using custom GitHub token for list-project-items-with-urls")
				}
			}
		}

		listProjectItemsWithUrlsLog.Printf("Parsed list-project-items-with-urls config: max=%d, hasCustomToken=%v",
			config.Max, config.GitHubToken != "")
		return config
	}
	listProjectItemsWithUrlsLog.Print("No list-project-items-with-urls configuration found")
	return nil
}
