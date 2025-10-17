package workflow

import (
	"fmt"
	"strings"
)

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	BaseSafeOutputConfig  `yaml:",inline"`
	TitlePrefix           string   `yaml:"title-prefix,omitempty"`
	Labels                []string `yaml:"labels,omitempty"`
	TargetRepoSlug        string   `yaml:"target-repo,omitempty"`            // Target repository in format "owner/repo" for cross-repository issues
	AssignToBot           string   `yaml:"assign-to-bot,omitempty"`          // Bot username to assign the created issue to (e.g., "copilot-swe-agent")
	AssignToBotGitHubToken string  `yaml:"assign-to-bot-github-token,omitempty"` // GitHub token specifically for bot assignment operations
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	if configData, exists := outputMap["create-issue"]; exists {
		issuesConfig := &CreateIssuesConfig{}
		issuesConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					issuesConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse labels
			if labels, exists := configMap["labels"]; exists {
				if labelsArray, ok := labels.([]any); ok {
					var labelStrings []string
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							labelStrings = append(labelStrings, labelStr)
						}
					}
					issuesConfig.Labels = labelStrings
				}
			}

			// Parse target-repo
			if targetRepoSlug, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepoSlug.(string); ok {
					// Validate that target-repo is not "*" - only definite strings are allowed
					if targetRepoStr == "*" {
						return nil // Invalid configuration, return nil to cause validation error
					}
					issuesConfig.TargetRepoSlug = targetRepoStr
				}
			}

			// Parse assign-to-bot
			if assignToBot, exists := configMap["assign-to-bot"]; exists {
				if assignToBotStr, ok := assignToBot.(string); ok {
					issuesConfig.AssignToBot = assignToBotStr
				}
			}

			// Parse assign-to-bot-github-token
			if assignToBotToken, exists := configMap["assign-to-bot-github-token"]; exists {
				if assignToBotTokenStr, ok := assignToBotToken.(string); ok {
					issuesConfig.AssignToBotGitHubToken = assignToBotTokenStr
				}
			}

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
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))
	// Pass the workflow source URL for installation instructions
	if data.Source != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_SOURCE: %q\n", data.Source))
		sourceURL := buildSourceURL(data.Source)
		if sourceURL != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_SOURCE_URL: %q\n", sourceURL))
		}
	}
	if data.SafeOutputs.CreateIssues.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_ISSUE_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateIssues.TitlePrefix))
	}
	if len(data.SafeOutputs.CreateIssues.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreateIssues.Labels, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_ISSUE_LABELS: %q\n", labelsStr))
	}
	if data.SafeOutputs.CreateIssues.AssignToBot != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_ISSUE_ASSIGN_TO_BOT: %q\n", data.SafeOutputs.CreateIssues.AssignToBot))
	}
	// Note: assign-to-bot-github-token is handled via the github-script action's 'github-token' parameter
	// See token selection logic above where AssignToBotGitHubToken takes precedence

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CreateIssues.TargetRepoSlug,
	)...)

	// Get token from config
	// If assign-to-bot-github-token is specified, use that for the github-script action
	// This allows bot assignment operations to use a different token with appropriate permissions
	var token string
	if data.SafeOutputs.CreateIssues != nil {
		if data.SafeOutputs.CreateIssues.AssignToBotGitHubToken != "" {
			token = data.SafeOutputs.CreateIssues.AssignToBotGitHubToken
		} else {
			token = data.SafeOutputs.CreateIssues.GitHubToken
		}
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

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
	}

	jobCondition := BuildSafeOutputType("create_issue", data.SafeOutputs.CreateIssues.Min)

	// Set base permissions
	permissions := "permissions:\n      contents: read\n      issues: write"

	job := &Job{
		Name:           "create_issue",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions,
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
