package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var createIssueLog = logger.New("workflow:create_issue")

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`
	Labels               []string `yaml:"labels,omitempty"`
	Assignees            []string `yaml:"assignees,omitempty"`     // List of users/bots to assign the issue to
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository issues
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // List of additional repositories that issues can be created in
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	if configData, exists := outputMap["create-issue"]; exists {
		createIssueLog.Print("Parsing create-issue configuration")
		issuesConfig := &CreateIssuesConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix using shared helper
			issuesConfig.TitlePrefix = parseTitlePrefixFromConfig(configMap)

			// Parse labels using shared helper
			issuesConfig.Labels = parseLabelsFromConfig(configMap)

			// Parse assignees using shared helper
			issuesConfig.Assignees = parseParticipantsFromConfig(configMap, "assignees")

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			issuesConfig.TargetRepoSlug = targetRepoSlug

			// Parse allowed-repos using shared helper
			issuesConfig.AllowedRepos = parseAllowedReposFromConfig(configMap)

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &issuesConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "create-issue:" with no value),
			// still set the default max
			issuesConfig.Max = 1
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

	if createIssueLog.Enabled() {
		createIssueLog.Printf("Building create-issue job: workflow=%s, main_job=%s, assignees=%d, labels=%d",
			data.Name, mainJobName, len(data.SafeOutputs.CreateIssues.Assignees), len(data.SafeOutputs.CreateIssues.Labels))
	}

	// Build custom environment variables specific to create-issue using shared helpers
	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", data.SafeOutputs.CreateIssues.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", data.SafeOutputs.CreateIssues.Labels)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", data.SafeOutputs.CreateIssues.AllowedRepos)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CreateIssues.TargetRepoSlug)...)

	// Build post-steps for assignees if configured
	var postSteps []string
	if len(data.SafeOutputs.CreateIssues.Assignees) > 0 {
		// Get the effective GitHub token to use for gh CLI
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}

		postSteps = buildCopilotParticipantSteps(CopilotParticipantConfig{
			Participants:       data.SafeOutputs.CreateIssues.Assignees,
			ParticipantType:    "assignee",
			CustomToken:        data.SafeOutputs.CreateIssues.GitHubToken,
			SafeOutputsToken:   safeOutputsToken,
			WorkflowToken:      data.GitHubToken,
			ConditionStepID:    "create_issue",
			ConditionOutputKey: "issue_number",
		})
	}

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number":     "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":        "${{ steps.create_issue.outputs.issue_url }}",
		"temporary_id_map": "${{ steps.create_issue.outputs.temporary_id_map }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "create_issue",
		StepName:       "Create Output Issue",
		StepID:         "create_issue",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCreateIssueScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		PostSteps:      postSteps,
		Token:          data.SafeOutputs.CreateIssues.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CreateIssues.TargetRepoSlug,
	})
}
