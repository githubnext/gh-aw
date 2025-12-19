package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateIssueLog = logger.New("workflow:update_issue")

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Status             *bool `yaml:"status,omitempty"` // Allow updating issue status (open/closed) - presence indicates field can be updated
	Title              *bool `yaml:"title,omitempty"`  // Allow updating issue title - presence indicates field can be updated
	Body               *bool `yaml:"body,omitempty"`   // Allow updating issue body - presence indicates field can be updated
}

// parseUpdateIssuesConfig handles update-issue configuration
func (c *Compiler) parseUpdateIssuesConfig(outputMap map[string]any) *UpdateIssuesConfig {
	// Parse base configuration using helper
	baseConfig, configMap := c.parseUpdateEntityBase(outputMap, UpdateEntityIssue, "update-issue", updateIssueLog)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateIssuesConfig with base fields
	updateIssuesConfig := &UpdateIssuesConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse issue-specific fields from config map
	if configMap != nil {
		// Parse status - presence of the key (even if nil/empty) indicates field can be updated
		if _, exists := configMap["status"]; exists {
			// If the key exists, it means we can update the status
			// We don't care about the value - just that the key is present
			updateIssuesConfig.Status = new(bool) // Allocate a new bool pointer (defaults to false)
		}

		// Parse title - presence of the key (even if nil/empty) indicates field can be updated
		if _, exists := configMap["title"]; exists {
			updateIssuesConfig.Title = new(bool)
		}

		// Parse body - presence of the key (even if nil/empty) indicates field can be updated
		if _, exists := configMap["body"]; exists {
			updateIssuesConfig.Body = new(bool)
		}
	}

	return updateIssuesConfig
}
