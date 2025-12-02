package workflow

import (
	"fmt"
)

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Status                 *bool `yaml:"status,omitempty"` // Allow updating issue status (open/closed) - presence indicates field can be updated
	Title                  *bool `yaml:"title,omitempty"`  // Allow updating issue title - presence indicates field can be updated
	Body                   *bool `yaml:"body,omitempty"`   // Allow updating issue body - presence indicates field can be updated
}

// buildCreateOutputUpdateIssueJob creates the update_issue job
func (c *Compiler) buildCreateOutputUpdateIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.update-issue configuration is required")
	}

	cfg := data.SafeOutputs.UpdateIssues

	// Build custom environment variables specific to update-issue
	customEnvVars := []string{
		fmt.Sprintf("          GH_AW_UPDATE_STATUS: %t\n", cfg.Status != nil),
		fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", cfg.Title != nil),
		fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", cfg.Body != nil),
	}

	// Pass the target configuration
	customEnvVars = append(customEnvVars, BuildTargetEnvVar("GH_AW_UPDATE_TARGET", cfg.Target)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
	}

	// Build job condition with event check if target is not specified
	jobCondition := BuildSafeOutputType("update_issue")
	if cfg.Target == "" {
		eventCondition := BuildPropertyAccess("github.event.issue.number")
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "update_issue",
		StepName:       "Update Issue",
		StepID:         "update_issue",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getUpdateIssueScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}

// parseUpdateIssuesConfig handles update-issue configuration
func (c *Compiler) parseUpdateIssuesConfig(outputMap map[string]any) *UpdateIssuesConfig {
	if configData, exists := outputMap["update-issue"]; exists {
		updateIssuesConfig := &UpdateIssuesConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target config (target, target-repo)
			targetConfig, _ := ParseTargetConfig(configMap)
			updateIssuesConfig.SafeOutputTargetConfig = targetConfig

			// Parse status - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["status"]; exists {
				// If the key exists, it means we can update the status
				// We don't care about the value - just that the key is present
				updateIssuesConfig.Status = new(bool) // Allocate a new bool pointer (defaults to false)
			}

			// Parse title - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["title"]; exists {
				updateIssuesConfig.Title = new(bool)
			}

			// Parse body - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["body"]; exists {
				updateIssuesConfig.Body = new(bool)
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &updateIssuesConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "update-issue:" with no value),
			// still set the default max
			updateIssuesConfig.Max = 1
		}

		return updateIssuesConfig
	}

	return nil
}
