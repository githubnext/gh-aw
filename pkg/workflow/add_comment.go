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
	Target               string `yaml:"target,omitempty"` // Target for comments: "triggering" (default), "*" (any issue), or explicit issue number
}

// buildCreateOutputAddCommentJob creates the add_comment job
func (c *Compiler) buildCreateOutputAddCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil {
		return nil, fmt.Errorf("safe-outputs.add-comment configuration is required")
	}

	// Prepare base environment variables
	env := make(map[string]string)
	env["GITHUB_AW_AGENT_OUTPUT"] = fmt.Sprintf("${{ needs.%s.outputs.output }}", mainJobName)
	env["GITHUB_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)
	// Pass the comment target configuration
	if data.SafeOutputs.AddComments.Target != "" {
		env["GITHUB_AW_COMMENT_TARGET"] = fmt.Sprintf("%q", data.SafeOutputs.AddComments.Target)
	}

	// Add custom environment variables from safe-outputs.env
	for key, value := range c.getCustomSafeOutputEnvVars(data) {
		env[key] = value
	}

	// Prepare with parameters
	withParams := make(map[string]string)
	// Get github-token if specified
	var token string
	if data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}
	if githubToken := c.getSafeOutputGitHubTokenForConfig(data, token); githubToken != "" {
		withParams["github-token"] = githubToken
	}

	// Build the github-script step using the simpler helper
	steps := BuildGitHubScriptStepLines(
		"Add Issue Comment",
		"add_comment",
		createCommentScript,
		env,
		withParams,
	)

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
		}

		return commentsConfig
	}

	return nil
}
