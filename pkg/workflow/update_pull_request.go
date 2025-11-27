package workflow

import (
	"fmt"
)

// UpdatePullRequestsConfig holds configuration for updating GitHub pull requests from agent output
type UpdatePullRequestsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string `yaml:"target,omitempty"`      // Target for updates: "triggering" (default), "*" (any PR), or explicit PR number
	Title                *bool  `yaml:"title,omitempty"`       // Allow updating PR title - presence indicates field can be updated
	Body                 *bool  `yaml:"body,omitempty"`        // Allow updating PR body - presence indicates field can be updated
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository PR updates
}

// buildCreateOutputUpdatePullRequestJob creates the update_pull_request job
func (c *Compiler) buildCreateOutputUpdatePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdatePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.update-pull-request configuration is required")
	}

	// Build custom environment variables specific to update-pull-request
	customEnvVars := []string{
		fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", data.SafeOutputs.UpdatePullRequests.Title != nil),
		fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", data.SafeOutputs.UpdatePullRequests.Body != nil),
	}

	if data.SafeOutputs.UpdatePullRequests.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdatePullRequests.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.UpdatePullRequests.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.update_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.update_pull_request.outputs.pull_request_url }}",
	}

	// Build job condition with event check if target is not specified
	jobCondition := BuildSafeOutputType("update_pull_request")
	if data.SafeOutputs.UpdatePullRequests != nil && data.SafeOutputs.UpdatePullRequests.Target == "" {
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
		Token:          data.SafeOutputs.UpdatePullRequests.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.UpdatePullRequests.TargetRepoSlug,
	})
}

// parseUpdatePullRequestsConfig handles update-pull-request configuration
func (c *Compiler) parseUpdatePullRequestsConfig(outputMap map[string]any) *UpdatePullRequestsConfig {
	if configData, exists := outputMap["update-pull-request"]; exists {
		updatePullRequestsConfig := &UpdatePullRequestsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					updatePullRequestsConfig.Target = targetStr
				}
			}

			// Parse target-repo
			if targetRepo, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepo.(string); ok {
					updatePullRequestsConfig.TargetRepoSlug = targetRepoStr
				}
			}

			// Parse title - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["title"]; exists {
				updatePullRequestsConfig.Title = new(bool)
			}

			// Parse body - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["body"]; exists {
				updatePullRequestsConfig.Body = new(bool)
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
