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
	// Start building the job with the fluent builder
	builder := c.NewSafeOutputJobBuilder(data, "update_issue").
		WithConfig(data.SafeOutputs == nil || data.SafeOutputs.UpdateIssues == nil).
		WithStepMetadata("Update Issue", "update_issue").
		WithMainJobName(mainJobName).
		WithScript(getUpdateIssueScript()).
		WithPermissions(NewPermissionsContentsReadIssuesWrite()).
		WithOutputs(map[string]string{
			"issue_number": "${{ steps.update_issue.outputs.issue_number }}",
			"issue_url":    "${{ steps.update_issue.outputs.issue_url }}",
		})

	// Add job-specific environment variables
	if data.SafeOutputs != nil && data.SafeOutputs.UpdateIssues != nil {
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_UPDATE_STATUS: %t\n", data.SafeOutputs.UpdateIssues.Status != nil))
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_UPDATE_TITLE: %t\n", data.SafeOutputs.UpdateIssues.Title != nil))
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_UPDATE_BODY: %t\n", data.SafeOutputs.UpdateIssues.Body != nil))

		if data.SafeOutputs.UpdateIssues.Target != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdateIssues.Target))
		}

		// Set token and target repo
		builder.WithToken(data.SafeOutputs.UpdateIssues.GitHubToken).
			WithTargetRepoSlug(data.SafeOutputs.UpdateIssues.TargetRepoSlug)

		// Build job condition with event check if target is not specified
		jobCondition := BuildSafeOutputType("update_issue")
		if data.SafeOutputs.UpdateIssues.Target == "" {
			eventCondition := BuildPropertyAccess("github.event.issue.number")
			jobCondition = buildAnd(jobCondition, eventCondition)
		}
		builder.WithCondition(jobCondition)
	}

	return builder.Build()
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
