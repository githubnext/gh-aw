package workflow

import (
	"fmt"
)

// UpdatePullRequestsConfig holds configuration for updating GitHub pull requests from agent output
type UpdatePullRequestsConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Title                  *bool `yaml:"title,omitempty"` // Allow updating PR title - defaults to true, set to false to disable
	Body                   *bool `yaml:"body,omitempty"`  // Allow updating PR body - defaults to true, set to false to disable
}

// buildCreateOutputUpdatePullRequestJob creates the update_pull_request job
func (c *Compiler) buildCreateOutputUpdatePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdatePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.update-pull-request configuration is required")
	}

	cfg := data.SafeOutputs.UpdatePullRequests

	// Default to true for both title and body unless explicitly set to false
	canUpdateTitle := cfg.Title == nil || *cfg.Title
	canUpdateBody := cfg.Body == nil || *cfg.Body

	// Build custom environment variables specific to update-pull-request
	customEnvVars := []string{
		fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", canUpdateTitle),
		fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", canUpdateBody),
	}

	// Pass the target configuration
	customEnvVars = append(customEnvVars, BuildTargetEnvVar("GH_AW_UPDATE_TARGET", cfg.Target)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.update_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.update_pull_request.outputs.pull_request_url }}",
	}

	// Build job condition with event check if target is not specified
	jobCondition := BuildSafeOutputType("update_pull_request")
	if cfg.Target == "" {
		eventCondition := BuildPropertyAccess("github.event.pull_request.number")
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "update_pull_request",
		StepName:       "Update Pull Request",
		StepID:         "update_pull_request",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getUpdatePullRequestScript(),
		Permissions:    NewPermissionsContentsReadPRWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// parseUpdatePullRequestsConfig handles update-pull-request configuration
func (c *Compiler) parseUpdatePullRequestsConfig(outputMap map[string]any) *UpdatePullRequestsConfig {
	if configData, exists := outputMap["update-pull-request"]; exists {
		updatePullRequestsConfig := &UpdatePullRequestsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target config (target, target-repo)
			targetConfig, _ := ParseTargetConfig(configMap)
			updatePullRequestsConfig.SafeOutputTargetConfig = targetConfig

			// Parse title - boolean to enable/disable (defaults to true if nil or not set)
			if titleVal, exists := configMap["title"]; exists {
				if titleBool, ok := titleVal.(bool); ok {
					updatePullRequestsConfig.Title = &titleBool
				}
				// If present but not a bool (e.g., null), leave as nil (defaults to enabled)
			}

			// Parse body - boolean to enable/disable (defaults to true if nil or not set)
			if bodyVal, exists := configMap["body"]; exists {
				if bodyBool, ok := bodyVal.(bool); ok {
					updatePullRequestsConfig.Body = &bodyBool
				}
				// If present but not a bool (e.g., null), leave as nil (defaults to enabled)
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &updatePullRequestsConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "update-pull-request:" with no value),
			// still set the default max
			updatePullRequestsConfig.Max = 1
		}

		return updatePullRequestsConfig
	}

	return nil
}
