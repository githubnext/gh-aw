package workflow

import (
	"fmt"
)

// CreatePullRequestReviewCommentsConfig holds configuration for creating GitHub pull request review comments from agent output
type CreatePullRequestReviewCommentsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Side                 string `yaml:"side,omitempty"`   // Side of the diff: "LEFT" or "RIGHT" (default: "RIGHT")
	Target               string `yaml:"target,omitempty"` // Target for comments: "triggering" (default), "*" (any PR), or explicit PR number
}

// buildCreateOutputPullRequestReviewCommentJob creates the create_pr_review_comment job
func (c *Compiler) buildCreateOutputPullRequestReviewCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequestReviewComments == nil {
		return nil, fmt.Errorf("safe-outputs.create-pull-request-review-comment configuration is required")
	}

	// Prepare base environment variables
	env := make(map[string]string)
	env["GITHUB_AW_AGENT_OUTPUT"] = fmt.Sprintf("${{ needs.%s.outputs.output }}", mainJobName)
	env["GITHUB_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)
	if data.SafeOutputs.CreatePullRequestReviewComments.Side != "" {
		env["GITHUB_AW_PR_REVIEW_COMMENT_SIDE"] = fmt.Sprintf("%q", data.SafeOutputs.CreatePullRequestReviewComments.Side)
	}
	if data.SafeOutputs.CreatePullRequestReviewComments.Target != "" {
		env["GITHUB_AW_PR_REVIEW_COMMENT_TARGET"] = fmt.Sprintf("%q", data.SafeOutputs.CreatePullRequestReviewComments.Target)
	}

	// Add custom environment variables from safe-outputs.env
	c.getCustomSafeOutputEnvVars(env, data)

	// Prepare with parameters
	withParams := make(map[string]string)
	// Get github-token if specified
	var token string
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		token = data.SafeOutputs.CreatePullRequestReviewComments.GitHubToken
	}
	c.populateGitHubTokenForSafeOutput(withParams, data, token)

	// Build the github-script step using the simpler helper
	steps := BuildGitHubScriptStepLines(
		"Create PR Review Comment",
		"create_pr_review_comment",
		createPRReviewCommentScript,
		env,
		withParams,
	)

	// Create outputs for the job
	outputs := map[string]string{
		"review_comment_id":  "${{ steps.create_pr_review_comment.outputs.review_comment_id }}",
		"review_comment_url": "${{ steps.create_pr_review_comment.outputs.review_comment_url }}",
	}

	var jobCondition = BuildSafeOutputType("create-pull-request-review-comment", data.SafeOutputs.CreatePullRequestReviewComments.Min)
	if data.SafeOutputs.CreatePullRequestReviewComments != nil && data.SafeOutputs.CreatePullRequestReviewComments.Target == "" {
		issueWithPR := &AndNode{
			Left:  &ExpressionNode{Expression: "github.event.issue.number"},
			Right: &ExpressionNode{Expression: "github.event.issue.pull_request"},
		}
		eventCondition := buildOr(
			issueWithPR,
			BuildPropertyAccess("github.event.pull_request"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "create_pr_review_comment",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parsePullRequestReviewCommentsConfig handles create-pull-request-review-comment configuration
func (c *Compiler) parsePullRequestReviewCommentsConfig(outputMap map[string]any) *CreatePullRequestReviewCommentsConfig {
	if _, exists := outputMap["create-pull-request-review-comment"]; !exists {
		return nil
	}

	configData := outputMap["create-pull-request-review-comment"]
	prReviewCommentsConfig := &CreatePullRequestReviewCommentsConfig{Side: "RIGHT"} // Default side is RIGHT
	prReviewCommentsConfig.Max = 10                                                 // Default max is 10

	if configMap, ok := configData.(map[string]any); ok {
		// Parse common base fields
		c.parseBaseSafeOutputConfig(configMap, &prReviewCommentsConfig.BaseSafeOutputConfig)

		// Parse side
		if side, exists := configMap["side"]; exists {
			if sideStr, ok := side.(string); ok {
				// Validate side value
				if sideStr == "LEFT" || sideStr == "RIGHT" {
					prReviewCommentsConfig.Side = sideStr
				}
			}
		}

		// Parse target
		if target, exists := configMap["target"]; exists {
			if targetStr, ok := target.(string); ok {
				prReviewCommentsConfig.Target = targetStr
			}
		}
	}

	return prReviewCommentsConfig
}
