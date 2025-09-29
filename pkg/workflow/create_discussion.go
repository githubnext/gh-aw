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

	var steps []string
	steps = append(steps, "      - name: Create Output Discussion\n")
	steps = append(steps, "        id: create_discussion\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_DISCUSSION_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateDiscussions.TitlePrefix))
	}
	if data.SafeOutputs.CreateDiscussions.CategoryId != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_DISCUSSION_CATEGORY_ID: %q\n", data.SafeOutputs.CreateDiscussions.CategoryId))
	}

	// Pass the staged flag if it's set to true
	if data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.CreateDiscussions != nil {
		token = data.SafeOutputs.CreateDiscussions.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createDiscussionScript)
	steps = append(steps, formattedScript...)

	outputs := map[string]string{
		"discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
	}

	// Build the job condition using expression tree
	jobCondition := BuildSafeOutputType("create-discussion")

	job := &Job{
		Name:           "create_discussion",
		If:             jobCondition.Render(),
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      discussions: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
