package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var notifyCommentLog = logger.New("workflow:notify_comment")

// buildConclusionJob creates a job that updates the activation comment with workflow completion status
// This job is only generated when both add-comment and ai-reaction are configured.
// This job runs when:
// 1. always() - runs even if agent fails
// 2. A comment was created in activation job (comment_id exists)
// 3. NO add_comment output was produced by the agent
// 4. NO create_pull_request output was produced by the agent
// This job depends on all safe output jobs to ensure it runs last
func (c *Compiler) buildConclusionJob(data *WorkflowData, mainJobName string, safeOutputJobNames []string) (*Job, error) {
	notifyCommentLog.Printf("Building conclusion job: main_job=%s, safe_output_jobs_count=%d", mainJobName, len(safeOutputJobNames))

	// Create this job when:
	// 1. add-comment is configured with a reaction, OR
	// 2. command is configured with a reaction (which auto-creates a comment in activation), OR
	// 3. noop is configured with a reaction (to post noop messages to activation comment)

	hasAddComment := data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil
	hasCommand := data.Command != ""
	hasNoOp := data.SafeOutputs != nil && data.SafeOutputs.NoOp != nil
	hasReaction := data.AIReaction != "" && data.AIReaction != "none"

	notifyCommentLog.Printf("Configuration checks: has_add_comment=%t, has_command=%t, has_noop=%t, has_reaction=%t", hasAddComment, hasCommand, hasNoOp, hasReaction)

	// Only create this job when reactions are being used AND either add-comment, command, or noop is configured
	// This job updates the activation comment, which is only created when AIReaction is configured
	if !hasReaction {
		notifyCommentLog.Printf("Skipping job: no reaction configured")
		return nil, nil // No reaction configured or explicitly disabled, no comment to update
	}

	if !hasAddComment && !hasCommand && !hasNoOp {
		notifyCommentLog.Printf("Skipping job: neither add-comment, command, nor noop configured")
		return nil, nil // Neither add-comment, command, nor noop is configured, no need for conclusion job
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
	// Pass the campaign if present
	if data.Campaign != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CAMPAIGN: %q\n", data.Campaign))
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))

	// Get token from config
	var token string
	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Update reaction comment with completion status",
		StepID:        "conclusion",
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
	// 4. add_comment job either doesn't exist OR hasn't created a comment yet
	//
	// Note: The job should run even when create_pull_request or push_to_pull_request_branch
	// output types are present, as those don't update the activation comment.

	alwaysFunc := BuildFunctionCall("always")

	// Check that agent job was activated (not skipped)
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)

	// Check that a comment was created in activation
	commentIdExists := BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.comment_id", constants.ActivationJobName))

	// Check if add_comment job exists in the safe output jobs
	hasAddCommentJob := false
	for _, jobName := range safeOutputJobNames {
		if jobName == "add_comment" {
			hasAddCommentJob = true
			break
		}
	}

	// Build the condition based on whether add_comment job exists
	var condition ConditionNode
	if hasAddCommentJob {
		// If add_comment job exists, check that it hasn't already created a comment
		// (i.e., check that needs.add_comment.outputs.comment_id is empty/false)
		noAddCommentOutput := &NotNode{
			Child: BuildPropertyAccess("needs.add_comment.outputs.comment_id"),
		}
		condition = buildAnd(
			buildAnd(
				buildAnd(alwaysFunc, agentNotSkipped),
				commentIdExists,
			),
			noAddCommentOutput,
		)
	} else {
		// If add_comment job doesn't exist, just check the basic conditions
		condition = buildAnd(
			buildAnd(alwaysFunc, agentNotSkipped),
			commentIdExists,
		)
	}

	// Build dependencies - this job depends on all safe output jobs to ensure it runs last
	needs := []string{mainJobName, constants.ActivationJobName}
	needs = append(needs, safeOutputJobNames...)

	notifyCommentLog.Printf("Job built successfully: dependencies_count=%d", len(needs))

	job := &Job{
		Name:        "conclusion",
		If:          condition.Render(),
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite().RenderToYAML(),
		Steps:       steps,
		Needs:       needs,
	}

	return job, nil
}
