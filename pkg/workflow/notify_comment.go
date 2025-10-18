package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// buildUpdateReactionJob creates a job that updates the activation comment when the agent fails
// This job runs when:
// 1. always() - runs even if agent fails
// 2. A comment was created in activation job (comment_id exists)
// 3. NO add_comment output was produced by the agent
// 4. NO create_pull_request output was produced by the agent
// This job depends on all safe output jobs to ensure it runs last
func (c *Compiler) buildUpdateReactionJob(data *WorkflowData, mainJobName string, safeOutputJobNames []string) (*Job, error) {
	// Only create this job when add-comment is configured OR reaction is configured
	// Both of these create a comment_id that can be updated on failure
	hasAddComment := data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil
	hasReaction := data.AIReaction != ""

	if !hasAddComment && !hasReaction {
		return nil, nil
	}

	// Build the job steps
	var steps []string

	// Add debug step
	steps = append(steps, "      - name: Debug job inputs\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
	steps = append(steps, fmt.Sprintf("          COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	steps = append(steps, fmt.Sprintf("          AGENT_OUTPUT_TYPES: ${{ needs.%s.outputs.output_types }}\n", mainJobName))
	steps = append(steps, fmt.Sprintf("          AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Comment ID: $COMMENT_ID\"\n")
	steps = append(steps, "          echo \"Comment Repo: $COMMENT_REPO\"\n")
	steps = append(steps, "          echo \"Agent Output Types: $AGENT_OUTPUT_TYPES\"\n")
	steps = append(steps, "          echo \"Agent Conclusion: $AGENT_CONCLUSION\"\n")

	// Build environment variables for the script
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	customEnvVars = append(customEnvVars, "          GITHUB_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))

	// Get token from config if add-comment is configured
	// Otherwise, empty token means use default GITHUB_TOKEN
	var token string
	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Update reaction comment with error notification",
		StepID:        "update_reaction",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        notifyCommentErrorScript,
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	// Build the condition for this job:
	// 1. always() - run even if agent fails
	// 2. agent was activated (not skipped)
	// 3. comment_id exists (comment was created in activation)
	// 4. NOT contains(output_types, 'add_comment')
	// 5. NOT contains(output_types, 'create_pull_request')
	// 6. NOT contains(output_types, 'push_to_pull_request_branch')

	alwaysFunc := BuildFunctionCall("always")

	// Check that agent job was activated (not skipped)
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)

	// Check that a comment was created in activation
	commentIdExists := BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.comment_id", constants.ActivationJobName))

	// Check that output_types doesn't contain add_comment
	noAddComment := &NotNode{
		Child: BuildFunctionCall("contains",
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
			BuildStringLiteral("add_comment"),
		),
	}

	// Check that output_types doesn't contain create_pull_request
	noCreatePR := &NotNode{
		Child: BuildFunctionCall("contains",
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
			BuildStringLiteral("create_pull_request"),
		),
	}

	// Check that output_types doesn't contain push_to_pull_request_branch
	noPushToBranch := &NotNode{
		Child: BuildFunctionCall("contains",
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
			BuildStringLiteral("push_to_pull_request_branch"),
		),
	}

	// Combine all conditions with AND
	condition := buildAnd(
		buildAnd(
			buildAnd(
				buildAnd(
					buildAnd(alwaysFunc, agentNotSkipped),
					commentIdExists,
				),
				noAddComment,
			),
			noCreatePR,
		),
		noPushToBranch,
	)

	// Build dependencies - this job depends on all safe output jobs to ensure it runs last
	needs := []string{mainJobName, constants.ActivationJobName}
	needs = append(needs, safeOutputJobNames...)

	job := &Job{
		Name:        "update_reaction",
		If:          condition.Render(),
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: "permissions:\n      contents: read\n      issues: write\n      pull-requests: write\n      discussions: write",
		Steps:       steps,
		Needs:       needs,
	}

	return job, nil
}
