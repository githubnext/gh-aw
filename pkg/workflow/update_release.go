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

	// Build custom environment variables specific to update-release
	// Uses buildStandardSafeOutputEnvVars for consistency with other update jobs
	var customEnvVars []string

	// Create outputs for the job
	outputs := map[string]string{
		"release_id":  "${{ steps.update_release.outputs.release_id }}",
		"release_url": "${{ steps.update_release.outputs.release_url }}",
		"release_tag": "${{ steps.update_release.outputs.release_tag }}",
	}

	// Build job condition - update_release doesn't have event context checks
	jobCondition := BuildSafeOutputType("update_release")

	params := UpdateEntityJobParams{
		EntityType:      UpdateEntityRelease,
		ConfigKey:       "update-release",
		JobName:         "update_release",
		StepName:        "Update Release",
		ScriptGetter:    getUpdateReleaseScript,
		PermissionsFunc: NewPermissionsContentsWrite,
		CustomEnvVars:   customEnvVars,
		Outputs:         outputs,
		Condition:       jobCondition,
	}

	return c.buildUpdateEntityJob(data, mainJobName, &cfg.UpdateEntityConfig, params, updateReleaseLog)
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
