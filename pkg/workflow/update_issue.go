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
	params := UpdateEntityJobParams{
		EntityType: UpdateEntityIssue,
		ConfigKey:  "update-issue",
	}

	parseSpecificFields := func(configMap map[string]any, baseConfig *UpdateEntityConfig) {
		// This will be called during parsing to handle issue-specific fields
		// The actual UpdateIssuesConfig fields are handled separately since they're not in baseConfig
	}

	baseConfig := c.parseUpdateEntityConfig(outputMap, params, updateIssueLog, parseSpecificFields)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateIssuesConfig and populate it
	updateIssuesConfig := &UpdateIssuesConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse issue-specific fields
	if configData, exists := outputMap["update-issue"]; exists {
		if configMap, ok := configData.(map[string]any); ok {
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
	}

	return updateIssuesConfig
}
