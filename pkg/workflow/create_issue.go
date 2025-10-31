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
	if data.SafeOutputs == nil || data.SafeOutputs.CreateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.create-issue configuration is required")
	}

	var steps []string

	// Permission checks are now handled by the separate check_membership job
	// which is always created when needed (when activation job is created)

	// Build custom environment variables specific to create-issue
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	// Pass the workflow source URL for installation instructions
	if data.Source != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE: %q\n", data.Source))
		sourceURL := buildSourceURL(data.Source)
		if sourceURL != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE_URL: %q\n", sourceURL))
		}
	}
	if data.SafeOutputs.CreateIssues.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ISSUE_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateIssues.TitlePrefix))
	}
	if len(data.SafeOutputs.CreateIssues.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreateIssues.Labels, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ISSUE_LABELS: %q\n", labelsStr))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CreateIssues.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.CreateIssues != nil {
		token = data.SafeOutputs.CreateIssues.GitHubToken
	}

	// Build the GitHub Script step using the common helper and append to existing steps
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Create Output Issue",
		StepID:        "create_issue",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        createIssueScript,
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	// Add assignee steps if assignees are configured
	if len(data.SafeOutputs.CreateIssues.Assignees) > 0 {
		// Get the effective GitHub token to use for gh CLI
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}

		assigneeSteps := buildCopilotParticipantSteps(CopilotParticipantConfig{
			Participants:       data.SafeOutputs.CreateIssues.Assignees,
			ParticipantType:    "assignee",
			CustomToken:        token,
			SafeOutputsToken:   safeOutputsToken,
			WorkflowToken:      data.GitHubToken,
			ConditionStepID:    "create_issue",
			ConditionOutputKey: "issue_number",
		})
		steps = append(steps, assigneeSteps...)
	}

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
	}

	jobCondition := BuildSafeOutputType("create_issue")

	job := &Job{
		Name:           "create_issue",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadIssuesWrite().RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
