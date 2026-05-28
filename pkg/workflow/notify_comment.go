package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
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
	if data.SafeOutputs == nil {
		notifyCommentLog.Printf("Skipping job: no safe-outputs configured")
		return nil, nil
	}

	steps := c.buildConclusionSetupSteps(data)
	steps = append(steps, c.buildConclusionSupportSteps(data, mainJobName)...)

	messagesJSON := serializeConclusionMessages(data.SafeOutputs)
	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}

	agentFailureEnvVars := c.buildAgentFailureEnvVars(data, mainJobName, engine, messagesJSON)
	steps = append(steps, c.buildAgentFailureStep(data, mainJobName, agentFailureEnvVars)...)
	steps = append(steps, c.buildConclusionUpdateSteps(data, mainJobName, safeOutputJobNames, messagesJSON)...)
	steps = append(steps, c.buildConclusionFinalSteps(data)...)

	job := &Job{
		Name:        "conclusion",
		If:          RenderCondition(buildConclusionCondition(safeOutputJobNames)),
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		Environment: c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions: ComputePermissionsForSafeOutputs(data.SafeOutputs).RenderToYAML(),
		Concurrency: c.buildConclusionConcurrency(data),
		Steps:       steps,
		Needs:       buildConclusionNeeds(data, mainJobName, safeOutputJobNames),
		Outputs:     buildConclusionOutputs(data),
	}
	return job, nil
}

func (c *Compiler) buildConclusionSetupSteps(data *WorkflowData) []string {
	var steps []string
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		notifyTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		notifyParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, notifyTraceID, notifyParentSpanID)...)
	}
	if data.SafeOutputs.GitHubApp != nil {
		permissions := ComputePermissionsForSafeOutputs(data.SafeOutputs)
		var appTokenFallbackRepo string
		if hasWorkflowCallTrigger(data.On) {
			appTokenFallbackRepo = "${{ needs.activation.outputs.target_repo_name }}"
		}
		steps = append(steps, c.buildGitHubAppTokenMintStep(data.SafeOutputs.GitHubApp, permissions, appTokenFallbackRepo)...)
	}
	return append(steps, buildAgentOutputDownloadSteps(artifactPrefixExprForDownstreamJob(data), c.getActionPin)...)
}

func (c *Compiler) buildConclusionSupportSteps(data *WorkflowData, mainJobName string) []string {
	var steps []string
	steps = append(steps, c.buildNoOpSteps(data, mainJobName)...)
	steps = append(steps, c.buildDetectionRunsSteps(data)...)
	steps = append(steps, c.buildMissingToolSteps(data, mainJobName)...)
	steps = append(steps, c.buildReportIncompleteSteps(data, mainJobName)...)
	return steps
}

func (c *Compiler) buildNoOpSteps(data *WorkflowData, mainJobName string) []string {
	if data.SafeOutputs.NoOp == nil {
		return nil
	}
	var noopEnvVars []string
	noopEnvVars = append(noopEnvVars, buildTemplatableIntEnvVar("GH_AW_NOOP_MAX", data.SafeOutputs.NoOp.Max)...)
	noopEnvVars = append(noopEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID, buildLocalWorkflowSourceURL(c.markdownPath))...)
	noopEnvVars = append(noopEnvVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	noopEnvVars = append(noopEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
	noopEnvVars = append(noopEnvVars, buildTemplatableBoolEnvVar("GH_AW_NOOP_REPORT_AS_ISSUE", data.SafeOutputs.NoOp.ReportAsIssue)...)
	if data.SafeOutputs.NoOp.ReportAsIssue == nil {
		noopEnvVars = append(noopEnvVars, "          GH_AW_NOOP_REPORT_AS_ISSUE: \"true\"\n")
	}
	return c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Process no-op messages",
		StepID:        "noop",
		MainJobName:   mainJobName,
		CustomEnvVars: noopEnvVars,
		ScriptFile:    "handle_noop_message.cjs",
		CustomToken:   data.SafeOutputs.NoOp.GitHubToken,
	})
}

func (c *Compiler) buildDetectionRunsSteps(data *WorkflowData) []string {
	if !IsDetectionJobEnabled(data.SafeOutputs) {
		return nil
	}
	var envVars []string
	envVars = append(envVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID, buildLocalWorkflowSourceURL(c.markdownPath))...)
	envVars = append(envVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	envVars = append(envVars, fmt.Sprintf("          GH_AW_DETECTION_CONCLUSION: ${{ needs.%s.outputs.detection_conclusion }}\n", constants.DetectionJobName))
	envVars = append(envVars, fmt.Sprintf("          GH_AW_DETECTION_REASON: ${{ needs.%s.outputs.detection_reason }}\n", constants.DetectionJobName))
	notifyCommentLog.Print("Added detection runs logging step to conclusion job")
	return c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Log detection run",
		StepID:        "detection_runs",
		CustomEnvVars: envVars,
		ScriptFile:    "handle_detection_runs.cjs",
	})
}

func (c *Compiler) buildMissingToolSteps(data *WorkflowData, mainJobName string) []string {
	if data.SafeOutputs.MissingTool == nil {
		return nil
	}
	var envVars []string
	envVars = append(envVars, buildTemplatableIntEnvVar("GH_AW_MISSING_TOOL_MAX", data.SafeOutputs.MissingTool.Max)...)
	envVars = append(envVars, buildTemplatableBoolEnvVar("GH_AW_MISSING_TOOL_CREATE_ISSUE", data.SafeOutputs.MissingTool.CreateIssue)...)
	if data.SafeOutputs.MissingTool.TitlePrefix != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_MISSING_TOOL_TITLE_PREFIX: %q\n", data.SafeOutputs.MissingTool.TitlePrefix))
	}
	if len(data.SafeOutputs.MissingTool.Labels) > 0 {
		if labelsJSON, err := json.Marshal(data.SafeOutputs.MissingTool.Labels); err == nil {
			envVars = append(envVars, fmt.Sprintf("          GH_AW_MISSING_TOOL_LABELS: %q\n", string(labelsJSON)))
		}
	}
	envVars = append(envVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID, buildLocalWorkflowSourceURL(c.markdownPath))...)
	return c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Record missing tool",
		StepID:        "missing_tool",
		MainJobName:   mainJobName,
		CustomEnvVars: envVars,
		Script:        "const { main } = require('${{ runner.temp }}/gh-aw/actions/missing_tool.cjs'); await main();",
		ScriptFile:    "missing_tool.cjs",
		CustomToken:   data.SafeOutputs.MissingTool.GitHubToken,
	})
}

func (c *Compiler) buildReportIncompleteSteps(data *WorkflowData, mainJobName string) []string {
	if data.SafeOutputs.ReportIncomplete == nil {
		return nil
	}
	var envVars []string
	envVars = append(envVars, buildTemplatableIntEnvVar("GH_AW_REPORT_INCOMPLETE_MAX", data.SafeOutputs.ReportIncomplete.Max)...)
	envVars = append(envVars, buildTemplatableBoolEnvVar("GH_AW_REPORT_INCOMPLETE_CREATE_ISSUE", data.SafeOutputs.ReportIncomplete.CreateIssue)...)
	if data.SafeOutputs.ReportIncomplete.TitlePrefix != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_REPORT_INCOMPLETE_TITLE_PREFIX: %q\n", data.SafeOutputs.ReportIncomplete.TitlePrefix))
	}
	if len(data.SafeOutputs.ReportIncomplete.Labels) > 0 {
		if labelsJSON, err := json.Marshal(data.SafeOutputs.ReportIncomplete.Labels); err == nil {
			envVars = append(envVars, fmt.Sprintf("          GH_AW_REPORT_INCOMPLETE_LABELS: %q\n", string(labelsJSON)))
		}
	}
	envVars = append(envVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID, buildLocalWorkflowSourceURL(c.markdownPath))...)
	return c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Record incomplete",
		StepID:        "report_incomplete",
		MainJobName:   mainJobName,
		CustomEnvVars: envVars,
		Script:        "const { main } = require('${{ runner.temp }}/gh-aw/actions/report_incomplete_handler.cjs'); await main();",
		ScriptFile:    "report_incomplete_handler.cjs",
		CustomToken:   data.SafeOutputs.ReportIncomplete.GitHubToken,
	})
}

func serializeConclusionMessages(config *SafeOutputsConfig) string {
	if config == nil || config.Messages == nil {
		return ""
	}
	jsonValue, err := serializeMessagesConfig(config.Messages)
	if err != nil {
		notifyCommentLog.Printf("Warning: failed to serialize messages config: %v", err)
		return ""
	}
	return jsonValue
}

func (c *Compiler) buildAgentFailureEnvVars(data *WorkflowData, mainJobName string, engine CodingAgentEngine, messagesJSON string) []string {
	envVars := c.buildAgentFailureMetadataEnvVars(data, mainJobName)
	envVars = append(envVars, c.buildAgentFailureEngineEnvVars(data, mainJobName, engine)...)
	envVars = append(envVars, buildAgentFailureSafeOutputEnvVars(data, messagesJSON)...)
	envVars = append(envVars, buildAgentFailureRepoMemoryEnvVars(data)...)
	envVars = append(envVars, buildAgentFailureControlEnvVars(data)...)
	return envVars
}

func (c *Compiler) buildAgentFailureMetadataEnvVars(data *WorkflowData, mainJobName string) []string {
	envVars := buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID, buildLocalWorkflowSourceURL(c.markdownPath))
	envVars = append(envVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	envVars = append(envVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
	envVars = append(envVars, fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", data.WorkflowID))
	expiresHours := DefaultActionFailureIssueExpiresHours
	repoConfig, err := c.loadRepoConfig()
	if err != nil {
		notifyCommentLog.Printf(
			"Warning: failed to load repo config for action failure issue expiration (using default %d hours): %v. Check that %s exists and matches schema requirements",
			DefaultActionFailureIssueExpiresHours,
			err,
			RepoConfigFileName,
		)
	} else {
		expiresHours = repoConfig.ActionFailureIssueExpiresHours()
	}
	envVars = append(envVars, fmt.Sprintf("          GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS: %q\n", strconv.Itoa(expiresHours)))
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_ENGINE_ID: %q\n", data.EngineConfig.ID))
	}
	return envVars
}

func (c *Compiler) buildAgentFailureEngineEnvVars(data *WorkflowData, mainJobName string, engine CodingAgentEngine) []string {
	var envVars []string
	if EngineHasValidateSecretStep(engine, data) {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_SECRET_VERIFICATION_RESULT: ${{ needs.%s.outputs.secret_verification_result }}\n", string(constants.ActivationJobName)))
	}
	if ShouldGeneratePRCheckoutStep(data) {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_CHECKOUT_PR_SUCCESS: ${{ needs.%s.outputs.checkout_pr_success }}\n", mainJobName))
	}
	envVars = append(envVars, fmt.Sprintf("          GH_AW_EFFECTIVE_TOKENS: ${{ needs.%s.outputs.effective_tokens || '' }}\n", mainJobName))
	envVars = append(envVars, fmt.Sprintf("          GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR: ${{ needs.%s.outputs.effective_tokens_rate_limit_error || 'false' }}\n", mainJobName))
	if _, ok := engine.(*CopilotEngine); ok {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_INFERENCE_ACCESS_ERROR: ${{ needs.%s.outputs.inference_access_error }}\n", mainJobName))
		envVars = append(envVars, fmt.Sprintf("          GH_AW_MCP_POLICY_ERROR: ${{ needs.%s.outputs.mcp_policy_error }}\n", mainJobName))
		envVars = append(envVars, fmt.Sprintf("          GH_AW_AGENTIC_ENGINE_TIMEOUT: ${{ needs.%s.outputs.agentic_engine_timeout }}\n", mainJobName))
		envVars = append(envVars, fmt.Sprintf("          GH_AW_MODEL_NOT_SUPPORTED_ERROR: ${{ needs.%s.outputs.model_not_supported_error }}\n", mainJobName))
	}
	if apiHosts := getEngineAPIHosts(data, engine); len(apiHosts) > 0 {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_ENGINE_API_HOSTS: %q\n", strings.Join(apiHosts, ",")))
	}
	return envVars
}

func buildAgentFailureSafeOutputEnvVars(data *WorkflowData, messagesJSON string) []string {
	var envVars []string
	if data.SafeOutputs.AssignToAgent != nil {
		envVars = append(envVars, "          GH_AW_ASSIGNMENT_ERRORS: ${{ needs.safe_outputs.outputs.assign_to_agent_assignment_errors }}\n")
		envVars = append(envVars, "          GH_AW_ASSIGNMENT_ERROR_COUNT: ${{ needs.safe_outputs.outputs.assign_to_agent_assignment_error_count }}\n")
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		envVars = append(envVars, "          GH_AW_CREATE_DISCUSSION_ERRORS: ${{ needs.safe_outputs.outputs.create_discussion_errors }}\n")
		envVars = append(envVars, "          GH_AW_CREATE_DISCUSSION_ERROR_COUNT: ${{ needs.safe_outputs.outputs.create_discussion_error_count }}\n")
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil || data.SafeOutputs.CreatePullRequests != nil {
		envVars = append(envVars, "          GH_AW_CODE_PUSH_FAILURE_ERRORS: ${{ needs.safe_outputs.outputs.code_push_failure_errors }}\n")
		envVars = append(envVars, "          GH_AW_CODE_PUSH_FAILURE_COUNT: ${{ needs.safe_outputs.outputs.code_push_failure_count }}\n")
	}
	if data.SafeOutputs.GitHubApp != nil {
		envVars = append(envVars, "          GH_AW_SAFE_OUTPUTS_APP_TOKEN_MINTING_FAILED: ${{ needs.safe_outputs.outputs.app_token_minting_failed }}\n")
		envVars = append(envVars, "          GH_AW_CONCLUSION_APP_TOKEN_MINTING_FAILED: ${{ steps.safe-outputs-app-token.outcome == 'failure' }}\n")
	}
	if data.ActivationGitHubApp != nil {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_ACTIVATION_APP_TOKEN_MINTING_FAILED: ${{ needs.%s.outputs.activation_app_token_minting_failed }}\n", string(constants.ActivationJobName)))
	}
	envVars = append(envVars, fmt.Sprintf("          GH_AW_LOCKDOWN_CHECK_FAILED: ${{ needs.%s.outputs.lockdown_check_failed }}\n", string(constants.ActivationJobName)))
	envVars = append(envVars, fmt.Sprintf("          GH_AW_STALE_LOCK_FILE_FAILED: ${{ needs.%s.outputs.stale_lock_file_failed }}\n", string(constants.ActivationJobName)))
	if messagesJSON != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
	}
	return envVars
}

func buildAgentFailureRepoMemoryEnvVars(data *WorkflowData) []string {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return nil
	}
	envVars := []string{"          GH_AW_PUSH_REPO_MEMORY_RESULT: ${{ needs.push_repo_memory.result }}\n"}
	for _, memory := range data.RepoMemoryConfig.Memories {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_REPO_MEMORY_VALIDATION_FAILED_%s: ${{ needs.push_repo_memory.outputs.validation_failed_%s }}\n", memory.ID, memory.ID))
		envVars = append(envVars, fmt.Sprintf("          GH_AW_REPO_MEMORY_VALIDATION_ERROR_%s: ${{ needs.push_repo_memory.outputs.validation_error_%s }}\n", memory.ID, memory.ID))
		envVars = append(envVars, fmt.Sprintf("          GH_AW_REPO_MEMORY_PATCH_SIZE_EXCEEDED_%s: ${{ needs.push_repo_memory.outputs.patch_size_exceeded_%s }}\n", memory.ID, memory.ID))
	}
	return envVars
}

func buildAgentFailureControlEnvVars(data *WorkflowData) []string {
	var envVars []string
	if data.SafeOutputs.GroupReports {
		envVars = append(envVars, "          GH_AW_GROUP_REPORTS: \"true\"\n")
	} else {
		envVars = append(envVars, "          GH_AW_GROUP_REPORTS: \"false\"\n")
	}
	if data.SafeOutputs.ReportFailureAsIssue != nil && !*data.SafeOutputs.ReportFailureAsIssue {
		envVars = append(envVars, "          GH_AW_FAILURE_REPORT_AS_ISSUE: \"false\"\n")
	} else {
		envVars = append(envVars, "          GH_AW_FAILURE_REPORT_AS_ISSUE: \"true\"\n")
	}
	if data.SafeOutputs.MissingTool != nil && data.SafeOutputs.MissingTool.ReportAsFailure != nil {
		envVars = append(envVars, buildTemplatableBoolEnvVar("GH_AW_MISSING_TOOL_REPORT_AS_FAILURE", data.SafeOutputs.MissingTool.ReportAsFailure)...)
	} else {
		envVars = append(envVars, "          GH_AW_MISSING_TOOL_REPORT_AS_FAILURE: \"true\"\n")
	}
	if data.SafeOutputs.MissingData != nil && data.SafeOutputs.MissingData.ReportAsFailure != nil {
		envVars = append(envVars, buildTemplatableBoolEnvVar("GH_AW_MISSING_DATA_REPORT_AS_FAILURE", data.SafeOutputs.MissingData.ReportAsFailure)...)
	} else {
		envVars = append(envVars, "          GH_AW_MISSING_DATA_REPORT_AS_FAILURE: \"true\"\n")
	}
	if data.SafeOutputs.FailureIssueRepo != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_FAILURE_ISSUE_REPO: %q\n", data.SafeOutputs.FailureIssueRepo))
	}
	return append(envVars, buildAgentFailureBudgetEnvVars(data)...)
}

func buildAgentFailureBudgetEnvVars(data *WorkflowData) []string {
	var envVars []string
	timeoutValue := strings.TrimPrefix(data.TimeoutMinutes, "timeout-minutes: ")
	if timeoutValue != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_TIMEOUT_MINUTES: %q\n", timeoutValue))
	}
	maxEffectiveTokens := compilerenv.ResolveDefaultMaxEffectiveTokens(constants.DefaultMaxEffectiveTokens)
	if data.EngineConfig != nil && data.EngineConfig.MaxEffectiveTokens != 0 {
		maxEffectiveTokens = data.EngineConfig.MaxEffectiveTokens
	}
	envVars = append(envVars, fmt.Sprintf("          GH_AW_MAX_EFFECTIVE_TOKENS: %q\n", strconv.FormatInt(maxEffectiveTokens, 10)))
	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		envVars = append(envVars, "          GH_AW_CACHE_MEMORY_ENABLED: \"true\"\n")
	}
	return envVars
}

func (c *Compiler) buildAgentFailureStep(data *WorkflowData, mainJobName string, envVars []string) []string {
	return c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Handle agent failure",
		StepID:        "handle_agent_failure",
		MainJobName:   mainJobName,
		CustomEnvVars: envVars,
		Script:        "const { main } = require('${{ runner.temp }}/gh-aw/actions/handle_agent_failure.cjs'); await main();",
		ScriptFile:    "handle_agent_failure.cjs",
		StepCondition: "always()",
	})
}

func (c *Compiler) buildConclusionUpdateSteps(data *WorkflowData, mainJobName string, safeOutputJobNames []string, messagesJSON string) []string {
	if data.StatusComment == nil || !*data.StatusComment {
		return nil
	}
	return c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
		StepName:      "Update reaction comment with completion status",
		StepID:        "conclusion",
		MainJobName:   mainJobName,
		CustomEnvVars: c.buildConclusionEnvVars(data, mainJobName, safeOutputJobNames, messagesJSON),
		Script:        getNotifyCommentErrorScript(),
		ScriptFile:    "notify_comment_error.cjs",
		CustomToken:   getConclusionCommentToken(data),
	})
}

func (c *Compiler) buildConclusionEnvVars(data *WorkflowData, mainJobName string, safeOutputJobNames []string, messagesJSON string) []string {
	envVars := []string{
		fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName),
		fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName),
		"          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n",
		fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name),
		fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName),
	}
	if data.TrackerID != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", data.TrackerID))
	}
	if slices.Contains(safeOutputJobNames, string(constants.SafeOutputsJobName)) {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_RESULT: ${{ needs.%s.result }}\n", constants.SafeOutputsJobName))
		notifyCommentLog.Print("Added safe_outputs job result environment variable to conclusion job")
	}
	if IsDetectionJobEnabled(data.SafeOutputs) {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_DETECTION_CONCLUSION: ${{ needs.%s.outputs.detection_conclusion }}\n", constants.DetectionJobName))
		envVars = append(envVars, fmt.Sprintf("          GH_AW_DETECTION_REASON: ${{ needs.%s.outputs.detection_reason }}\n", constants.DetectionJobName))
		notifyCommentLog.Print("Added detection conclusion and reason environment variables to conclusion job")
	}
	if data.SafeOutputs.AssignToAgent != nil {
		envVars = append(envVars, "          GH_AW_ASSIGNMENT_ERROR_COUNT: ${{ needs.safe_outputs.outputs.assign_to_agent_assignment_error_count }}\n")
	}
	if messagesJSON != "" {
		envVars = append(envVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
	}
	if len(safeOutputJobNames) > 0 {
		if jobsJSON, jobURLEnvVars := buildSafeOutputJobsEnvVars(safeOutputJobNames); jobsJSON != "" {
			envVars = append(envVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_JOBS: %q\n", jobsJSON))
			envVars = append(envVars, jobURLEnvVars...)
			notifyCommentLog.Printf("Added safe output jobs info for %d job(s)", len(safeOutputJobNames))
		}
	}
	return envVars
}

func getConclusionCommentToken(data *WorkflowData) string {
	if data.SafeOutputs == nil || data.SafeOutputs.AddComments == nil {
		return ""
	}
	return data.SafeOutputs.AddComments.GitHubToken
}

func buildConclusionCondition(safeOutputJobNames []string) ConditionNode {
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	lockdownCheckFailed := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.lockdown_check_failed", string(constants.ActivationJobName))),
		BuildStringLiteral("true"),
	)
	staleLockFileFailed := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.stale_lock_file_failed", string(constants.ActivationJobName))),
		BuildStringLiteral("true"),
	)
	condition := BuildAnd(
		BuildFunctionCall("always"),
		BuildOr(BuildOr(agentNotSkipped, lockdownCheckFailed), staleLockFileFailed),
	)
	if slices.Contains(safeOutputJobNames, "add_comment") {
		return BuildAnd(condition, &NotNode{Child: BuildPropertyAccess("needs.add_comment.outputs.comment_id")})
	}
	return condition
}

func buildConclusionNeeds(data *WorkflowData, mainJobName string, safeOutputJobNames []string) []string {
	needs := []string{mainJobName, string(constants.ActivationJobName)}
	needs = append(needs, safeOutputJobNames...)
	if IsDetectionJobEnabled(data.SafeOutputs) {
		needs = append(needs, string(constants.DetectionJobName))
		notifyCommentLog.Print("Added detection job dependency to conclusion job")
	}
	notifyCommentLog.Printf("Job built successfully: dependencies_count=%d", len(needs))
	return needs
}

func buildConclusionOutputs(data *WorkflowData) map[string]string {
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
	return outputs
}

func (c *Compiler) buildConclusionConcurrency(data *WorkflowData) string {
	if data.WorkflowID == "" {
		return ""
	}
	group := "gh-aw-conclusion-" + data.WorkflowID
	if data.ConcurrencyJobDiscriminator != "" {
		notifyCommentLog.Printf("Appending job discriminator to conclusion job concurrency group: %s", data.ConcurrencyJobDiscriminator)
		group = fmt.Sprintf("%s-%s", group, data.ConcurrencyJobDiscriminator)
	}
	concurrencyValue := fmt.Sprintf("concurrency:\n  group: %q\n  cancel-in-progress: false", group)
	if isGroupConcurrencyQueueEnabled(data) {
		concurrencyValue += "\n  queue: max"
	}
	notifyCommentLog.Printf("Configuring conclusion job concurrency group: %s", group)
	return c.indentYAMLLines(concurrencyValue, "    ")
}

func (c *Compiler) buildConclusionFinalSteps(data *WorkflowData) []string {
	var steps []string
	if data.SafeOutputs.GitHubApp != nil {
		notifyCommentLog.Print("Adding GitHub App token invalidation step to conclusion job")
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}
	return steps
}

// isGroupConcurrencyQueueEnabled reports whether compiler-generated concurrency groups
// should include queue: max. The feature is enabled by default and can be disabled
// with features.group-concurrency-queue: false.
func isGroupConcurrencyQueueEnabled(data *WorkflowData) bool {
	flag := strings.ToLower(strings.TrimSpace(string(constants.GroupConcurrencyQueueFeatureFlag)))
	if data != nil && data.Features != nil {
		for key, value := range data.Features {
			if strings.ToLower(key) == flag {
				return parseGroupConcurrencyQueueFeatureValue(value)
			}
		}
	}
	return true
}

func parseGroupConcurrencyQueueFeatureValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		switch normalized {
		case "false", "0", "off", "no":
			return false
		default:
			return true
		}
	default:
		return true
	}
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

// getEngineAPIHosts returns the primary AI inference API hostnames for the given engine and
// workflow data. These are the hosts that appear in the firewall audit log when the engine
// makes authenticated API calls. The returned slice is used to populate GH_AW_ENGINE_API_HOSTS
// so the failure handler can detect credential authentication rejections without relying solely
// on hardcoded host patterns.
//
// Resolution order (per engine):
//   - engine.api-target (explicit GHES / enterprise override) takes precedence
//   - Default public API hostname(s) for the engine
func getEngineAPIHosts(data *WorkflowData, engine CodingAgentEngine) []string {
	if engine == nil {
		return nil
	}

	// Explicit api-target overrides the engine-specific default for all engine types.
	if data != nil && data.EngineConfig != nil && data.EngineConfig.APITarget != "" {
		return []string{data.EngineConfig.APITarget}
	}

	switch engine.(type) {
	case *CopilotEngine:
		// Return the full set of known Copilot inference endpoints so that any variant
		// (enterprise, business, individual, or the routing hub) is covered.
		return []string{
			"api.enterprise.githubcopilot.com",
			"api.githubcopilot.com",
			"api.business.githubcopilot.com",
			"api.individual.githubcopilot.com",
		}
	case *ClaudeEngine:
		return []string{"api.anthropic.com"}
	case *CodexEngine:
		return []string{"api.openai.com"}
	case *GeminiEngine:
		return []string{DefaultGeminiAPITarget}
	case *AntigravityEngine:
		return []string{DefaultAntigravityAPITarget}
	default:
		// Custom or unknown engine — no known API hosts without explicit api-target.
		return nil
	}
}
