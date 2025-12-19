package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updatePullRequestLog = logger.New("workflow:update_pull_request")

// UpdatePullRequestsConfig holds configuration for updating GitHub pull requests from agent output
type UpdatePullRequestsConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Title              *bool `yaml:"title,omitempty"` // Allow updating PR title - defaults to true, set to false to disable
	Body               *bool `yaml:"body,omitempty"`  // Allow updating PR body - defaults to true, set to false to disable
}

// parseUpdatePullRequestsConfig handles update-pull-request configuration
func (c *Compiler) parseUpdatePullRequestsConfig(outputMap map[string]any) *UpdatePullRequestsConfig {
	updatePullRequestLog.Print("Parsing update pull request configuration")

	// Parse base configuration using helper
	baseConfig, configMap := c.parseUpdateEntityBase(outputMap, UpdateEntityPullRequest, "update-pull-request", updatePullRequestLog)
	if baseConfig == nil {
		return nil
	}

	// Create UpdatePullRequestsConfig with base fields
	updatePullRequestsConfig := &UpdatePullRequestsConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse PR-specific fields from config map
	if configMap != nil {
		// Parse title - boolean to enable/disable (defaults to true if nil or not set)
		if titleVal, exists := configMap["title"]; exists {
			if titleBool, ok := titleVal.(bool); ok {
				updatePullRequestsConfig.Title = &titleBool
			}
			// If present but not a bool (e.g., null), leave as nil (defaults to enabled)
		}

		// Parse body - boolean to enable/disable (defaults to true if nil or not set)
		if bodyVal, exists := configMap["body"]; exists {
			if bodyBool, ok := bodyVal.(bool); ok {
				updatePullRequestsConfig.Body = &bodyBool
			}
			// If present but not a bool (e.g., null), leave as nil (defaults to enabled)
		}
	}

	return updatePullRequestsConfig
}
