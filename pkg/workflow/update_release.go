package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateReleaseLog = logger.New("workflow:update_release")

// UpdateReleaseConfig holds configuration for updating GitHub releases from agent output
type UpdateReleaseConfig struct {
	UpdateEntityConfig `yaml:",inline"`
}

// parseUpdateReleaseConfig handles update-release configuration
func (c *Compiler) parseUpdateReleaseConfig(outputMap map[string]any) *UpdateReleaseConfig {
	// Create config struct
	cfg := &UpdateReleaseConfig{}

	// Parse base config (no entity-specific fields for releases)
	baseConfig, _ := c.parseUpdateEntityConfigWithFields(outputMap, UpdateEntityParseOptions{
		EntityType: UpdateEntityRelease,
		ConfigKey:  "update-release",
		Logger:     updateReleaseLog,
		Fields:     nil, // No entity-specific fields
	})
	if baseConfig == nil {
		return nil
	}

	// Set base fields
	cfg.UpdateEntityConfig = *baseConfig
	return cfg
}
