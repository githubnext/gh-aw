package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateDiscussionLog = logger.New("workflow:update_discussion")

// UpdateDiscussionsConfig holds configuration for updating GitHub discussions from agent output
type UpdateDiscussionsConfig struct {
	UpdateEntityConfig `yaml:",inline"`
	Title              *bool    `yaml:"title,omitempty"`          // Allow updating discussion title - presence indicates field can be updated
	Body               *bool    `yaml:"body,omitempty"`           // Allow updating discussion body - presence indicates field can be updated
	Labels             *bool    `yaml:"labels,omitempty"`         // Allow updating discussion labels - presence indicates field can be updated
	AllowedLabels      []string `yaml:"allowed-labels,omitempty"` // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
}

// parseUpdateDiscussionsConfig handles update-discussion configuration
func (c *Compiler) parseUpdateDiscussionsConfig(outputMap map[string]any) *UpdateDiscussionsConfig {
	// Parse base configuration using helper
	baseConfig, configMap := c.parseUpdateEntityBase(outputMap, UpdateEntityDiscussion, "update-discussion", updateDiscussionLog)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateDiscussionsConfig with base fields
	updateDiscussionsConfig := &UpdateDiscussionsConfig{
		UpdateEntityConfig: *baseConfig,
	}

	// Parse discussion-specific fields from config map
	if configMap != nil {
		// Parse title - presence of the key (even if nil/empty) indicates field can be updated
		if _, exists := configMap["title"]; exists {
			updateDiscussionsConfig.Title = new(bool)
		}

		// Parse body - presence of the key (even if nil/empty) indicates field can be updated
		if _, exists := configMap["body"]; exists {
			updateDiscussionsConfig.Body = new(bool)
		}

		// Parse labels - presence of the key (even if nil/empty) indicates field can be updated
		if _, exists := configMap["labels"]; exists {
			updateDiscussionsConfig.Labels = new(bool)
		}

		// Parse allowed-labels using shared helper
		updateDiscussionsConfig.AllowedLabels = parseAllowedLabelsFromConfig(configMap)
		if len(updateDiscussionsConfig.AllowedLabels) > 0 {
			updateDiscussionLog.Printf("Allowed labels configured: %v", updateDiscussionsConfig.AllowedLabels)
			// If allowed-labels is specified, implicitly enable labels
			if updateDiscussionsConfig.Labels == nil {
				updateDiscussionsConfig.Labels = new(bool)
			}
		}
	}

	return updateDiscussionsConfig
}
