package workflow

import (
	"fmt"
)

// CreateDiscussionsConfig holds configuration for creating GitHub discussions from agent output
type CreateDiscussionsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string `yaml:"title-prefix,omitempty"`
	CategoryId           string `yaml:"category-id,omitempty"` // Discussion category ID
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

			// Parse title-prefix
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					discussionsConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse category-id
			if categoryId, exists := configMap["category-id"]; exists {
				if categoryIdStr, ok := categoryId.(string); ok {
					discussionsConfig.CategoryId = categoryIdStr
				}
			}

			// Parse target-repo
			if targetRepoSlug, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepoSlug.(string); ok {
					// Validate that target-repo is not "*" - only definite strings are allowed
					if targetRepoStr == "*" {
						return nil // Invalid configuration, return nil to cause validation error
					}
					discussionsConfig.TargetRepoSlug = targetRepoStr
				}
			}
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
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))
	if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_DISCUSSION_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateDiscussions.TitlePrefix))
	}
	if data.SafeOutputs.CreateDiscussions.CategoryId != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_DISCUSSION_CATEGORY_ID: %q\n", data.SafeOutputs.CreateDiscussions.CategoryId))
	}

	// Pass the staged flag if it's set to true
	if c.trialMode || data.SafeOutputs.Staged {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}
	// Set GITHUB_AW_TARGET_REPO_SLUG - prefer target-repo config over trial target repo
	if data.SafeOutputs.CreateDiscussions.TargetRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", data.SafeOutputs.CreateDiscussions.TargetRepoSlug))
	} else if c.trialMode && c.trialSimulatedRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", c.trialSimulatedRepoSlug))
	}

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

	jobCondition := BuildSafeOutputType("create-discussion", data.SafeOutputs.CreateDiscussions.Min)

	job := &Job{
		Name:           "create_discussion",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      discussions: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
