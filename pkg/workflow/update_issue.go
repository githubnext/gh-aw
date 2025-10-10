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
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_UPDATE_STATUS: %t\n", data.SafeOutputs.UpdateIssues.Status != nil))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_UPDATE_TITLE: %t\n", data.SafeOutputs.UpdateIssues.Title != nil))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_UPDATE_BODY: %t\n", data.SafeOutputs.UpdateIssues.Body != nil))

	if data.SafeOutputs.UpdateIssues.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdateIssues.Target))
	}
	if c.trialMode || data.SafeOutputs.Staged {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Pass target repository - prefer explicit config over trial mode setting
	if data.SafeOutputs.UpdateIssues.TargetRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", data.SafeOutputs.UpdateIssues.TargetRepoSlug))
	} else if c.trialMode && c.trialLogicalRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", c.trialLogicalRepoSlug))
	}

	// Get token from config
	var token string
	if data.SafeOutputs.UpdateIssues != nil {
		token = data.SafeOutputs.UpdateIssues.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Update Issue",
		StepID:        "update_issue",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        updateIssueScript,
		Token:         token,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
	}

	var jobCondition = BuildSafeOutputType("update-issue", data.SafeOutputs.UpdateIssues.Min)
	if data.SafeOutputs.UpdateIssues != nil && data.SafeOutputs.UpdateIssues.Target == "" {
		eventCondition := BuildPropertyAccess("github.event.issue.number")
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "update_issue",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      issues: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
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
