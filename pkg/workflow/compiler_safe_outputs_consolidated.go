package workflow

import (
	"fmt"
	"path/filepath"
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
		workflowFilename := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
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
		newSteps := make([]string, 0, len(steps)+len(appTokenSteps))
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

	job := &Job{
		Name:           "safe_outputs",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions.RenderToYAML(),
		TimeoutMinutes: 15, // Slightly longer timeout for consolidated job with multiple steps
		Env: map[string]string{
			"GH_AW_WORKFLOW_ID": fmt.Sprintf("%q", mainJobName),
		},
		Steps:   steps,
		Outputs: outputs,
		Needs:   needs,
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

// buildDetectionSuccessCondition builds the condition to check if detection passed
func buildDetectionSuccessCondition() ConditionNode {
	return BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.success", constants.DetectionJobName)),
		BuildStringLiteral("true"),
	)
}

// === Step Config Builders ===
// These functions build the SafeOutputStepConfig for each safe output type

func (c *Compiler) buildCreateIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateIssues

	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", cfg.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", cfg.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", cfg.AllowedLabels)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", cfg.AllowedRepos)...)
	if cfg.Expires > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ISSUE_EXPIRES: \"%d\"\n", cfg.Expires))
	}
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("create_issue")

	return SafeOutputStepConfig{
		StepName:      "Create Issue",
		StepID:        "create_issue",
		ScriptName:    "create_issue",
		Script:        getCreateIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCreateDiscussionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateDiscussions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_discussion")

	return SafeOutputStepConfig{
		StepName:      "Create Discussion",
		StepID:        "create_discussion",
		ScriptName:    "create_discussion",
		Script:        getCreateDiscussionScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCreatePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreatePullRequests

	var customEnvVars []string
	// Pass the base branch from GitHub context (required by create_pull_request.cjs)
	// Note: GH_AW_WORKFLOW_ID is now set at the job level and inherited by all steps
	customEnvVars = append(customEnvVars, "          GH_AW_BASE_BRANCH: ${{ github.ref_name }}\n")
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_PR_TITLE_PREFIX", cfg.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_LABELS", cfg.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_ALLOWED_LABELS", cfg.AllowedLabels)...)
	// Add draft setting - always set with default to true for backwards compatibility
	draftValue := true // Default value
	if cfg.Draft != nil {
		draftValue = *cfg.Draft
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_DRAFT: %q\n", fmt.Sprintf("%t", draftValue)))
	// Add if-no-changes setting - always set with default to "warn"
	ifNoChanges := cfg.IfNoChanges
	if ifNoChanges == "" {
		ifNoChanges = "warn" // Default value
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_IF_NO_CHANGES: %q\n", ifNoChanges))
	// Add allow-empty setting
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_ALLOW_EMPTY: %q\n", fmt.Sprintf("%t", cfg.AllowEmpty)))
	// Add max patch size setting
	maxPatchSize := 1024 // default 1024 KB
	if data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))
	// Add activation comment information if available (for updating the comment with PR link)
	if data.AIReaction != "" && data.AIReaction != "none" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	}
	// Add expires value if set (only for same-repo PRs - when target-repo is not set)
	if cfg.Expires > 0 && cfg.TargetRepoSlug == "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_EXPIRES: \"%d\"\n", cfg.Expires))
	}
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("create_pull_request")

	// Build pre-steps for checkout and git config
	preSteps := c.buildCreatePullRequestPreStepsConsolidated(data, cfg, condition)

	return SafeOutputStepConfig{
		StepName:      "Create Pull Request",
		StepID:        "create_pull_request",
		ScriptName:    "create_pull_request",
		Script:        getCreatePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
		PreSteps:      preSteps,
	}
}

func (c *Compiler) buildAddCommentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool, createIssueEnabled, createDiscussionEnabled, createPullRequestEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddComments

	var customEnvVars []string
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_TARGET: %q\n", cfg.Target))
	}
	if cfg.Discussion != nil && *cfg.Discussion {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_COMMENT_DISCUSSION: \"true\"\n")
	}
	if cfg.HideOlderComments {
		customEnvVars = append(customEnvVars, "          GH_AW_HIDE_OLDER_COMMENTS: \"true\"\n")
	}

	// Reference outputs from earlier steps in the same job
	if createIssueEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_URL: ${{ steps.create_issue.outputs.issue_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_NUMBER: ${{ steps.create_issue.outputs.issue_number }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_TEMPORARY_ID_MAP: ${{ steps.create_issue.outputs.temporary_id_map }}\n")
	}
	if createDiscussionEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_DISCUSSION_URL: ${{ steps.create_discussion.outputs.discussion_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_DISCUSSION_NUMBER: ${{ steps.create_discussion.outputs.discussion_number }}\n")
	}
	if createPullRequestEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_PULL_REQUEST_URL: ${{ steps.create_pull_request.outputs.pull_request_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_PULL_REQUEST_NUMBER: ${{ steps.create_pull_request.outputs.pull_request_number }}\n")
	}
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("add_comment")

	return SafeOutputStepConfig{
		StepName:      "Add Comment",
		StepID:        "add_comment",
		ScriptName:    "add_comment",
		Script:        getAddCommentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCloseDiscussionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CloseDiscussions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_discussion")

	return SafeOutputStepConfig{
		StepName:      "Close Discussion",
		StepID:        "close_discussion",
		ScriptName:    "close_discussion",
		Script:        getCloseDiscussionScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCloseIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CloseIssues

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_issue")

	return SafeOutputStepConfig{
		StepName:      "Close Issue",
		StepID:        "close_issue",
		ScriptName:    "close_issue",
		Script:        getCloseIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildClosePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.ClosePullRequests

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_pull_request")

	return SafeOutputStepConfig{
		StepName:      "Close Pull Request",
		StepID:        "close_pull_request",
		ScriptName:    "close_pull_request",
		Script:        getClosePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCreatePRReviewCommentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreatePullRequestReviewComments

	var customEnvVars []string
	// Add side configuration
	if cfg.Side != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_REVIEW_COMMENT_SIDE: %q\n", cfg.Side))
	}
	// Add target configuration
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_REVIEW_COMMENT_TARGET: %q\n", cfg.Target))
	}
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_pull_request_review_comment")

	return SafeOutputStepConfig{
		StepName:      "Create PR Review Comment",
		StepID:        "create_pr_review_comment",
		ScriptName:    "create_pr_review_comment",
		Script:        getCreatePRReviewCommentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCreateCodeScanningAlertStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool, workflowFilename string) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateCodeScanningAlerts

	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_FILENAME: %q\n", workflowFilename))
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_code_scanning_alert")

	return SafeOutputStepConfig{
		StepName:      "Create Code Scanning Alert",
		StepID:        "create_code_scanning_alert",
		ScriptName:    "create_code_scanning_alert",
		Script:        getCreateCodeScanningAlertScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildAddLabelsStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddLabels

	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_LABELS_ALLOWED", cfg.Allowed)...)
	if cfg.Max > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_MAX_COUNT: %d\n", cfg.Max))
	}
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_TARGET: %q\n", cfg.Target))
	}
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("add_labels")

	return SafeOutputStepConfig{
		StepName:      "Add Labels",
		StepID:        "add_labels",
		ScriptName:    "add_labels",
		Script:        getAddLabelsScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildAddReviewerStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddReviewer

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("add_reviewer")

	return SafeOutputStepConfig{
		StepName:      "Add Reviewer",
		StepID:        "add_reviewer",
		ScriptName:    "add_reviewer",
		Script:        getAddReviewerScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildAssignMilestoneStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AssignMilestone

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("assign_milestone")

	return SafeOutputStepConfig{
		StepName:      "Assign Milestone",
		StepID:        "assign_milestone",
		ScriptName:    "assign_milestone",
		Script:        getAssignMilestoneScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildAssignToAgentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AssignToAgent

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("assign_to_agent")

	return SafeOutputStepConfig{
		StepName:      "Assign To Agent",
		StepID:        "assign_to_agent",
		ScriptName:    "assign_to_agent",
		Script:        getAssignToAgentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
		UseAgentToken: true,
	}
}

func (c *Compiler) buildAssignToUserStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AssignToUser

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("assign_to_user")

	return SafeOutputStepConfig{
		StepName:      "Assign To User",
		StepID:        "assign_to_user",
		ScriptName:    "assign_to_user",
		Script:        getAssignToUserScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildUpdateIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateIssues

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("update_issue")

	return SafeOutputStepConfig{
		StepName:      "Update Issue",
		StepID:        "update_issue",
		ScriptName:    "update_issue",
		Script:        getUpdateIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildUpdatePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdatePullRequests

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("update_pull_request")

	return SafeOutputStepConfig{
		StepName:      "Update Pull Request",
		StepID:        "update_pull_request",
		ScriptName:    "update_pull_request",
		Script:        getUpdatePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildUpdateDiscussionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateDiscussions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Add target environment variable if set
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_TARGET: %q\n", cfg.Target))
	}

	// Add field update flags - presence of pointer indicates field can be updated
	if cfg.Title != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_UPDATE_TITLE: \"true\"\n")
	}
	if cfg.Body != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_UPDATE_BODY: \"true\"\n")
	}
	if cfg.Labels != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_UPDATE_LABELS: \"true\"\n")
	}

	condition := BuildSafeOutputType("update_discussion")

	return SafeOutputStepConfig{
		StepName:      "Update Discussion",
		StepID:        "update_discussion",
		ScriptName:    "update_discussion",
		Script:        getUpdateDiscussionScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildPushToPullRequestBranchStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.PushToPullRequestBranch

	var customEnvVars []string
	// Add target config if set
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PUSH_TARGET: %q\n", cfg.Target))
	}
	// Add if-no-changes config if set
	if cfg.IfNoChanges != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PUSH_IF_NO_CHANGES: %q\n", cfg.IfNoChanges))
	}
	// Add title prefix if set (using same env var as create-pull-request)
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_PR_TITLE_PREFIX", cfg.TitlePrefix)...)
	// Add labels if set
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_LABELS", cfg.Labels)...)
	// Add commit title suffix if set
	if cfg.CommitTitleSuffix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMIT_TITLE_SUFFIX: %q\n", cfg.CommitTitleSuffix))
	}
	// Add max patch size setting
	maxPatchSize := 1024 // default 1024 KB
	if data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("push_to_pull_request_branch")

	// Build pre-steps for checkout and git config
	preSteps := c.buildPushToPullRequestBranchPreStepsConsolidated(data, cfg, condition)

	return SafeOutputStepConfig{
		StepName:      "Push To Pull Request Branch",
		StepID:        "push_to_pull_request_branch",
		ScriptName:    "push_to_pull_request_branch",
		Script:        getPushToPullRequestBranchScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
		PreSteps:      preSteps,
	}
}

func (c *Compiler) buildUploadAssetsStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UploadAssets

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("upload_asset")

	return SafeOutputStepConfig{
		StepName:      "Upload Assets",
		StepID:        "upload_assets",
		ScriptName:    "upload_assets",
		Script:        getUploadAssetsScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildUpdateReleaseStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateRelease

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("update_release")

	return SafeOutputStepConfig{
		StepName:      "Update Release",
		StepID:        "update_release",
		ScriptName:    "update_release",
		Script:        getUpdateReleaseScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildLinkSubIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool, createIssueEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.LinkSubIssue

	var customEnvVars []string
	if createIssueEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_NUMBER: ${{ steps.create_issue.outputs.issue_number }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_TEMPORARY_ID_MAP: ${{ steps.create_issue.outputs.temporary_id_map }}\n")
	}
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("link_sub_issue")

	return SafeOutputStepConfig{
		StepName:      "Link Sub Issue",
		StepID:        "link_sub_issue",
		ScriptName:    "link_sub_issue",
		Script:        getLinkSubIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildHideCommentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.HideComment

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("hide_comment")

	return SafeOutputStepConfig{
		StepName:      "Hide Comment",
		StepID:        "hide_comment",
		ScriptName:    "hide_comment",
		Script:        getHideCommentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

func (c *Compiler) buildCreateAgentTaskStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateAgentTasks

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_agent_task")

	return SafeOutputStepConfig{
		StepName:        "Create Agent Task",
		StepID:          "create_agent_task",
		Script:          createAgentTaskScript,
		CustomEnvVars:   customEnvVars,
		Condition:       condition,
		Token:           cfg.GitHubToken,
		UseCopilotToken: true,
	}
}

func (c *Compiler) buildUpdateProjectStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateProjects

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("update_project")

	return SafeOutputStepConfig{
		StepName:      "Update Project",
		StepID:        "update_project",
		ScriptName:    "update_project",
		Script:        getUpdateProjectScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildCreatePullRequestPreSteps builds the pre-steps for create-pull-request
func (c *Compiler) buildCreatePullRequestPreStepsConsolidated(data *WorkflowData, cfg *CreatePullRequestsConfig, condition ConditionNode) []string {
	// This is a simplified version - the actual implementation would include
	// checkout, git config, and patch application steps
	return nil
}

// buildPushToPullRequestBranchPreSteps builds the pre-steps for push-to-pull-request-branch
func (c *Compiler) buildPushToPullRequestBranchPreStepsConsolidated(data *WorkflowData, cfg *PushToPullRequestBranchConfig, condition ConditionNode) []string {
	// This is a simplified version - the actual implementation would include
	// checkout and git config steps
	return nil
}
