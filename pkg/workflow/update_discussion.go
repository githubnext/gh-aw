package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateDiscussionLog = logger.New("workflow:update_discussion")

// UpdateDiscussionsConfig holds configuration for updating GitHub discussions from agent output
type UpdateDiscussionsConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Title              *bool `yaml:"title,omitempty"` // Allow updating discussion title - presence indicates field can be updated
	Body               *bool `yaml:"body,omitempty"`  // Allow updating discussion body - presence indicates field can be updated
}

// parseUpdateDiscussionsConfig handles update-discussion configuration
func (c *Compiler) parseUpdateDiscussionsConfig(outputMap map[string]any) *UpdateDiscussionsConfig {
	params := UpdateEntityJobParams{
		EntityType: UpdateEntityDiscussion,
		ConfigKey:  "update-discussion",
	}

	parseSpecificFields := func(configMap map[string]any, baseConfig *UpdateEntityConfig) {
		// This will be called during parsing to handle discussion-specific fields
		// The actual UpdateDiscussionsConfig fields are handled separately since they're not in baseConfig
	}

	baseConfig := c.parseUpdateEntityConfig(outputMap, params, updateDiscussionLog, parseSpecificFields)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateDiscussionsConfig and populate it
	updateDiscussionsConfig := &UpdateDiscussionsConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse discussion-specific fields
	if configData, exists := outputMap["update-discussion"]; exists {
		if configMap, ok := configData.(map[string]any); ok {
			// Parse title - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["title"]; exists {
				updateDiscussionsConfig.Title = new(bool)
			}

			// Parse body - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["body"]; exists {
				updateDiscussionsConfig.Body = new(bool)
			}
		}
	}

	return updateDiscussionsConfig
}
