package workflow

import (
	"fmt"
)

// CreatePullRequestReviewCommentsConfig holds configuration for creating GitHub pull request review comments from agent output
type CreatePullRequestReviewCommentsConfig struct {
	Max         int    `yaml:"max,omitempty"`          // Maximum number of review comments to create (default: 1)
	Side        string `yaml:"side,omitempty"`         // Side of the diff: "LEFT" or "RIGHT" (default: "RIGHT")
	GitHubToken string `yaml:"github-token,omitempty"` // GitHub token for this specific output type
}

// buildCreateOutputPullRequestReviewCommentJob creates the create_pr_review_comment job
func (c *Compiler) buildCreateOutputPullRequestReviewCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequestReviewComments == nil {
		return nil, fmt.Errorf("safe-outputs.create-pull-request-review-comment configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create PR Review Comment\n")
	steps = append(steps, "        id: create_pr_review_comment\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the side configuration
	if data.SafeOutputs.CreatePullRequestReviewComments.Side != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_REVIEW_COMMENT_SIDE: %q\n", data.SafeOutputs.CreatePullRequestReviewComments.Side))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		token = data.SafeOutputs.CreatePullRequestReviewComments.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createPRReviewCommentScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"review_comment_id":  "${{ steps.create_pr_review_comment.outputs.review_comment_id }}",
		"review_comment_url": "${{ steps.create_pr_review_comment.outputs.review_comment_url }}",
	}

	// We only run in pull request context, Note that in pull request comments only github.event.issue.pull_request is set.
	baseCondition := "(github.event.issue.number && github.event.issue.pull_request) || github.event.pull_request"

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		jobCondition = fmt.Sprintf("(%s) && (%s)", commandConditionStr, baseCondition)
	} else {
		// No command trigger, just use the base condition
		jobCondition = baseCondition
	}

	job := &Job{
		Name:           "create_pr_review_comment",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
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
	prReviewCommentsConfig := &CreatePullRequestReviewCommentsConfig{Max: 10, Side: "RIGHT"} // Default max is 10, side is RIGHT

	if configMap, ok := configData.(map[string]any); ok {
		// Parse max
		if max, exists := configMap["max"]; exists {
			if maxInt, ok := parseIntValue(max); ok {
				prReviewCommentsConfig.Max = maxInt
			}
		}

		// Parse side
		if side, exists := configMap["side"]; exists {
			if sideStr, ok := side.(string); ok {
				// Validate side value
				if sideStr == "LEFT" || sideStr == "RIGHT" {
					prReviewCommentsConfig.Side = sideStr
				}
			}
		}

		// Parse github-token
		if githubToken, exists := configMap["github-token"]; exists {
			if githubTokenStr, ok := githubToken.(string); ok {
				prReviewCommentsConfig.GitHubToken = githubTokenStr
			}
		}
	}

	return prReviewCommentsConfig
}