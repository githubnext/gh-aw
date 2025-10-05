package workflow

import (
	"fmt"
)

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Status               *bool  `yaml:"status,omitempty"` // Allow updating issue status (open/closed) - presence indicates field can be updated
	Target               string `yaml:"target,omitempty"` // Target for updates: "triggering" (default), "*" (any issue), or explicit issue number
	Title                *bool  `yaml:"title,omitempty"`  // Allow updating issue title - presence indicates field can be updated
	Body                 *bool  `yaml:"body,omitempty"`   // Allow updating issue body - presence indicates field can be updated
}

// buildCreateOutputUpdateIssueJob creates the update_issue job
func (c *Compiler) buildCreateOutputUpdateIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.update-issue configuration is required")
	}

	var steps []string

	// Build environment variables
	env := make(map[string]string)

	// Build with parameters
	withParams := make(map[string]string)
	token := ""
	if data.SafeOutputs.UpdateIssues != nil {
		token = data.SafeOutputs.UpdateIssues.GitHubToken
	}

	envConfig := &SafeOutputEnvConfig{
		TargetValue:   data.SafeOutputs.UpdateIssues.Target,
		TargetEnvName: "GITHUB_AW_UPDATE_TARGET",
		GitHubToken:   token,
	}
	c.getCustomSafeOutputEnvVars(env, data, mainJobName, envConfig, withParams)

	// Pass the configuration flags
	env["GITHUB_AW_UPDATE_STATUS"] = fmt.Sprintf("%t", data.SafeOutputs.UpdateIssues.Status != nil)
	env["GITHUB_AW_UPDATE_TITLE"] = fmt.Sprintf("%t", data.SafeOutputs.UpdateIssues.Title != nil)
	env["GITHUB_AW_UPDATE_BODY"] = fmt.Sprintf("%t", data.SafeOutputs.UpdateIssues.Body != nil)

	// Build github-script step
	stepLines := BuildGitHubScriptStepLines("Update Issue", "update_issue", updateIssueScript, env, withParams)
	steps = append(steps, stepLines...)

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
