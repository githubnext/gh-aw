package workflow

import (
	"fmt"
	"strings"
)

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`
	Labels               []string `yaml:"labels,omitempty"`
	Assignees            []string `yaml:"assignees,omitempty"`   // List of users/bots to assign the issue to
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository issues
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	if configData, exists := outputMap["create-issue"]; exists {
		issuesConfig := &CreateIssuesConfig{}
		issuesConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix using shared helper
			issuesConfig.TitlePrefix = parseTitlePrefixFromConfig(configMap)

			// Parse labels using shared helper
			issuesConfig.Labels = parseLabelsFromConfig(configMap)

			// Parse assignees (supports both string and array)
			if assignees, exists := configMap["assignees"]; exists {
				if assigneeStr, ok := assignees.(string); ok {
					// Single string format
					issuesConfig.Assignees = []string{assigneeStr}
				} else if assigneesArray, ok := assignees.([]any); ok {
					// Array format
					var assigneeStrings []string
					for _, assignee := range assigneesArray {
						if assigneeStr, ok := assignee.(string); ok {
							assigneeStrings = append(assigneeStrings, assigneeStr)
						}
					}
					issuesConfig.Assignees = assigneeStrings
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			// Validate that target-repo is not "*" - only definite strings are allowed
			if targetRepoSlug == "*" {
				return nil // Invalid configuration, return nil to cause validation error
			}
			issuesConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &issuesConfig.BaseSafeOutputConfig)
		}

		return issuesConfig
	}

	return nil
}

// buildCreateOutputIssueJob creates the create_issue job
func (c *Compiler) buildCreateOutputIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	// Start building the job with the fluent builder
	builder := c.NewSafeOutputJobBuilder(data, "create_issue").
		WithConfig(data.SafeOutputs == nil || data.SafeOutputs.CreateIssues == nil).
		WithStepMetadata("Create Output Issue", "create_issue").
		WithMainJobName(mainJobName).
		WithScript(getCreateIssueScript()).
		WithPermissions(NewPermissionsContentsReadIssuesWrite()).
		WithOutputs(map[string]string{
			"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
			"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
		})

	// Add job-specific environment variables
	builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

	if data.Source != "" {
		builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE: %q\n", data.Source))
		sourceURL := buildSourceURL(data.Source)
		if sourceURL != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE_URL: %q\n", sourceURL))
		}
	}

	if data.SafeOutputs != nil && data.SafeOutputs.CreateIssues != nil {
		if data.SafeOutputs.CreateIssues.TitlePrefix != "" {
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_ISSUE_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateIssues.TitlePrefix))
		}
		if len(data.SafeOutputs.CreateIssues.Labels) > 0 {
			labelsStr := strings.Join(data.SafeOutputs.CreateIssues.Labels, ",")
			builder.AddEnvVar(fmt.Sprintf("          GH_AW_ISSUE_LABELS: %q\n", labelsStr))
		}

		// Set token and target repo
		builder.WithToken(data.SafeOutputs.CreateIssues.GitHubToken).
			WithTargetRepoSlug(data.SafeOutputs.CreateIssues.TargetRepoSlug)

		// Build post-steps for assignees if configured
		if len(data.SafeOutputs.CreateIssues.Assignees) > 0 {
			var safeOutputsToken string
			if data.SafeOutputs != nil {
				safeOutputsToken = data.SafeOutputs.GitHubToken
			}

			postSteps := buildCopilotParticipantSteps(CopilotParticipantConfig{
				Participants:       data.SafeOutputs.CreateIssues.Assignees,
				ParticipantType:    "assignee",
				CustomToken:        data.SafeOutputs.CreateIssues.GitHubToken,
				SafeOutputsToken:   safeOutputsToken,
				WorkflowToken:      data.GitHubToken,
				ConditionStepID:    "create_issue",
				ConditionOutputKey: "issue_number",
			})
			builder.WithPostSteps(postSteps)
		}
	}

	return builder.Build()
}
