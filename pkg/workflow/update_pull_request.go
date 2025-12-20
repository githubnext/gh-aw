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

	// Parse PR-specific fields using bool value mode (defaults to true if nil)
	updatePullRequestsConfig.Title = parseUpdateEntityBoolField(configMap, "title", FieldParsingBoolValue)
	updatePullRequestsConfig.Body = parseUpdateEntityBoolField(configMap, "body", FieldParsingBoolValue)

	return updatePullRequestsConfig
}
