package workflow

import (
	"fmt"
)

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Status               *bool  `yaml:"status,omitempty"`      // Allow updating issue status (open/closed) - presence indicates field can be updated
	Target               string `yaml:"target,omitempty"`      // Target for updates: "triggering" (default), "*" (any issue), or explicit issue number
	Title                *bool  `yaml:"title,omitempty"`       // Allow updating issue title - presence indicates field can be updated
	Body                 *bool  `yaml:"body,omitempty"`        // Allow updating issue body - presence indicates field can be updated
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository issue updates
}

// buildCreateOutputUpdateIssueJob creates the update_issue job
func (c *Compiler) buildCreateOutputUpdateIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.update-issue configuration is required")
	}

	// Build custom environment variables specific to update-issue
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_STATUS: %t\n", data.SafeOutputs.UpdateIssues.Status != nil))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", data.SafeOutputs.UpdateIssues.Title != nil))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", data.SafeOutputs.UpdateIssues.Body != nil))

	if data.SafeOutputs.UpdateIssues.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdateIssues.Target))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.UpdateIssues.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.UpdateIssues != nil {
		token = data.SafeOutputs.UpdateIssues.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
	}

	// Build job condition with event check if target is not specified
	jobCondition := BuildSafeOutputType("update_issue")
	if data.SafeOutputs.UpdateIssues != nil && data.SafeOutputs.UpdateIssues.Target == "" {
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
		Token:          token,
		TargetRepoSlug: data.SafeOutputs.UpdateIssues.TargetRepoSlug,
	})
}

// parseUpdateIssuesConfig handles update-issue configuration
func (c *Compiler) parseUpdateIssuesConfig(outputMap map[string]any) *UpdateIssuesConfig {
	if configData, exists := outputMap["update-issue"]; exists {
		updateIssuesConfig := &UpdateIssuesConfig{}
		updateIssuesConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &updateIssuesConfig.BaseSafeOutputConfig)

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					updateIssuesConfig.Target = targetStr
				}
			}

			// Parse target-repo
			if targetRepo, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepo.(string); ok {
					updateIssuesConfig.TargetRepoSlug = targetRepoStr
				}
			}

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
		}

		return updateIssuesConfig
	}

	return nil
}
