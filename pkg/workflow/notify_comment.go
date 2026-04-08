package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var notifyCommentLog = logger.New("workflow:notify_comment")

// buildConclusionJob creates a job that handles workflow completion tasks
// This job is generated when safe-outputs are configured and handles:
// - Updating status comments (if status-comment: true)
// - Processing noop messages
// - Handling agent failures
// - Recording missing tools
// This job runs when:
// 1. always() - runs even if agent fails
// 2. Agent job was not skipped
// 3. NO add_comment output was produced by the agent (avoids duplicate updates)
// This job depends on all safe output jobs to ensure it runs last
func (c *Compiler) buildConclusionJob(data *WorkflowData, mainJobName string, safeOutputJobNames []string) (*Job, error) {
	notifyCommentLog.Printf("Building conclusion job: main_job=%s, safe_output_jobs_count=%d", mainJobName, len(safeOutputJobNames))

	// Always create this job when safe-outputs exist (because noop is always enabled)
	// This ensures noop messages can be handled even without reactions
	if data.SafeOutputs == nil {
		notifyCommentLog.Printf("Skipping job: no safe-outputs configured")
		return nil, nil // No safe-outputs configured, no need for conclusion job
	}

	// Build the job steps
	var steps []string

	// Add setup step to copy scripts
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)

		// Notify comment job doesn't need project support
		// Conclusion/notify job depends on activation, reuse its trace ID
		notifyTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false, false, notifyTraceID)...)
	}

	// Add GitHub App token minting step if app is configured
	if data.SafeOutputs.GitHubApp != nil {
		// Compute permissions based on configured safe outputs (principle of least privilege)
		permissions := ComputePermissionsForSafeOutputs(data.SafeOutputs)
		// For workflow_call relay workflows, scope the token to the platform repo name only
		// (not the full slug) because actions/create-github-app-token expects repo names
		// without the owner prefix when `owner` is also set.
		var appTokenFallbackRepo string
		if hasWorkflowCallTrigger(data.On) {
			appTokenFallbackRepo = "${{ needs.activation.outputs.target_repo_name }}"
		}
		steps = append(steps, c.buildGitHubAppTokenMintStep(data.SafeOutputs.GitHubApp, permissions, appTokenFallbackRepo)...)
	}

	// Add artifact download steps once (shared by noop and conclusion steps).
	// In workflow_call context, use the per-invocation prefix to avoid artifact name clashes.
	steps = append(steps, buildAgentOutputDownloadSteps(artifactPrefixExprForDownstreamJob(data))...)

	// Add noop processing step if noop is configured.
	// This single step replaces the former two-step "Process No-Op Messages" + "Handle No-Op Message"
	// sequence: handle_noop_message.cjs now loads agent output directly (no cross-step dep).
	if data.SafeOutputs.NoOp != nil {
		// Build custom environment variables for the merged noop step
		var noopEnvVars []string
		noopEnvVars = append(noopEnvVars, buildTemplatableIntEnvVar("GH_AW_NOOP_MAX", data.SafeOutputs.NoOp.Max)...)

		// Add workflow metadata for consistency
		noopEnvVars = append(noopEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

		// Agent conclusion and run URL are used to decide whether to post to the runs issue
		noopEnvVars = append(noopEnvVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
		noopEnvVars = append(noopEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
		noopEnvVars = append(noopEnvVars, buildTemplatableBoolEnvVar("GH_AW_NOOP_REPORT_AS_ISSUE", data.SafeOutputs.NoOp.ReportAsIssue)...)
		if data.SafeOutputs.NoOp.ReportAsIssue == nil {
			noopEnvVars = append(noopEnvVars, "          GH_AW_NOOP_REPORT_AS_ISSUE: \"true\"\n")
		}

		// Build the merged noop step (without artifact downloads - already added above)
		noopSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
			StepName:      "Process No-Op Messages",
			StepID:        "noop",
			MainJobName:   mainJobName,
			CustomEnvVars: noopEnvVars,
			ScriptFile:    "handle_noop_message.cjs",
			CustomToken:   data.SafeOutputs.NoOp.GitHubToken,
		})
		steps = append(steps, noopSteps...)
	}

	// Add missing_tool processing step if missing-tool is configured
	if data.SafeOutputs.MissingTool != nil {
		// Build custom environment variables specific to missing-tool
		var missingToolEnvVars []string
		missingToolEnvVars = append(missingToolEnvVars, buildTemplatableIntEnvVar("GH_AW_MISSING_TOOL_MAX", data.SafeOutputs.MissingTool.Max)...)

		// Add create-issue configuration
		if data.SafeOutputs.MissingTool.CreateIssue {
			missingToolEnvVars = append(missingToolEnvVars, "          GH_AW_MISSING_TOOL_CREATE_ISSUE: \"true\"\n")
		}

		// Add title-prefix configuration
		if data.SafeOutputs.MissingTool.TitlePrefix != "" {
			missingToolEnvVars = append(missingToolEnvVars, fmt.Sprintf("          GH_AW_MISSING_TOOL_TITLE_PREFIX: %q\n", data.SafeOutputs.MissingTool.TitlePrefix))
		}

		// Add labels configuration
		if len(data.SafeOutputs.MissingTool.Labels) > 0 {
			labelsJSON, err := json.Marshal(data.SafeOutputs.MissingTool.Labels)
			if err == nil {
				missingToolEnvVars = append(missingToolEnvVars, fmt.Sprintf("          GH_AW_MISSING_TOOL_LABELS: %q\n", string(labelsJSON)))
			}
		}

		// Add workflow metadata for consistency
		missingToolEnvVars = append(missingToolEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

		// Build the missing_tool processing step (without artifact downloads - already added above)
		missingToolSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
			StepName:      "Record missing tool",
			StepID:        "missing_tool",
			MainJobName:   mainJobName,
			CustomEnvVars: missingToolEnvVars,
			Script:        "const { main } = require('${{ runner.temp }}/gh-aw/actions/missing_tool.cjs'); await main();",
			ScriptFile:    "missing_tool.cjs",
			CustomToken:   data.SafeOutputs.MissingTool.GitHubToken,
		})
		steps = append(steps, missingToolSteps...)
	}

	// Add report_incomplete processing step if report-incomplete is configured
	if data.SafeOutputs.ReportIncomplete != nil {
		// Build custom environment variables specific to report-incomplete
		var reportIncompleteEnvVars []string
		reportIncompleteEnvVars = append(reportIncompleteEnvVars, buildTemplatableIntEnvVar("GH_AW_REPORT_INCOMPLETE_MAX", data.SafeOutputs.ReportIncomplete.Max)...)

		// Add create-issue configuration
		if data.SafeOutputs.ReportIncomplete.CreateIssue {
			reportIncompleteEnvVars = append(reportIncompleteEnvVars, "          GH_AW_REPORT_INCOMPLETE_CREATE_ISSUE: \"true\"\n")
		}

		// Add title-prefix configuration
		if data.SafeOutputs.ReportIncomplete.TitlePrefix != "" {
			reportIncompleteEnvVars = append(reportIncompleteEnvVars, fmt.Sprintf("          GH_AW_REPORT_INCOMPLETE_TITLE_PREFIX: %q\n", data.SafeOutputs.ReportIncomplete.TitlePrefix))
		}

		// Add labels configuration
		if len(data.SafeOutputs.ReportIncomplete.Labels) > 0 {
			labelsJSON, err := json.Marshal(data.SafeOutputs.ReportIncomplete.Labels)
			if err == nil {
				reportIncompleteEnvVars = append(reportIncompleteEnvVars, fmt.Sprintf("          GH_AW_REPORT_INCOMPLETE_LABELS: %q\n", string(labelsJSON)))
			}
		}

		// Add workflow metadata for consistency
		reportIncompleteEnvVars = append(reportIncompleteEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

		// Build the report_incomplete processing step (without artifact downloads - already added above)
		reportIncompleteSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
			StepName:      "Record incomplete",
			StepID:        "report_incomplete",
			MainJobName:   mainJobName,
			CustomEnvVars: reportIncompleteEnvVars,
			Script:        "const { main } = require('${{ runner.temp }}/gh-aw/actions/report_incomplete_handler.cjs'); await main();",
			ScriptFile:    "report_incomplete_handler.cjs",
			CustomToken:   data.SafeOutputs.ReportIncomplete.GitHubToken,
		})
		steps = append(steps, reportIncompleteSteps...)
	}

	// Add agent failure handling step - creates/updates an issue when agent job fails
	// This step always runs and checks if the agent job failed
	// Build environment variables for the agent failure handler

	// Serialize messages config once for reuse in both handler steps below.
	var messagesJSON string
	if data.SafeOutputs != nil && data.SafeOutputs.Messages != nil {
		if json, jsonErr := serializeMessagesConfig(data.SafeOutputs.Messages); jsonErr != nil {
			notifyCommentLog.Printf("Warning: failed to serialize messages config: %v", jsonErr)
		} else {
			messagesJSON = json
		}
	}

	var agentFailureEnvVars []string
	agentFailureEnvVars = append(agentFailureEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)
	agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
	agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", data.WorkflowID))

	// Pass the engine ID so the failure handler can surface which AI engine terminated
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_ENGINE_ID: %q\n", data.EngineConfig.ID))
	}

	// Only add secret_verification_result if the engine provides a validate-secret step.
	// The validate-secret step runs in the activation job, so the output is on needs.activation.
	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}
	if EngineHasValidateSecretStep(engine, data) {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_SECRET_VERIFICATION_RESULT: ${{ needs.%s.outputs.secret_verification_result }}\n", string(constants.ActivationJobName)))
	}

	// Add checkout_pr_success to detect PR checkout failures (e.g., PR merged and branch deleted)
	// Only add if the checkout-pr step will be generated (requires contents read access)
	if ShouldGeneratePRCheckoutStep(data) {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_CHECKOUT_PR_SUCCESS: ${{ needs.%s.outputs.checkout_pr_success }}\n", mainJobName))
	}

	// Pass inference access error output for Copilot engine
	// This detects when the Copilot CLI fails due to the token lacking inference access
	if _, ok := engine.(*CopilotEngine); ok {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_INFERENCE_ACCESS_ERROR: ${{ needs.%s.outputs.inference_access_error }}\n", mainJobName))
	}

	// Pass assignment error outputs from safe_outputs job if assign-to-agent is configured
	if data.SafeOutputs != nil && data.SafeOutputs.AssignToAgent != nil {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_ASSIGNMENT_ERRORS: ${{ needs.safe_outputs.outputs.assign_to_agent_assignment_errors }}\n")
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_ASSIGNMENT_ERROR_COUNT: ${{ needs.safe_outputs.outputs.assign_to_agent_assignment_error_count }}\n")
	}

	// Pass copilot assignment failure outputs from safe_outputs job if create-issue with copilot assignee is configured
	if data.SafeOutputs != nil && data.SafeOutputs.CreateIssues != nil && hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees) {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_ASSIGN_COPILOT_FAILURE_COUNT: ${{ needs.safe_outputs.outputs.assign_copilot_failure_count }}\n")
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_ASSIGN_COPILOT_ERRORS: ${{ needs.safe_outputs.outputs.assign_copilot_errors }}\n")
	}

	// Pass create_discussion error outputs from safe_outputs job if create-discussions is configured
	if data.SafeOutputs != nil && data.SafeOutputs.CreateDiscussions != nil {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_CREATE_DISCUSSION_ERRORS: ${{ needs.safe_outputs.outputs.create_discussion_errors }}\n")
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_CREATE_DISCUSSION_ERROR_COUNT: ${{ needs.safe_outputs.outputs.create_discussion_error_count }}\n")
	}

	// Pass code-push failure outputs from safe_outputs job if push-to-pull-request-branch or create-pull-request is configured
	if data.SafeOutputs != nil && (data.SafeOutputs.PushToPullRequestBranch != nil || data.SafeOutputs.CreatePullRequests != nil) {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_CODE_PUSH_FAILURE_ERRORS: ${{ needs.safe_outputs.outputs.code_push_failure_errors }}\n")
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_CODE_PUSH_FAILURE_COUNT: ${{ needs.safe_outputs.outputs.code_push_failure_count }}\n")
	}

	// Pass GitHub App token minting failure status so the handler can surface auth errors.
	// The safe_outputs job tracks whether its token step failed as a job output.
	// The conclusion job tracks its own app token step outcome directly via steps context.
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubApp != nil {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_SAFE_OUTPUTS_APP_TOKEN_MINTING_FAILED: ${{ needs.safe_outputs.outputs.app_token_minting_failed }}\n")
		// Also check the conclusion job's own app token step outcome; this is important because
		// the Handle Agent Failure step must use if: always() to run even when this step fails.
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_CONCLUSION_APP_TOKEN_MINTING_FAILED: ${{ steps.safe-outputs-app-token.outcome == 'failure' }}\n")
	}

	// Pass activation job GitHub App token minting failure status if configured.
	if data.ActivationGitHubApp != nil {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_ACTIVATION_APP_TOKEN_MINTING_FAILED: ${{ needs.%s.outputs.activation_app_token_minting_failed }}\n", string(constants.ActivationJobName)))
	}

	// Always pass lockdown check failure status so the handler can surface configuration
	// errors even when the agent job was skipped due to the lockdown check failing.
	agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_LOCKDOWN_CHECK_FAILED: ${{ needs.%s.outputs.lockdown_check_failed }}\n", string(constants.ActivationJobName)))

	// Pass custom messages config if present (JSON computed once above)
	if messagesJSON != "" {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
	}

	// Pass repo-memory failure outputs if repo-memory is configured
	// This allows the agent failure handler to report both job-level failures and validation issues
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		// Pass the overall push_repo_memory job result so the failure handler
		// can report when the push job itself fails (e.g. permission or configuration errors)
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_PUSH_REPO_MEMORY_RESULT: ${{ needs.push_repo_memory.result }}\n")
		for _, memory := range data.RepoMemoryConfig.Memories {
			// Add validation status for each memory
			agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_REPO_MEMORY_VALIDATION_FAILED_%s: ${{ needs.push_repo_memory.outputs.validation_failed_%s }}\n", memory.ID, memory.ID))
			agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_REPO_MEMORY_VALIDATION_ERROR_%s: ${{ needs.push_repo_memory.outputs.validation_error_%s }}\n", memory.ID, memory.ID))
			agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_REPO_MEMORY_PATCH_SIZE_EXCEEDED_%s: ${{ needs.push_repo_memory.outputs.patch_size_exceeded_%s }}\n", memory.ID, memory.ID))
		}
	}

	// Pass group-reports configuration flag (defaults to false if not specified)
	if data.SafeOutputs != nil && data.SafeOutputs.GroupReports {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_GROUP_REPORTS: \"true\"\n")
	} else {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_GROUP_REPORTS: \"false\"\n")
	}

	// Pass report-failure-as-issue configuration flag (defaults to true if not specified)
	if data.SafeOutputs != nil && data.SafeOutputs.ReportFailureAsIssue != nil && !*data.SafeOutputs.ReportFailureAsIssue {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_FAILURE_REPORT_AS_ISSUE: \"false\"\n")
	} else {
		agentFailureEnvVars = append(agentFailureEnvVars, "          GH_AW_FAILURE_REPORT_AS_ISSUE: \"true\"\n")
	}

	// Pass failure-issue-repo configuration (optional, defaults to current repo)
	if data.SafeOutputs != nil && data.SafeOutputs.FailureIssueRepo != "" {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_FAILURE_ISSUE_REPO: %q\n", data.SafeOutputs.FailureIssueRepo))
	}

	// Pass timeout minutes value so the failure handler can provide an actionable hint when timed out
	timeoutValue := strings.TrimPrefix(data.TimeoutMinutes, "timeout-minutes: ")
	if timeoutValue != "" {
		agentFailureEnvVars = append(agentFailureEnvVars, fmt.Sprintf("          GH_AW_TIMEOUT_MINUTES: %q\n", timeoutValue))
	}

	// Build the agent failure handling step.
	// Use if: always() so this step runs even when an earlier step in the conclusion job
	// (such as the GitHub App token minting step) has failed. The handler uses the default
	// GITHUB_TOKEN and does not depend on the app-minted token.
	agentFailureSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Handle agent failure",
		StepID:        "handle_agent_failure",
		MainJobName:   mainJobName,
		CustomEnvVars: agentFailureEnvVars,
		Script:        "const { main } = require('${{ runner.temp }}/gh-aw/actions/handle_agent_failure.cjs'); await main();",
		ScriptFile:    "handle_agent_failure.cjs",
		CustomToken:   "", // Will use default GITHUB_TOKEN
		StepCondition: "always()",
	})
	steps = append(steps, agentFailureSteps...)

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

	// Pass detection conclusion if threat detection is enabled (in separate detection job)
	if IsDetectionJobEnabled(data.SafeOutputs) {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DETECTION_CONCLUSION: ${{ needs.%s.outputs.detection_conclusion }}\n", constants.DetectionJobName))
		notifyCommentLog.Print("Added detection conclusion environment variable to conclusion job")
	}

	// Pass assignment error count to the conclusion step so the status comment reflects assignment failures
	if data.SafeOutputs != nil && data.SafeOutputs.AssignToAgent != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_ASSIGNMENT_ERROR_COUNT: ${{ needs.safe_outputs.outputs.assign_to_agent_assignment_error_count }}\n")
	}

	// Pass custom messages config if present (JSON computed once above, reused here)
	if messagesJSON != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
	}

	// Pass safe output job information for link generation
	if len(safeOutputJobNames) > 0 {
		safeOutputJobsJSON, jobURLEnvVars := buildSafeOutputJobsEnvVars(safeOutputJobNames)
		if safeOutputJobsJSON != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_JOBS: %q\n", safeOutputJobsJSON))
			customEnvVars = append(customEnvVars, jobURLEnvVars...)
			notifyCommentLog.Printf("Added safe output jobs info for %d job(s)", len(safeOutputJobNames))
		}
	}

	// Get token from config
	var token string
	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}

	// Only add the conclusion update step if status comments are explicitly enabled
	if data.StatusComment != nil && *data.StatusComment {
		// Build the conclusion GitHub Script step (without artifact downloads - already added above)
		scriptSteps := c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
			StepName:      "Update reaction comment with completion status",
			StepID:        "conclusion",
			MainJobName:   mainJobName,
			CustomEnvVars: customEnvVars,
			Script:        getNotifyCommentErrorScript(),
			ScriptFile:    "notify_comment_error.cjs",
			CustomToken:   token,
		})
		steps = append(steps, scriptSteps...)
	}

	// Note: Unlock step has been moved to a dedicated unlock job
	// that always runs, even if this conclusion job doesn't run.
	// See buildUnlockJob() in compiler_unlock_job.go

	// Add GitHub App token invalidation step if app is configured
	if data.SafeOutputs.GitHubApp != nil {
		notifyCommentLog.Print("Adding GitHub App token invalidation step to conclusion job")
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}

	// Append OTLP conclusion span step (no-op when endpoint is not configured).
	// Note: this step is now handled by the action post step (post.js) so no
	// injected step is needed here.

	// Build the condition for this job:
	// 1. always() - run even if agent fails
	// 2. agent was activated (not skipped) OR lockdown check failed in activation job
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

	// Check if the lockdown check failed in the activation job — when this happens the agent
	// is skipped, but we still want the conclusion job to run so it can report the failure.
	lockdownCheckFailed := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.lockdown_check_failed", string(constants.ActivationJobName))),
		BuildStringLiteral("true"),
	)

	// Agent not skipped OR lockdown check failed
	agentNotSkippedOrLockdownFailed := BuildOr(agentNotSkipped, lockdownCheckFailed)

	// Check if add_comment job exists in the safe output jobs
	hasAddCommentJob := slices.Contains(safeOutputJobNames, "add_comment")

	// Build the condition based on whether add_comment job exists
	var condition ConditionNode
	if hasAddCommentJob {
		// If add_comment job exists, also check that it hasn't already created a comment
		// This prevents duplicate updates when add_comment has already updated the activation comment
		noAddCommentOutput := &NotNode{
			Child: BuildPropertyAccess("needs.add_comment.outputs.comment_id"),
		}
		condition = BuildAnd(
			BuildAnd(alwaysFunc, agentNotSkippedOrLockdownFailed),
			noAddCommentOutput,
		)
	} else {
		// If add_comment job doesn't exist, just check the basic conditions
		condition = BuildAnd(alwaysFunc, agentNotSkippedOrLockdownFailed)
	}

	// Build dependencies - this job depends on all safe output jobs to ensure it runs last
	needs := []string{mainJobName, string(constants.ActivationJobName)}
	needs = append(needs, safeOutputJobNames...)

	// When threat detection is enabled, the conclusion job also depends on the detection job
	// so that needs.detection.outputs.detection_conclusion is accessible.
	if IsDetectionJobEnabled(data.SafeOutputs) {
		needs = append(needs, string(constants.DetectionJobName))
		notifyCommentLog.Print("Added detection job dependency to conclusion job")
	}

	notifyCommentLog.Printf("Job built successfully: dependencies_count=%d", len(needs))

	// Create outputs for the job (include noop and missing_tool outputs if configured)
	outputs := map[string]string{}
	if data.SafeOutputs.NoOp != nil {
		outputs["noop_message"] = "${{ steps.noop.outputs.noop_message }}"
	}
	if data.SafeOutputs.MissingTool != nil {
		outputs["tools_reported"] = "${{ steps.missing_tool.outputs.tools_reported }}"
		outputs["total_count"] = "${{ steps.missing_tool.outputs.total_count }}"
	}
	if data.SafeOutputs.ReportIncomplete != nil {
		outputs["incomplete_count"] = "${{ steps.report_incomplete.outputs.incomplete_count }}"
	}

	// Compute permissions based on configured safe outputs (principle of least privilege)
	permissions := ComputePermissionsForSafeOutputs(data.SafeOutputs)

	// Build concurrency config for the conclusion job using the workflow ID.
	// This prevents concurrent agents on the same workflow from interfering with each other.
	var concurrency string
	if data.WorkflowID != "" {
		group := "gh-aw-conclusion-" + data.WorkflowID
		// If the user specified a job-discriminator, append it so that concurrent
		// runs with different inputs (fan-out pattern) do not share the same group.
		if data.ConcurrencyJobDiscriminator != "" {
			notifyCommentLog.Printf("Appending job discriminator to conclusion job concurrency group: %s", data.ConcurrencyJobDiscriminator)
			group = fmt.Sprintf("%s-%s", group, data.ConcurrencyJobDiscriminator)
		}
		concurrency = c.indentYAMLLines(fmt.Sprintf("concurrency:\n  group: %q\n  cancel-in-progress: false", group), "    ")
		notifyCommentLog.Printf("Configuring conclusion job concurrency group: %s", group)
	}

	// In script mode, explicitly add a cleanup step (mirrors post.js in dev/release/action mode).
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	job := &Job{
		Name:        "conclusion",
		If:          RenderCondition(condition),
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		Environment: c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions: permissions.RenderToYAML(),
		Concurrency: concurrency,
		Steps:       steps,
		Needs:       needs,
		Outputs:     outputs,
	}

	return job, nil
}

// systemSafeOutputJobNames contains job names that are built-in system jobs and should not be
// treated as custom safe output job types in the GH_AW_SAFE_OUTPUT_JOBS mapping.
// The safe output handler manager uses this mapping to determine which message types are
// handled by custom job steps (and therefore should be silently skipped rather than flagged
// as "no handler loaded").
var systemSafeOutputJobNames = map[string]bool{
	"safe_outputs":  true, // consolidated safe outputs job
	"upload_assets": true, // upload assets job
}

// buildSafeOutputJobsEnvVars creates environment variables for safe output job URLs
// Returns both a JSON mapping and the actual environment variable declarations.
// The mapping includes:
//   - Built-in jobs with known URL outputs (e.g., create_issue → issue_url)
//   - Custom safe-output jobs (from safe-outputs.jobs) with an empty URL key, so the handler
//     manager knows those message types are handled by a dedicated job step and should be
//     skipped gracefully rather than reported as "No handler loaded".
func buildSafeOutputJobsEnvVars(jobNames []string) (string, []string) {
	// Map job names to their expected URL output keys
	jobOutputMapping := make(map[string]string)
	var envVars []string

	for _, jobName := range jobNames {
		var urlKey string
		switch jobName {
		case "create_issue":
			urlKey = "issue_url"
		case "add_comment":
			urlKey = "comment_url"
		case "create_pull_request":
			urlKey = "pull_request_url"
		case "create_discussion":
			urlKey = "discussion_url"
		case "create_pr_review_comment":
			urlKey = "review_comment_url"
		case "close_issue":
			urlKey = "issue_url"
		case "close_pull_request":
			urlKey = "pull_request_url"
		case "close_discussion":
			urlKey = "discussion_url"
		case "create_agent_session":
			urlKey = "task_url"
		case "push_to_pull_request_branch":
			urlKey = "commit_url"
		default:
			if !systemSafeOutputJobNames[jobName] {
				// Custom safe-output job: include in the mapping with an empty URL key so the
				// handler manager can silently skip messages of this type.
				jobOutputMapping[jobName] = ""
			}
			continue
		}

		jobOutputMapping[jobName] = urlKey

		// Add environment variable for this job's URL output
		envVarName := fmt.Sprintf("GH_AW_OUTPUT_%s_%s",
			toEnvVarCase(jobName),
			toEnvVarCase(urlKey))
		envVars = append(envVars,
			fmt.Sprintf("          %s: ${{ needs.%s.outputs.%s }}\n",
				envVarName, jobName, urlKey))
	}

	if len(jobOutputMapping) == 0 {
		return "", nil
	}

	jsonBytes, err := json.Marshal(jobOutputMapping)
	if err != nil {
		notifyCommentLog.Printf("Warning: failed to marshal safe output jobs info: %v", err)
		return "", nil
	}

	return string(jsonBytes), envVars
}

// toEnvVarCase converts a string to uppercase environment variable case
func toEnvVarCase(s string) string {
	// Convert to uppercase and keep underscores
	var result strings.Builder
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			result.WriteRune(ch - 32) // Convert to uppercase
		} else if ch >= 'A' && ch <= 'Z' {
			result.WriteRune(ch)
		} else if ch == '_' {
			result.WriteString("_")
		}
	}
	return result.String()
}
