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

// buildCreateOutputAddCommentJob creates the create_issue_comment job
func (c *Compiler) buildCreateOutputAddCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil {
		return nil, fmt.Errorf("safe-outputs.add-comment configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Add Issue Comment\n")
	steps = append(steps, "        id: add_comment\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the comment target configuration
	if data.SafeOutputs.AddComments.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_COMMENT_TARGET: %q\n", data.SafeOutputs.AddComments.Target))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createCommentScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id":  "${{ steps.add_comment.outputs.comment_id }}",
		"comment_url": "${{ steps.add_comment.outputs.comment_url }}",
	}

	// Determine the job condition based on target configuration
	var baseCondition string
	if data.SafeOutputs.AddComments.Target == "*" {
		// Allow the job to run in any context when target is "*"
		baseCondition = "always()" // This allows the job to run even without triggering issue/PR
	} else {
		// Default behavior: only run in issue or PR context
		baseCondition = "github.event.issue.number || github.event.pull_request.number"
	}

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		if baseCondition == "always()" {
			// If base condition is always(), just use the command condition
			jobCondition = commandConditionStr
		} else {
			// Combine both conditions with AND
			jobCondition = fmt.Sprintf("(%s) && (%s)", commandConditionStr, baseCondition)
		}
	} else {
		// No command trigger, just use the base condition
		jobCondition = baseCondition
	}

	job := &Job{
		Name:           "create_issue_comment",
		If:             jobCondition,
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
