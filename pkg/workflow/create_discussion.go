package workflow

import (
	"fmt"
)

// CreateDiscussionsConfig holds configuration for creating GitHub discussions from agent output
type CreateDiscussionsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string `yaml:"title-prefix,omitempty"`
	CategoryId           string `yaml:"category-id,omitempty"` // Discussion category ID
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

	// Prepare base environment variables
	env := make(map[string]string)
	env["GITHUB_AW_AGENT_OUTPUT"] = fmt.Sprintf("${{ needs.%s.outputs.output }}", mainJobName)
	env["GITHUB_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)
	if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
		env["GITHUB_AW_DISCUSSION_TITLE_PREFIX"] = fmt.Sprintf("%q", data.SafeOutputs.CreateDiscussions.TitlePrefix)
	}
	if data.SafeOutputs.CreateDiscussions.CategoryId != "" {
		env["GITHUB_AW_DISCUSSION_CATEGORY_ID"] = fmt.Sprintf("%q", data.SafeOutputs.CreateDiscussions.CategoryId)
	}
	// Pass the staged flag if it's set to true
	if c.trialMode || data.SafeOutputs.Staged {
		env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "\"true\""
	}

	// Add custom environment variables from safe-outputs.env
	for key, value := range c.getCustomSafeOutputEnvVars(data) {
		env[key] = value
	}

	// Prepare with parameters
	withParams := make(map[string]string)
	// Get github-token if specified
	var token string
	if data.SafeOutputs.CreateDiscussions != nil {
		token = data.SafeOutputs.CreateDiscussions.GitHubToken
	}
	if githubToken := c.getSafeOutputGitHubTokenForConfig(data, token); githubToken != "" {
		withParams["github-token"] = githubToken
	}

	// Build the github-script step using the simpler helper
	steps := BuildGitHubScriptStepLines(
		"Create Output Discussion",
		"create_discussion",
		createDiscussionScript,
		env,
		withParams,
	)

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
