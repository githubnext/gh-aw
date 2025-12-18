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
	params := UpdateEntityJobParams{
		EntityType: UpdateEntityRelease,
		ConfigKey:  "update-release",
	}

	baseConfig := c.parseUpdateEntityConfig(outputMap, params, updateReleaseLog, nil)
	if baseConfig == nil {
		return nil
	}

	// Create UpdateReleaseConfig and populate it
	updateReleaseConfig := &UpdateReleaseConfig{
		UpdateEntityConfig: *baseConfig,
	}

	return updateReleaseConfig
}
