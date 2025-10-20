package workflow

import (
	"fmt"
)

// CreateDiscussionsConfig holds configuration for creating GitHub discussions from agent output
type CreateDiscussionsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string `yaml:"title-prefix,omitempty"`
	Category             string `yaml:"category,omitempty"`    // Discussion category ID or name
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository discussions
}

// parseDiscussionsConfig handles create-discussion configuration
func (c *Compiler) parseDiscussionsConfig(outputMap map[string]any) *CreateDiscussionsConfig {
	if configData, exists := outputMap["create-discussion"]; exists {
		discussionsConfig := &CreateDiscussionsConfig{}
		discussionsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &discussionsConfig.BaseSafeOutputConfig)

			// Parse title-prefix using shared helper
			discussionsConfig.TitlePrefix = parseTitlePrefixFromConfig(configMap)

			// Parse category (can be string or number)
			if category, exists := configMap["category"]; exists {
				switch v := category.(type) {
				case string:
					discussionsConfig.Category = v
				case int:
					discussionsConfig.Category = fmt.Sprintf("%d", v)
				case int64:
					discussionsConfig.Category = fmt.Sprintf("%d", v)
				case float64:
					discussionsConfig.Category = fmt.Sprintf("%.0f", v)
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			// Validate that target-repo is not "*" - only definite strings are allowed
			if targetRepoSlug == "*" {
				return nil // Invalid configuration, return nil to cause validation error
			}
			discussionsConfig.TargetRepoSlug = targetRepoSlug
		}

		return discussionsConfig
	}

	return nil
}

// buildCreateOutputDiscussionJob creates the create_discussion job
func (c *Compiler) buildCreateOutputDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateDiscussions == nil {
		return nil, fmt.Errorf("safe-outputs.create-discussion configuration is required")
	}

	// Build custom environment variables specific to create-discussion
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISCUSSION_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateDiscussions.TitlePrefix))
	}
	if data.SafeOutputs.CreateDiscussions.Category != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISCUSSION_CATEGORY: %q\n", data.SafeOutputs.CreateDiscussions.Category))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CreateDiscussions.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.CreateDiscussions != nil {
		token = data.SafeOutputs.CreateDiscussions.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Create Output Discussion",
		StepID:        "create_discussion",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        createDiscussionScript,
		Token:         token,
	})

	outputs := map[string]string{
		"discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
	}

	jobCondition := BuildSafeOutputType("create_discussion", data.SafeOutputs.CreateDiscussions.Min)

	job := &Job{
		Name:           "create_discussion",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsReadDiscussionsWrite().RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
