package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var unifiedSafeOutputsLog = logger.New("workflow:unified_safe_outputs")

// buildUnifiedSafeOutputsJob builds a single job with a single step that processes all safe output types.
// This is the refactored version that uses safe_outputs_processor_main.cjs with the handler manager.
//
// The unified processor:
// 1. Loads agent output once
// 2. Dispatches each message to the appropriate registered handler
// 3. Collects temporary IDs and makes them available to subsequent handlers
// 4. Sets all required outputs
func (c *Compiler) buildUnifiedSafeOutputsJob(data *WorkflowData, mainJobName, markdownPath string) (*Job, []string, error) {
	if data.SafeOutputs == nil {
		unifiedSafeOutputsLog.Print("No safe outputs configured, skipping unified job")
		return nil, nil, nil
	}

	unifiedSafeOutputsLog.Print("Building unified safe outputs job with single processor step")

	var steps []string
	var outputs = make(map[string]string)
	var permissions = NewPermissions()

	// Track whether threat detection job is enabled for step conditions
	threatDetectionEnabled := data.SafeOutputs.ThreatDetection != nil

	// Collect all permissions needed by any enabled safe output type
	if data.SafeOutputs.CreateIssues != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		permissions.Merge(NewPermissionsContentsWriteIssuesWritePRWrite())
	}
	if data.SafeOutputs.AddComments != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite())
	}
	if data.SafeOutputs.CloseDiscussions != nil {
		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}
	if data.SafeOutputs.CloseIssues != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.ClosePullRequests != nil {
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		permissions.Merge(NewPermissionsContentsReadSecurityEventsWrite())
	}
	if data.SafeOutputs.AddLabels != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.AddReviewer != nil {
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if data.SafeOutputs.AssignMilestone != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.AssignToAgent != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.AssignToUser != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.UpdateIssues != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.UpdatePullRequests != nil {
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		permissions.Merge(NewPermissionsContentsWriteIssuesWritePRWrite())
	}
	if data.SafeOutputs.UploadAssets != nil {
		permissions.Merge(NewPermissionsContentsWrite())
	}
	if data.SafeOutputs.UpdateRelease != nil {
		permissions.Merge(NewPermissionsContentsWrite())
	}
	if data.SafeOutputs.LinkSubIssue != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.HideComment != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.CreateAgentTasks != nil {
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if data.SafeOutputs.UpdateProjects != nil {
		permissions.Merge(NewPermissionsContentsReadProjectsWrite())
	}

	// Add GitHub App token minting step if app is configured
	if data.SafeOutputs.App != nil {
		unifiedSafeOutputsLog.Print("Adding GitHub App token minting step")
	}

	// Add artifact download steps once at the beginning
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Collect all script files needed by the unified processor
	// The processor needs: safe_outputs_processor_main, handler_manager, create_issue_handler, and all their dependencies
	scriptNames := []string{
		"safe_outputs_processor_main",
		"safe_output_handler_manager",
		"create_issue_handler",
	}

	sources := GetJavaScriptSources()
	scriptFilesResult, err := CollectAllJobScriptFiles(scriptNames, sources)
	if err != nil {
		unifiedSafeOutputsLog.Printf("Failed to collect script files: %v, falling back to inline mode", err)
		scriptFilesResult = nil
	} else {
		unifiedSafeOutputsLog.Printf("File mode: collected %d files, %d bytes total",
			len(scriptFilesResult.Files), scriptFilesResult.TotalSize)
	}

	// Add JavaScript files setup step if using file mode
	if scriptFilesResult != nil && len(scriptFilesResult.Files) > 0 {
		preparedFiles := PrepareFilesForFileMode(scriptFilesResult.Files)
		setupSteps := GenerateWriteScriptsStep(preparedFiles)
		steps = append(steps, setupSteps...)
		unifiedSafeOutputsLog.Printf("Added setup_scripts step with %d files", len(preparedFiles))
	}

	// Add GitHub App token minting step after artifact download but before main processing
	if data.SafeOutputs.App != nil {
		appTokenSteps := c.buildGitHubAppTokenMintStep(data.SafeOutputs.App, permissions)
		steps = append(steps, appTokenSteps...)
	}

	// Build the unified processor step
	// This single step replaces all the individual safe output steps
	stepScript := sources["safe_outputs_processor_main.cjs"]
	if stepScript == "" {
		return nil, nil, fmt.Errorf("safe_outputs_processor_main.cjs script not found")
	}

	// Collect all custom environment variables from all safe output configs
	var customEnvVars []string

	// Add environment variables for each enabled safe output type
	// These mirror what the individual steps would have had
	if data.SafeOutputs.CreateIssues != nil {
		customEnvVars = append(customEnvVars, c.buildCreateIssueEnvVars(data.SafeOutputs.CreateIssues)...)
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		customEnvVars = append(customEnvVars, c.buildCreateDiscussionEnvVars(data.SafeOutputs.CreateDiscussions)...)
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		customEnvVars = append(customEnvVars, c.buildCreatePullRequestEnvVars(data.SafeOutputs.CreatePullRequests)...)
	}
	// Add other env vars as needed...

	// Build the step using inline mode (for now)
	processorStepConfig := GitHubScriptStepConfig{
		StepName:      "Process All Safe Outputs",
		StepID:        "process_safe_outputs",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        stepScript,
		// Token will be determined by buildGitHubScriptStep
	}

	processorSteps := c.buildGitHubScriptStep(data, processorStepConfig)
	steps = append(steps, processorSteps...)

	// Add all outputs from the unified processor
	// The processor sets these outputs which match what individual steps would have set
	outputs["issue_number"] = "${{ steps.process_safe_outputs.outputs.issue_number }}"
	outputs["issue_url"] = "${{ steps.process_safe_outputs.outputs.issue_url }}"
	outputs["temporary_id_map"] = "${{ steps.process_safe_outputs.outputs.temporary_id_map }}"
	outputs["pull_request_number"] = "${{ steps.process_safe_outputs.outputs.pull_request_number }}"
	outputs["pull_request_url"] = "${{ steps.process_safe_outputs.outputs.pull_request_url }}"
	outputs["discussion_number"] = "${{ steps.process_safe_outputs.outputs.discussion_number }}"
	outputs["discussion_url"] = "${{ steps.process_safe_outputs.outputs.discussion_url }}"
	outputs["issues_to_assign_copilot"] = "${{ steps.process_safe_outputs.outputs.issues_to_assign_copilot }}"
	outputs["comment_id"] = "${{ steps.process_safe_outputs.outputs.comment_id }}"
	outputs["comment_url"] = "${{ steps.process_safe_outputs.outputs.comment_url }}"

	// Add GitHub App token invalidation step at the end if app is configured
	if data.SafeOutputs.App != nil {
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}

	// Build the job condition
	agentNotSkipped := BuildAnd(
		&NotNode{Child: BuildFunctionCall("cancelled")},
		BuildNotEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
			BuildStringLiteral("skipped"),
		),
	)

	jobCondition := agentNotSkipped
	if threatDetectionEnabled {
		jobCondition = BuildAnd(agentNotSkipped, buildDetectionSuccessCondition())
	}

	// Build dependencies
	needs := []string{mainJobName}
	if threatDetectionEnabled {
		needs = append(needs, constants.DetectionJobName)
	}
	// Add activation job dependency for jobs that need it
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		needs = append(needs, constants.ActivationJobName)
	}

	job := &Job{
		Name:           "safe_outputs",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions.RenderToYAML(),
		TimeoutMinutes: 15,
		Steps:          steps,
		Outputs:        outputs,
		Needs:          needs,
	}

	unifiedSafeOutputsLog.Printf("Built unified safe outputs job with single processor step")

	// Return step names for compatibility
	stepNames := []string{"process_safe_outputs"}
	return job, stepNames, nil
}

// Helper functions to build environment variables for each safe output type
// These use the existing helpers from safe_outputs_env_helpers.go

func (c *Compiler) buildCreateIssueEnvVars(cfg *CreateIssuesConfig) []string {
	var envVars []string

	envVars = append(envVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", cfg.TitlePrefix)...)
	envVars = append(envVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", cfg.Labels)...)
	envVars = append(envVars, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", cfg.AllowedLabels)...)
	envVars = append(envVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", cfg.AllowedRepos)...)

	if cfg.Expires != 0 {
		envVars = append(envVars, fmt.Sprintf("GH_AW_ISSUE_EXPIRES=%d", cfg.Expires))
	}

	// Handle assignees - check for "copilot" assignee
	for _, assignee := range cfg.Assignees {
		if assignee == "copilot" || assignee == "github-copilot" {
			envVars = append(envVars, "GH_AW_ASSIGN_COPILOT=true")
			break
		}
	}

	return envVars
}

func (c *Compiler) buildCreateDiscussionEnvVars(cfg *CreateDiscussionsConfig) []string {
	var envVars []string

	envVars = append(envVars, buildCategoryEnvVar("GH_AW_DISCUSSION_CATEGORY", cfg.Category)...)
	envVars = append(envVars, buildTitlePrefixEnvVar("GH_AW_DISCUSSION_TITLE_PREFIX", cfg.TitlePrefix)...)
	envVars = append(envVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", cfg.AllowedRepos)...)

	if cfg.Expires != 0 {
		envVars = append(envVars, fmt.Sprintf("GH_AW_DISCUSSION_EXPIRES=%d", cfg.Expires))
	}

	if cfg.CloseOlderDiscussions {
		envVars = append(envVars, "GH_AW_CLOSE_OLDER_DISCUSSIONS=true")
	}

	return envVars
}

func (c *Compiler) buildCreatePullRequestEnvVars(cfg *CreatePullRequestsConfig) []string {
	var envVars []string

	envVars = append(envVars, buildLabelsEnvVar("GH_AW_PR_LABELS", cfg.Labels)...)
	envVars = append(envVars, buildTitlePrefixEnvVar("GH_AW_PR_TITLE_PREFIX", cfg.TitlePrefix)...)

	if cfg.Draft != nil {
		if *cfg.Draft {
			envVars = append(envVars, "GH_AW_PR_DRAFT=true")
		} else {
			envVars = append(envVars, "GH_AW_PR_DRAFT=false")
		}
	}

	if cfg.Expires != 0 {
		envVars = append(envVars, fmt.Sprintf("GH_AW_PR_EXPIRES=%d", cfg.Expires))
	}

	if cfg.AllowEmpty {
		envVars = append(envVars, "GH_AW_PR_ALLOW_EMPTY=true")
	}
	if cfg.IfNoChanges != "" {
		envVars = append(envVars, fmt.Sprintf("GH_AW_PR_IF_NO_CHANGES=%s", cfg.IfNoChanges))
	}

	return envVars
}
