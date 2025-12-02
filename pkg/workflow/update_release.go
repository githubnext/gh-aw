package workflow

import (
	"fmt"
)

// UpdateReleaseConfig holds configuration for updating GitHub releases from agent output
type UpdateReleaseConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
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

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"release_id":  "${{ steps.update_release.outputs.release_id }}",
		"release_url": "${{ steps.update_release.outputs.release_url }}",
		"release_tag": "${{ steps.update_release.outputs.release_tag }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "update_release",
		StepName:       "Update Release",
		StepID:         "update_release",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getUpdateReleaseScript(),
		Permissions:    NewPermissionsContentsWrite(),
		Outputs:        outputs,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// parseUpdateReleaseConfig handles update-release configuration
func (c *Compiler) parseUpdateReleaseConfig(outputMap map[string]any) *UpdateReleaseConfig {
	if configData, exists := outputMap["update-release"]; exists {
		updateReleaseConfig := &UpdateReleaseConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target config (target-repo) with validation
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				return nil // Invalid configuration (e.g., wildcard target-repo), return nil to cause validation error
			}
			updateReleaseConfig.SafeOutputTargetConfig = targetConfig

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &updateReleaseConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map, still set the default max
			updateReleaseConfig.Max = 1
		}

		return updateReleaseConfig
	}

	return nil
}
