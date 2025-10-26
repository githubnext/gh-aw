package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// buildUpdateReactionJob creates a job that updates the activation comment with workflow completion status
// This job is only generated when both add-comment and ai-reaction are configured.
// This job runs when:
// 1. always() - runs even if agent fails
// 2. A comment was created in activation job (comment_id exists)
// 3. NO add_comment output was produced by the agent
// 4. NO create_pull_request output was produced by the agent
// This job depends on all safe output jobs to ensure it runs last
func (c *Compiler) buildUpdateReactionJob(data *WorkflowData, mainJobName string, safeOutputJobNames []string) (*Job, error) {
	// Create this job when:
	// 1. add-comment is configured with a reaction, OR
	// 2. command is configured with a reaction (which auto-creates a comment in activation)

	hasAddComment := data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil
	hasCommand := data.Command != ""
	hasReaction := data.AIReaction != "" && data.AIReaction != "none"

	// Only create this job when reactions are being used AND either add-comment or command is configured
	// This job updates the activation comment, which is only created when AIReaction is configured
	if !hasReaction {
		return nil, nil // No reaction configured or explicitly disabled, no comment to update
	}

	if !hasAddComment && !hasCommand {
		return nil, nil // Neither add-comment nor command is configured, no need for update_reaction job
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
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	customEnvVars = append(customEnvVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))

	// Get token from config
	var token string
	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Update reaction comment with completion status",
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
		Permissions: NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite().RenderToYAML(),
		Steps:       steps,
		Needs:       needs,
	}

	return job, nil
}
