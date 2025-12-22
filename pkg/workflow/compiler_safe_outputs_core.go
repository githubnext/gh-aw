package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var consolidatedSafeOutputsLog = logger.New("workflow:compiler_safe_outputs_consolidated")

// SafeOutputStepConfig holds configuration for building a single safe output step
// within the consolidated safe-outputs job
type SafeOutputStepConfig struct {
	StepName        string            // Human-readable step name (e.g., "Create Issue")
	StepID          string            // Step ID for referencing outputs (e.g., "create_issue")
	Script          string            // JavaScript script to execute (for inline mode)
	ScriptName      string            // Name of the script in the registry (for file mode)
	CustomEnvVars   []string          // Environment variables specific to this step
	Condition       ConditionNode     // Step-level condition (if clause)
	Token           string            // GitHub token for this step
	UseCopilotToken bool              // Whether to use Copilot token preference chain
	UseAgentToken   bool              // Whether to use agent token preference chain
	PreSteps        []string          // Optional steps to run before the script step
	PostSteps       []string          // Optional steps to run after the script step
	Outputs         map[string]string // Outputs from this step
}

// buildConsolidatedSafeOutputsJob builds a single job containing all safe output operations
// as separate steps within that job. This reduces the number of jobs in the workflow
// while maintaining observability through distinct step names, IDs, and outputs.
//
// File mode: Instead of inlining bundled JavaScript in YAML, this function:
// 1. Collects all JavaScript files needed by enabled safe outputs
// 2. Generates a "Setup JavaScript files" step to write them to /tmp/gh-aw/scripts/
// 3. Each safe output step requires from the local filesystem
func (c *Compiler) buildConsolidatedSafeOutputsJob(data *WorkflowData, mainJobName, markdownPath string) (*Job, []string, error) {
	if data.SafeOutputs == nil {
		consolidatedSafeOutputsLog.Print("No safe outputs configured, skipping consolidated job")
		return nil, nil, nil
	}

	consolidatedSafeOutputsLog.Print("Building consolidated safe outputs job with file mode")

	var steps []string
	var outputs = make(map[string]string)
	var permissions = NewPermissions()
	var safeOutputStepNames []string

	// Track whether threat detection job is enabled for step conditions
	threatDetectionEnabled := data.SafeOutputs.ThreatDetection != nil

	// Track which outputs are created for dependency tracking
	var createIssueEnabled bool
	var createDiscussionEnabled bool
	var createPullRequestEnabled bool

	// Collect all script names that will be used in this job
	var scriptNames []string
	if data.SafeOutputs.CreateIssues != nil {
		scriptNames = append(scriptNames, "create_issue")
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		scriptNames = append(scriptNames, "create_discussion")
	}
	if data.SafeOutputs.UpdateDiscussions != nil {
		scriptNames = append(scriptNames, "update_discussion")
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		scriptNames = append(scriptNames, "create_pull_request")
	}
	if data.SafeOutputs.AddComments != nil {
		scriptNames = append(scriptNames, "add_comment")
	}
	if data.SafeOutputs.CloseDiscussions != nil {
		scriptNames = append(scriptNames, "close_discussion")
	}
	if data.SafeOutputs.CloseIssues != nil {
		scriptNames = append(scriptNames, "close_issue")
	}
	if data.SafeOutputs.ClosePullRequests != nil {
		scriptNames = append(scriptNames, "close_pull_request")
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		scriptNames = append(scriptNames, "create_pr_review_comment")
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		scriptNames = append(scriptNames, "create_code_scanning_alert")
	}
	if data.SafeOutputs.AddLabels != nil {
		scriptNames = append(scriptNames, "add_labels")
	}
	if data.SafeOutputs.AddReviewer != nil {
		scriptNames = append(scriptNames, "add_reviewer")
	}
	if data.SafeOutputs.AssignMilestone != nil {
		scriptNames = append(scriptNames, "assign_milestone")
	}
	if data.SafeOutputs.AssignToAgent != nil {
		scriptNames = append(scriptNames, "assign_to_agent")
	}
	if data.SafeOutputs.AssignToUser != nil {
		scriptNames = append(scriptNames, "assign_to_user")
	}
	if data.SafeOutputs.UpdateIssues != nil {
		scriptNames = append(scriptNames, "update_issue")
	}
	if data.SafeOutputs.UpdatePullRequests != nil {
		scriptNames = append(scriptNames, "update_pull_request")
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		scriptNames = append(scriptNames, "push_to_pull_request_branch")
	}
	if data.SafeOutputs.UploadAssets != nil {
		scriptNames = append(scriptNames, "upload_assets")
	}
	if data.SafeOutputs.UpdateRelease != nil {
		scriptNames = append(scriptNames, "update_release")
	}
	if data.SafeOutputs.LinkSubIssue != nil {
		scriptNames = append(scriptNames, "link_sub_issue")
	}
	if data.SafeOutputs.HideComment != nil {
		scriptNames = append(scriptNames, "hide_comment")
	}
	// create_agent_task is handled separately through its direct source
	if data.SafeOutputs.UpdateProjects != nil {
		scriptNames = append(scriptNames, "update_project")
	}

	// Collect all JavaScript files for file mode
	var scriptFilesResult *ScriptFilesResult
	if len(scriptNames) > 0 {
		sources := GetJavaScriptSources()
		var err error
		scriptFilesResult, err = CollectAllJobScriptFiles(scriptNames, sources)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Failed to collect script files: %v, falling back to inline mode", err)
			scriptFilesResult = nil
		} else {
			consolidatedSafeOutputsLog.Printf("File mode: collected %d files, %d bytes total",
				len(scriptFilesResult.Files), scriptFilesResult.TotalSize)
		}
	}

	// Add GitHub App token minting step if app is configured
	if data.SafeOutputs.App != nil {
		consolidatedSafeOutputsLog.Print("Adding GitHub App token minting step")
		// We'll compute permissions after collecting all step requirements
	}

	// Add artifact download steps once at the beginning
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Add patch artifact download if create-pull-request or push-to-pull-request-branch is enabled
	// Both of these safe outputs require the patch file to apply changes
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		consolidatedSafeOutputsLog.Print("Adding patch artifact download for create-pull-request or push-to-pull-request-branch")
		patchDownloadSteps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
			ArtifactName: "aw.patch",
			DownloadPath: "/tmp/gh-aw/",
			SetupEnvStep: false, // No environment variable needed, the script checks the file directly
			StepName:     "Download patch artifact",
		})
		steps = append(steps, patchDownloadSteps...)
	}

	// Add JavaScript files setup step if using file mode
	if scriptFilesResult != nil && len(scriptFilesResult.Files) > 0 {
		// Prepare files with rewritten require paths
		preparedFiles := PrepareFilesForFileMode(scriptFilesResult.Files)
		setupSteps := GenerateWriteScriptsStep(preparedFiles)
		steps = append(steps, setupSteps...)
		consolidatedSafeOutputsLog.Printf("Added setup_scripts step with %d files", len(preparedFiles))
	}

	// === Build individual safe output steps ===

	// 1. Create Issue step
	if data.SafeOutputs.CreateIssues != nil {
		createIssueEnabled = true
		stepConfig := c.buildCreateIssueStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		// Add outputs
		outputs["create_issue_issue_number"] = "${{ steps.create_issue.outputs.issue_number }}"
		outputs["create_issue_issue_url"] = "${{ steps.create_issue.outputs.issue_url }}"
		outputs["create_issue_temporary_id_map"] = "${{ steps.create_issue.outputs.temporary_id_map }}"

		// Merge permissions
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}

	// 2. Create Discussion step
	if data.SafeOutputs.CreateDiscussions != nil {
		createDiscussionEnabled = true
		stepConfig := c.buildCreateDiscussionStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["create_discussion_discussion_number"] = "${{ steps.create_discussion.outputs.discussion_number }}"
		outputs["create_discussion_discussion_url"] = "${{ steps.create_discussion.outputs.discussion_url }}"

		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}

	// 2a. Update Discussion step
	if data.SafeOutputs.UpdateDiscussions != nil {
		stepConfig := c.buildUpdateDiscussionStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["update_discussion_discussion_number"] = "${{ steps.update_discussion.outputs.discussion_number }}"
		outputs["update_discussion_discussion_url"] = "${{ steps.update_discussion.outputs.discussion_url }}"

		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}

	// 3. Create Pull Request step
	if data.SafeOutputs.CreatePullRequests != nil {
		createPullRequestEnabled = true
		stepConfig := c.buildCreatePullRequestStepConfig(data, mainJobName, threatDetectionEnabled)
		// Add pre-steps (checkout, git config, etc.)
		steps = append(steps, stepConfig.PreSteps...)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		steps = append(steps, stepConfig.PostSteps...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["create_pull_request_pull_request_number"] = "${{ steps.create_pull_request.outputs.pull_request_number }}"
		outputs["create_pull_request_pull_request_url"] = "${{ steps.create_pull_request.outputs.pull_request_url }}"

		permissions.Merge(NewPermissionsContentsWriteIssuesWritePRWrite())
	}

	// 4. Add Comment step (needs to come after create_issue, create_discussion, create_pull_request)
	if data.SafeOutputs.AddComments != nil {
		stepConfig := c.buildAddCommentStepConfig(data, mainJobName, threatDetectionEnabled,
			createIssueEnabled, createDiscussionEnabled, createPullRequestEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["add_comment_comment_id"] = "${{ steps.add_comment.outputs.comment_id }}"
		outputs["add_comment_comment_url"] = "${{ steps.add_comment.outputs.comment_url }}"

		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite())
	}

	// 5. Close Discussion step
	if data.SafeOutputs.CloseDiscussions != nil {
		stepConfig := c.buildCloseDiscussionStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}

	// 6. Close Issue step
	if data.SafeOutputs.CloseIssues != nil {
		stepConfig := c.buildCloseIssueStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}

	// 7. Close Pull Request step
	if data.SafeOutputs.ClosePullRequests != nil {
		stepConfig := c.buildClosePullRequestStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}

	// 8. Create PR Review Comment step
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		stepConfig := c.buildCreatePRReviewCommentStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}

	// 9. Create Code Scanning Alert step
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		workflowFilename := GetWorkflowIDFromPath(markdownPath)
		stepConfig := c.buildCreateCodeScanningAlertStepConfig(data, mainJobName, threatDetectionEnabled, workflowFilename)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadSecurityEventsWrite())
	}

	// 10. Add Labels step
	if data.SafeOutputs.AddLabels != nil {
		stepConfig := c.buildAddLabelsStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["add_labels_labels_added"] = "${{ steps.add_labels.outputs.labels_added }}"

		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWrite())
	}

	// 11. Add Reviewer step
	if data.SafeOutputs.AddReviewer != nil {
		stepConfig := c.buildAddReviewerStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["add_reviewer_reviewers_added"] = "${{ steps.add_reviewer.outputs.reviewers_added }}"

		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}

	// 12. Assign Milestone step
	if data.SafeOutputs.AssignMilestone != nil {
		stepConfig := c.buildAssignMilestoneStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["assign_milestone_milestone_assigned"] = "${{ steps.assign_milestone.outputs.milestone_assigned }}"

		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWrite())
	}

	// 13. Assign To Agent step
	if data.SafeOutputs.AssignToAgent != nil {
		stepConfig := c.buildAssignToAgentStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["assign_to_agent_assigned"] = "${{ steps.assign_to_agent.outputs.assigned }}"

		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}

	// 14. Assign To User step
	if data.SafeOutputs.AssignToUser != nil {
		stepConfig := c.buildAssignToUserStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["assign_to_user_assigned"] = "${{ steps.assign_to_user.outputs.assigned }}"

		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWrite())
	}

	// 15. Update Issue step
	if data.SafeOutputs.UpdateIssues != nil {
		stepConfig := c.buildUpdateIssueStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}

	// 16. Update Pull Request step
	if data.SafeOutputs.UpdatePullRequests != nil {
		stepConfig := c.buildUpdatePullRequestStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}

	// 17. Push To Pull Request Branch step
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		stepConfig := c.buildPushToPullRequestBranchStepConfig(data, mainJobName, threatDetectionEnabled)
		// Add pre-steps (checkout, git config, etc.)
		steps = append(steps, stepConfig.PreSteps...)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["push_to_pull_request_branch_commit_url"] = "${{ steps.push_to_pull_request_branch.outputs.commit_url }}"

		permissions.Merge(NewPermissionsContentsWriteIssuesWritePRWrite())
	}

	// 18. Upload Assets step
	if data.SafeOutputs.UploadAssets != nil {
		stepConfig := c.buildUploadAssetsStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsWrite())
	}

	// 19. Update Release step
	if data.SafeOutputs.UpdateRelease != nil {
		stepConfig := c.buildUpdateReleaseStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsWrite())
	}

	// 20. Link Sub Issue step
	if data.SafeOutputs.LinkSubIssue != nil {
		stepConfig := c.buildLinkSubIssueStepConfig(data, mainJobName, threatDetectionEnabled, createIssueEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}

	// 21. Hide Comment step
	if data.SafeOutputs.HideComment != nil {
		stepConfig := c.buildHideCommentStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite())
	}

	// 22. Create Agent Task step
	if data.SafeOutputs.CreateAgentTasks != nil {
		stepConfig := c.buildCreateAgentTaskStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["create_agent_task_task_number"] = "${{ steps.create_agent_task.outputs.task_number }}"
		outputs["create_agent_task_task_url"] = "${{ steps.create_agent_task.outputs.task_url }}"

		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}

	// 23. Update Project step
	if data.SafeOutputs.UpdateProjects != nil {
		stepConfig := c.buildUpdateProjectStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		// Update project requires organization-projects permission (via GitHub App token)
		// Note: Projects v2 cannot use GITHUB_TOKEN; it requires a PAT or GitHub App token
		permissions.Merge(NewPermissionsContentsReadProjectsWrite())
	}

	// If no steps were added, return nil
	if len(safeOutputStepNames) == 0 {
		consolidatedSafeOutputsLog.Print("No safe output steps were added")
		return nil, nil, nil
	}

	// Add GitHub App token minting step at the beginning if app is configured
	if data.SafeOutputs.App != nil {
		appTokenSteps := c.buildGitHubAppTokenMintStep(data.SafeOutputs.App, permissions)
		// Prepend app token steps (after artifact download but before safe output steps)
		insertIndex := len(buildAgentOutputDownloadSteps())
		newSteps := make([]string, 0)
		newSteps = append(newSteps, steps[:insertIndex]...)
		newSteps = append(newSteps, appTokenSteps...)
		newSteps = append(newSteps, steps[insertIndex:]...)
		steps = newSteps
	}

	// Add GitHub App token invalidation step at the end if app is configured
	if data.SafeOutputs.App != nil {
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}

	// Build the job condition
	// The job should run if agent job completed (not skipped) AND detection passed (if enabled)
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
	// Add activation job dependency for jobs that need it (create_pull_request, push_to_pull_request_branch)
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		needs = append(needs, constants.ActivationJobName)
	}

	// Extract workflow ID from markdown path for GH_AW_WORKFLOW_ID
	workflowID := GetWorkflowIDFromPath(markdownPath)

	// Build job-level environment variables that are common to all safe output steps
	jobEnv := c.buildJobLevelSafeOutputEnvVars(data, workflowID)

	job := &Job{
		Name:           "safe_outputs",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions.RenderToYAML(),
		TimeoutMinutes: 15, // Slightly longer timeout for consolidated job with multiple steps
		Env:            jobEnv,
		Steps:          steps,
		Outputs:        outputs,
		Needs:          needs,
	}

	consolidatedSafeOutputsLog.Printf("Built consolidated safe outputs job with %d steps", len(safeOutputStepNames))

	return job, safeOutputStepNames, nil
}

// buildConsolidatedSafeOutputStep builds a single safe output step within the consolidated job
func (c *Compiler) buildConsolidatedSafeOutputStep(data *WorkflowData, config SafeOutputStepConfig) []string {
	var steps []string

	// Build step condition if provided
	var conditionStr string
	if config.Condition != nil {
		conditionStr = config.Condition.Render()
	}

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	if conditionStr != "" {
		steps = append(steps, fmt.Sprintf("        if: %s\n", conditionStr))
	}
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables section
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")
	steps = append(steps, config.CustomEnvVars...)

	// Add custom safe output env vars
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	if config.UseAgentToken {
		c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, config.Token)
	} else if config.UseCopilotToken {
		c.addSafeOutputCopilotGitHubTokenForConfig(&steps, data, config.Token)
	} else {
		c.addSafeOutputGitHubTokenForConfig(&steps, data, config.Token)
	}

	steps = append(steps, "          script: |\n")

	// Add the formatted JavaScript script
	// Use file mode if ScriptName is set, otherwise inline the bundled script
	if config.ScriptName != "" {
		// File mode: inline the main script with requires transformed to absolute paths
		// The script is inlined (not required) so it runs in the GitHub Script context
		// with access to github, context, core, exec, io globals
		inlinedScript, err := GetInlinedScriptForFileMode(config.ScriptName)
		if err != nil {
			// Fall back to require() mode if script not found in registry
			consolidatedSafeOutputsLog.Printf("Script %s not in registry, using require: %v", config.ScriptName, err)
			requireScript := GenerateRequireScript(config.ScriptName + ".cjs")
			formattedScript := FormatJavaScriptForYAML(requireScript)
			steps = append(steps, formattedScript...)
		} else {
			formattedScript := FormatJavaScriptForYAML(inlinedScript)
			steps = append(steps, formattedScript...)
		}
	} else {
		// Inline mode: embed the bundled script directly
		formattedScript := FormatJavaScriptForYAML(config.Script)
		steps = append(steps, formattedScript...)
	}

	return steps
}

// buildJobLevelSafeOutputEnvVars builds environment variables that should be set at the job level
// for the consolidated safe_outputs job. These are variables that are common to all safe output steps.
func (c *Compiler) buildJobLevelSafeOutputEnvVars(data *WorkflowData, workflowID string) map[string]string {
	envVars := make(map[string]string)

	// Set GH_AW_WORKFLOW_ID to the workflow ID (filename without extension)
	// This is used for branch naming in create_pull_request and other operations
	envVars["GH_AW_WORKFLOW_ID"] = fmt.Sprintf("%q", workflowID)

	// Add workflow metadata that's common to all steps
	envVars["GH_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)

	if data.Source != "" {
		envVars["GH_AW_WORKFLOW_SOURCE"] = fmt.Sprintf("%q", data.Source)
		sourceURL := buildSourceURL(data.Source)
		if sourceURL != "" {
			envVars["GH_AW_WORKFLOW_SOURCE_URL"] = fmt.Sprintf("%q", sourceURL)
		}
	}

	if data.TrackerID != "" {
		envVars["GH_AW_TRACKER_ID"] = fmt.Sprintf("%q", data.TrackerID)
	}

	// Add engine metadata that's common to all steps
	if data.EngineConfig != nil {
		if data.EngineConfig.ID != "" {
			envVars["GH_AW_ENGINE_ID"] = fmt.Sprintf("%q", data.EngineConfig.ID)
		}
		if data.EngineConfig.Version != "" {
			envVars["GH_AW_ENGINE_VERSION"] = fmt.Sprintf("%q", data.EngineConfig.Version)
		}
		if data.EngineConfig.Model != "" {
			envVars["GH_AW_ENGINE_MODEL"] = fmt.Sprintf("%q", data.EngineConfig.Model)
		}
	}

	// Add safe output job environment variables (staged/target repo)
	if c.trialMode || data.SafeOutputs.Staged {
		envVars["GH_AW_SAFE_OUTPUTS_STAGED"] = "\"true\""
	}

	// Set GH_AW_TARGET_REPO_SLUG - prefer trial target repo (applies to all steps)
	// Note: Individual steps with target-repo config will override this in their step-level env
	if c.trialMode && c.trialLogicalRepoSlug != "" {
		envVars["GH_AW_TARGET_REPO_SLUG"] = fmt.Sprintf("%q", c.trialLogicalRepoSlug)
	}

	// Add messages config if present (applies to all steps)
	if data.SafeOutputs.Messages != nil {
		messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Warning: failed to serialize messages config: %v", err)
		} else if messagesJSON != "" {
			envVars["GH_AW_SAFE_OUTPUT_MESSAGES"] = fmt.Sprintf("%q", messagesJSON)
		}
	}

	// Add asset upload configuration if present (applies to all steps)
	if data.SafeOutputs.UploadAssets != nil {
		envVars["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
		envVars["GH_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", data.SafeOutputs.UploadAssets.MaxSizeKB)
		envVars["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
	}

	return envVars
}

// buildDetectionSuccessCondition builds the condition to check if detection passed
func buildDetectionSuccessCondition() ConditionNode {
	return BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.success", constants.DetectionJobName)),
		BuildStringLiteral("true"),
	)
}
