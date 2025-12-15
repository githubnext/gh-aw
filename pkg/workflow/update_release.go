package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateReleaseLog = logger.New("workflow:update_release")

// UpdateReleaseConfig holds configuration for updating GitHub releases from agent output
type UpdateReleaseConfig struct {
	UpdateEntityConfig `yaml:",inline"`
}

// buildCreateOutputUpdateReleaseJob creates the update_release job using the shared builder
func (c *Compiler) buildCreateOutputUpdateReleaseJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateRelease == nil {
		return nil, fmt.Errorf("safe-outputs.update-release configuration is required")
	}

	cfg := data.SafeOutputs.UpdateRelease

	return c.buildStandardUpdateEntityJob(
		data,
		mainJobName,
		&cfg.UpdateEntityConfig,
		UpdateEntityRelease,
		"update-release",
		"update_release",
		"Update Release",
		getUpdateReleaseScript,
		NewPermissionsContentsWrite,
		func(config *UpdateEntityConfig) []string {
			// Update-release doesn't have entity-specific env vars like status/title/body
			return []string{}
		},
		func() map[string]string {
			return map[string]string{
				"release_id":  "${{ steps.update_release.outputs.release_id }}",
				"release_url": "${{ steps.update_release.outputs.release_url }}",
				"release_tag": "${{ steps.update_release.outputs.release_tag }}",
			}
		},
		nil, // BuildEventCondition is nil - update_release doesn't have event context checks
		updateReleaseLog,
	)
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
