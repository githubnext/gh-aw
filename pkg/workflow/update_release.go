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
	// Parse base configuration using helper
	baseConfig, _ := c.parseUpdateEntityBase(outputMap, UpdateEntityRelease, "update-release", updateReleaseLog)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateReleaseConfig with base fields (no entity-specific fields for releases)
	updateReleaseConfig := &UpdateReleaseConfig{
		UpdateEntityConfig: *baseConfig,
	}

	return updateReleaseConfig
}
