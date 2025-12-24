package workflow

import (
	"fmt"

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

	// Add GitHub App token minting step if app is configured
	if data.SafeOutputs.App != nil {
		consolidatedSafeOutputsLog.Print("Adding GitHub App token minting step")
		// We'll compute permissions after collecting all step requirements
	}

	// Add setup action to copy JavaScript files
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" {
		// For dev mode (local action path), checkout the actions folder first
		// Only add checkout if we have contents read permission
		if c.actionMode.IsDev() {
			permParser := NewPermissionsParser(data.Permissions)
			if permParser.HasContentsReadAccess() {
				steps = append(steps, "      - name: Checkout actions folder\n")
				steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
				steps = append(steps, "        with:\n")
				steps = append(steps, "          persist-credentials: false\n")
				steps = append(steps, "          sparse-checkout: |\n")
				steps = append(steps, "            actions\n")
			}
		}

		steps = append(steps, "      - name: Setup Scripts\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", setupActionRef))
		steps = append(steps, "        with:\n")
		steps = append(steps, fmt.Sprintf("          destination: %s\n", SetupActionDestination))
	}

	// Add artifact download steps after setup
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

	// 18. Upload Assets - now handled as a separate job (see buildSafeOutputsJobs)
	// This was moved out of the consolidated job to allow proper git configuration
	// for pushing to orphaned branches

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
		// Calculate insertion index: after setup action (if present) and artifact downloads, but before safe output steps
		insertIndex := 0

		// Count setup action steps (checkout + setup if in dev mode, or just setup)
		setupActionRef := c.resolveActionReference("./actions/setup", data)
		if setupActionRef != "" {
			if c.actionMode.IsDev() {
				// Only count checkout step if we have contents read permission
				permParser := NewPermissionsParser(data.Permissions)
				if permParser.HasContentsReadAccess() {
					insertIndex += 6 // Checkout step (6 lines: name, uses, with, persist-credentials, sparse-checkout header, path)
				}
			}
			insertIndex += 4 // Setup step (4 lines: name, uses, with, destination)
		}

		// Add artifact download steps count
		insertIndex += len(buildAgentOutputDownloadSteps())

		// Add patch download steps if present
		if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
			patchDownloadSteps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
				ArtifactName: "aw.patch",
				DownloadPath: "/tmp/gh-aw/",
				SetupEnvStep: false,
				StepName:     "Download patch artifact",
			})
			insertIndex += len(patchDownloadSteps)
		}

		// Insert app token steps
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
	// Use require mode if ScriptName is set, otherwise inline the bundled script
	if config.ScriptName != "" {
		// Require mode: Attach GitHub Actions builtin objects to global scope before requiring
		steps = append(steps, "            global.core = core;\n")
		steps = append(steps, "            global.github = github;\n")
		steps = append(steps, "            global.context = context;\n")
		steps = append(steps, "            global.exec = exec;\n")
		steps = append(steps, "            global.io = io;\n")
		steps = append(steps, fmt.Sprintf("            const { main } = require('"+SetupActionDestination+"/%s.cjs');\n", config.ScriptName))
		steps = append(steps, "            await main();\n")
	} else {
		// Inline JavaScript: Attach GitHub Actions builtin objects to global scope before script execution
		steps = append(steps, "            global.core = core;\n")
		steps = append(steps, "            global.github = github;\n")
		steps = append(steps, "            global.context = context;\n")
		steps = append(steps, "            global.exec = exec;\n")
		steps = append(steps, "            global.io = io;\n")
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

	// Note: Asset upload configuration is not needed here because upload_assets
	// is now handled as a separate job (see buildUploadAssetsJob)

	return envVars
}

// buildDetectionSuccessCondition builds the condition to check if detection passed
func buildDetectionSuccessCondition() ConditionNode {
	return BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.success", constants.DetectionJobName)),
		BuildStringLiteral("true"),
	)
}
