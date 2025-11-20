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
	// 1. Safe outputs are configured (because noop is always enabled as a fallback)
	// The job will:
	// - Update activation comment with noop messages (if comment exists)
	// - Write noop messages to step summary (if no comment)

	hasAddComment := data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil
	hasCommand := data.Command != ""
	hasNoOp := data.SafeOutputs != nil && data.SafeOutputs.NoOp != nil
	hasReaction := data.AIReaction != "" && data.AIReaction != "none"
	hasSafeOutputs := data.SafeOutputs != nil

	notifyCommentLog.Printf("Configuration checks: has_add_comment=%t, has_command=%t, has_noop=%t, has_reaction=%t, has_safe_outputs=%t", hasAddComment, hasCommand, hasNoOp, hasReaction, hasSafeOutputs)

	// Always create this job when safe-outputs exist (because noop is always enabled)
	// This ensures noop messages can be handled even without reactions
	if !hasSafeOutputs {
		notifyCommentLog.Printf("Skipping job: no safe-outputs configured")
		return nil, nil // No safe-outputs configured, no need for conclusion job
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

	// Add artifact download steps once (shared by noop and conclusion steps)
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Add noop processing step if noop is configured
	if data.SafeOutputs.NoOp != nil {
		// Build custom environment variables specific to noop
		var noopEnvVars []string
		if data.SafeOutputs.NoOp.Max > 0 {
			noopEnvVars = append(noopEnvVars, fmt.Sprintf("          GH_AW_NOOP_MAX: %d\n", data.SafeOutputs.NoOp.Max))
		}

		// Add workflow metadata for consistency
		noopEnvVars = append(noopEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

		// Build the noop processing step (without artifact downloads - already added above)
		noopSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
			StepName:      "Process No-Op Messages",
			StepID:        "noop",
			MainJobName:   mainJobName,
			CustomEnvVars: noopEnvVars,
			Script:        getNoOpScript(),
			Token:         data.SafeOutputs.NoOp.GitHubToken,
		})
		steps = append(steps, noopSteps...)
	}

	// Build environment variables for the conclusion script
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	customEnvVars = append(customEnvVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	// Pass the tracker-id if present
	if data.TrackerID != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", data.TrackerID))
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))

	// Get token from config
	var token string
	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Build the conclusion GitHub Script step (without artifact downloads - already added above)
	scriptSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Update reaction comment with completion status",
		StepID:        "conclusion",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getNotifyCommentErrorScript(),
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	// Build the condition for this job:
	// 1. always() - run even if agent fails
	// 2. agent was activated (not skipped)
	// 3. IF comment_id exists: add_comment job either doesn't exist OR hasn't created a comment yet
	//
	// Note: The job should always run to handle noop messages (either update comment or write to summary)
	// The script (notify_comment_error.cjs) handles the case where there's no comment gracefully

	alwaysFunc := BuildFunctionCall("always")

	// Check that agent job was activated (not skipped)
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)

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
		// If add_comment job exists, also check that it hasn't already created a comment
		// This prevents duplicate updates when add_comment has already updated the activation comment
		noAddCommentOutput := &NotNode{
			Child: BuildPropertyAccess("needs.add_comment.outputs.comment_id"),
		}
		condition = buildAnd(
			buildAnd(alwaysFunc, agentNotSkipped),
			noAddCommentOutput,
		)
	} else {
		// If add_comment job doesn't exist, just check the basic conditions
		condition = buildAnd(alwaysFunc, agentNotSkipped)
	}

	// Build dependencies - this job depends on all safe output jobs to ensure it runs last
	needs := []string{mainJobName, constants.ActivationJobName}
	needs = append(needs, safeOutputJobNames...)

	notifyCommentLog.Printf("Job built successfully: dependencies_count=%d", len(needs))

	// Create outputs for the job (include noop output if configured)
	outputs := map[string]string{}
	if data.SafeOutputs.NoOp != nil {
		outputs["noop_message"] = "${{ steps.noop.outputs.noop_message }}"
	}

	job := &Job{
		Name:        "conclusion",
		If:          condition.Render(),
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite().RenderToYAML(),
		Steps:       steps,
		Needs:       needs,
		Outputs:     outputs,
	}

	return job, nil
}
