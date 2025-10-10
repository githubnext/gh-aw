package workflow

import (
	"fmt"
)

// AddCommentConfig holds configuration for creating GitHub issue/PR comments from agent output (deprecated, use AddCommentsConfig)
type AddCommentConfig struct {
	// Empty struct for now, as per requirements, but structured for future expansion
}

// AddCommentsConfig holds configuration for creating GitHub issue/PR comments from agent output
type AddCommentsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string `yaml:"target,omitempty"`      // Target for comments: "triggering" (default), "*" (any issue), or explicit issue number
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository comments
}

// buildCreateOutputAddCommentJob creates the add_comment job
func (c *Compiler) buildCreateOutputAddCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil {
		return nil, fmt.Errorf("safe-outputs.add-comment configuration is required")
	}

	var steps []string
	// Add debug step to echo the output values using environment variables to prevent shell injection
	steps = append(steps, "      - name: Debug agent outputs\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	steps = append(steps, fmt.Sprintf("          AGENT_OUTPUT_TYPES: ${{ needs.%s.outputs.output_types }}\n", mainJobName))
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Output: $AGENT_OUTPUT\"\n")
	steps = append(steps, "          echo \"Output types: $AGENT_OUTPUT_TYPES\"\n")

	// Build custom environment variables specific to add-comment
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
	// Pass the comment target configuration
	if data.SafeOutputs.AddComments.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_COMMENT_TARGET: %q\n", data.SafeOutputs.AddComments.Target))
	}
	if c.trialMode || data.SafeOutputs.Staged {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}
	// Set GITHUB_AW_TARGET_REPO_SLUG - prefer target-repo config over trial target repo
	if data.SafeOutputs.AddComments.TargetRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", data.SafeOutputs.AddComments.TargetRepoSlug))
	} else if c.trialMode && c.trialSimulatedRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", c.trialSimulatedRepoSlug))
	}

	// Get token from config
	var token string
	if data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Build the GitHub Script step using the common helper and append to existing steps
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Add Issue Comment",
		StepID:        "add_comment",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        createCommentScript,
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id":  "${{ steps.add_comment.outputs.comment_id }}",
		"comment_url": "${{ steps.add_comment.outputs.comment_url }}",
	}

	var jobCondition = BuildSafeOutputType("add-comment", data.SafeOutputs.AddComments.Min)
	if data.SafeOutputs.AddComments != nil && data.SafeOutputs.AddComments.Target == "" {
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.issue.number"),
			BuildPropertyAccess("github.event.pull_request.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "add_comment",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseCommentsConfig handles add-comment configuration
func (c *Compiler) parseCommentsConfig(outputMap map[string]any) *AddCommentsConfig {
	if configData, exists := outputMap["add-comment"]; exists {
		commentsConfig := &AddCommentsConfig{}
		commentsConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &commentsConfig.BaseSafeOutputConfig)

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					commentsConfig.Target = targetStr
				}
			}

			// Parse target-repo
			if targetRepoSlug, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepoSlug.(string); ok {
					// Validate that target-repo is not "*" - only definite strings are allowed
					if targetRepoStr == "*" {
						return nil // Invalid configuration, return nil to cause validation error
					}
					commentsConfig.TargetRepoSlug = targetRepoStr
				}
			}
		}

		return commentsConfig
	}

	return nil
}
