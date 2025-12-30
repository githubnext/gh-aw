package workflow

import (
	"encoding/json"
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
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)

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

	// Add shared checkout and git config steps for PR operations
	// Both create-pull-request and push-to-pull-request-branch need these steps,
	// so we add them once with a combined condition to avoid duplication
	var prCheckoutStepsAdded bool
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		consolidatedSafeOutputsLog.Print("Adding shared checkout step for PR operations")
		checkoutSteps := c.buildSharedPRCheckoutSteps(data)
		steps = append(steps, checkoutSteps...)
		prCheckoutStepsAdded = true
	}

	// === Build safe output steps ===

	// Check if any handler-manager-supported types are enabled
	hasHandlerManagerTypes := data.SafeOutputs.CreateIssues != nil ||
		data.SafeOutputs.AddComments != nil ||
		data.SafeOutputs.CreateDiscussions != nil ||
		data.SafeOutputs.CloseIssues != nil ||
		data.SafeOutputs.CloseDiscussions != nil ||
		data.SafeOutputs.AddLabels != nil ||
		data.SafeOutputs.UpdateIssues != nil ||
		data.SafeOutputs.UpdateDiscussions != nil

	// If we have handler manager types, use the handler manager step
	if hasHandlerManagerTypes {
		consolidatedSafeOutputsLog.Print("Using handler manager for safe outputs")
		handlerManagerSteps := c.buildHandlerManagerStep(data)
		steps = append(steps, handlerManagerSteps...)
		safeOutputStepNames = append(safeOutputStepNames, "process_safe_outputs")

		// Track enabled types for other steps
		if data.SafeOutputs.CreateIssues != nil {
			createIssueEnabled = true
		}

		// Add outputs from handler manager
		outputs["process_safe_outputs_temporary_id_map"] = "${{ steps.process_safe_outputs.outputs.temporary_id_map }}"
		outputs["process_safe_outputs_processed_count"] = "${{ steps.process_safe_outputs.outputs.processed_count }}"

		// Merge permissions for all handler-managed types
		if data.SafeOutputs.CreateIssues != nil {
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		}
		if data.SafeOutputs.CreateDiscussions != nil {
			permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
		}
		if data.SafeOutputs.AddComments != nil {
			permissions.Merge(NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite())
		}
		if data.SafeOutputs.CloseIssues != nil {
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		}
		if data.SafeOutputs.CloseDiscussions != nil {
			permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
		}
		if data.SafeOutputs.AddLabels != nil {
			permissions.Merge(NewPermissionsContentsReadIssuesWritePRWrite())
		}
		if data.SafeOutputs.UpdateIssues != nil {
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		}
		if data.SafeOutputs.UpdateDiscussions != nil {
			permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
		}
	}

	// Create Pull Request step (not handled by handler manager)
	if data.SafeOutputs.CreatePullRequests != nil {
		createPullRequestEnabled = true
		_ = createPullRequestEnabled // Track for potential future use
		stepConfig := c.buildCreatePullRequestStepConfig(data, mainJobName, threatDetectionEnabled)
		// Skip pre-steps if we've already added the shared checkout steps
		if !prCheckoutStepsAdded {
			steps = append(steps, stepConfig.PreSteps...)
		}
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		steps = append(steps, stepConfig.PostSteps...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["create_pull_request_pull_request_number"] = "${{ steps.create_pull_request.outputs.pull_request_number }}"
		outputs["create_pull_request_pull_request_url"] = "${{ steps.create_pull_request.outputs.pull_request_url }}"

		permissions.Merge(NewPermissionsContentsWriteIssuesWritePRWrite())
	}

	// Close Pull Request step (not handled by handler manager)
	if data.SafeOutputs.ClosePullRequests != nil {
		stepConfig := c.buildClosePullRequestStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}

	// Create PR Review Comment step
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
		// Skip pre-steps if we've already added the shared checkout steps
		if !prCheckoutStepsAdded {
			steps = append(steps, stepConfig.PreSteps...)
		}
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

		// Count setup action steps (checkout + setup if in dev mode without action-tag, or just setup)
		setupActionRef := c.resolveActionReference("./actions/setup", data)
		if setupActionRef != "" {
			if len(c.generateCheckoutActionsFolder(data)) > 0 {
				insertIndex += 6 // Checkout step (6 lines: name, uses, with, sparse-checkout header, actions, persist-credentials)
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
		needs = append(needs, string(constants.DetectionJobName))
	}
	// Add activation job dependency for jobs that need it (create_pull_request, push_to_pull_request_branch)
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		needs = append(needs, string(constants.ActivationJobName))
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
		// Require mode: Use setup_globals helper
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
		steps = append(steps, fmt.Sprintf("            const { main } = require('"+SetupActionDestination+"/%s.cjs');\n", config.ScriptName))
		steps = append(steps, "            await main();\n")
	} else {
		// Inline JavaScript: Use setup_globals helper
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
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

// buildSharedPRCheckoutSteps builds the shared checkout and git config steps
// used by both create-pull-request and push-to-pull-request-branch operations.
// This avoids duplicate checkout steps when both operations are configured.
func (c *Compiler) buildSharedPRCheckoutSteps(data *WorkflowData) []string {
	consolidatedSafeOutputsLog.Print("Building shared PR checkout steps")
	var steps []string

	// Determine which token to use for checkout
	var checkoutToken string
	var gitRemoteToken string
	if data.SafeOutputs.App != nil {
		checkoutToken = "${{ steps.app-token.outputs.token }}"
		gitRemoteToken = "${{ steps.app-token.outputs.token }}"
	} else {
		checkoutToken = "${{ github.token }}"
		gitRemoteToken = "${{ github.token }}"
	}

	// Build combined condition: execute if either create_pull_request or push_to_pull_request_branch will run
	var condition ConditionNode
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.PushToPullRequestBranch != nil {
		// Both enabled: combine conditions with OR
		condition = BuildOr(
			BuildSafeOutputType("create_pull_request"),
			BuildSafeOutputType("push_to_pull_request_branch"),
		)
	} else if data.SafeOutputs.CreatePullRequests != nil {
		// Only create_pull_request
		condition = BuildSafeOutputType("create_pull_request")
	} else {
		// Only push_to_pull_request_branch
		condition = BuildSafeOutputType("push_to_pull_request_branch")
	}

	// Step 1: Checkout repository with conditional execution
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, fmt.Sprintf("        if: %s\n", condition.Render()))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          token: %s\n", checkoutToken))
	steps = append(steps, "          persist-credentials: false\n")
	steps = append(steps, "          fetch-depth: 1\n")
	if c.trialMode {
		if c.trialLogicalRepoSlug != "" {
			steps = append(steps, fmt.Sprintf("          repository: %s\n", c.trialLogicalRepoSlug))
		}
	}

	// Step 2: Configure Git credentials with conditional execution
	gitConfigSteps := []string{
		"      - name: Configure Git credentials\n",
		fmt.Sprintf("        if: %s\n", condition.Render()),
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"          SERVER_URL: ${{ github.server_url }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		fmt.Sprintf("          git remote set-url origin \"https://x-access-token:%s@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n", gitRemoteToken),
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
	steps = append(steps, gitConfigSteps...)

	consolidatedSafeOutputsLog.Printf("Added shared checkout with condition: %s", condition.Render())
	return steps
}

// buildHandlerManagerStep builds a single step that uses the safe output handler manager
// to dispatch messages to appropriate handlers. This replaces multiple individual steps
// with a single dispatcher step.
func (c *Compiler) buildHandlerManagerStep(data *WorkflowData) []string {
	consolidatedSafeOutputsLog.Print("Building handler manager step")

	var steps []string

	// Step name and metadata
	steps = append(steps, "      - name: Process Safe Outputs\n")
	steps = append(steps, "        id: process_safe_outputs\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")

	// Add custom safe output env vars
	c.addCustomSafeOutputEnvVars(&steps, data)

	// Add handler manager config as JSON
	c.addHandlerManagerConfigEnvVar(&steps, data)

	// Add all safe output configuration env vars (still needed by individual handlers)
	c.addAllSafeOutputConfigEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	c.addSafeOutputGitHubTokenForConfig(&steps, data, "")

	steps = append(steps, "          script: |\n")
	steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
	steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
	steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/safe_output_handler_manager.cjs');\n")
	steps = append(steps, "            await main();\n")

	return steps
}

// addHandlerManagerConfigEnvVar adds a JSON config environment variable for the handler manager
// This config indicates which handlers should be loaded and includes their type-specific options
// The presence of a config key indicates that handler is enabled (no explicit "enabled" field needed)
func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	config := make(map[string]map[string]any)

	// Add config for each enabled safe output type with their options
	// Presence in config = enabled, so no need for "enabled": true field
	if data.SafeOutputs.CreateIssues != nil {
		cfg := data.SafeOutputs.CreateIssues
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.AllowedLabels) > 0 {
			handlerConfig["allowed_labels"] = cfg.AllowedLabels
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		if cfg.Expires > 0 {
			handlerConfig["expires"] = cfg.Expires
		}
		// Add labels, title_prefix to config
		if len(cfg.Labels) > 0 {
			handlerConfig["labels"] = cfg.Labels
		}
		if cfg.TitlePrefix != "" {
			handlerConfig["title_prefix"] = cfg.TitlePrefix
		}
		config["create_issue"] = handlerConfig
	}

	if data.SafeOutputs.AddComments != nil {
		cfg := data.SafeOutputs.AddComments
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if cfg.HideOlderComments {
			handlerConfig["hide_older_comments"] = true
		}
		config["add_comment"] = handlerConfig
	}

	if data.SafeOutputs.CreateDiscussions != nil {
		cfg := data.SafeOutputs.CreateDiscussions
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Category != "" {
			handlerConfig["category"] = cfg.Category
		}
		if len(cfg.AllowedLabels) > 0 {
			handlerConfig["allowed_labels"] = cfg.AllowedLabels
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		if cfg.Expires > 0 {
			handlerConfig["expires"] = cfg.Expires
		}
		config["create_discussion"] = handlerConfig
	}

	if data.SafeOutputs.CloseIssues != nil {
		cfg := data.SafeOutputs.CloseIssues
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if len(cfg.RequiredLabels) > 0 {
			handlerConfig["required_labels"] = cfg.RequiredLabels
		}
		if cfg.RequiredTitlePrefix != "" {
			handlerConfig["required_title_prefix"] = cfg.RequiredTitlePrefix
		}
		config["close_issue"] = handlerConfig
	}

	if data.SafeOutputs.CloseDiscussions != nil {
		cfg := data.SafeOutputs.CloseDiscussions
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if len(cfg.RequiredLabels) > 0 {
			handlerConfig["required_labels"] = cfg.RequiredLabels
		}
		if cfg.RequiredTitlePrefix != "" {
			handlerConfig["required_title_prefix"] = cfg.RequiredTitlePrefix
		}
		if cfg.RequiredCategory != "" {
			handlerConfig["required_category"] = cfg.RequiredCategory
		}
		config["close_discussion"] = handlerConfig
	}

	if data.SafeOutputs.AddLabels != nil {
		cfg := data.SafeOutputs.AddLabels
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.Allowed) > 0 {
			handlerConfig["allowed"] = cfg.Allowed
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		config["add_labels"] = handlerConfig
	}

	if data.SafeOutputs.UpdateIssues != nil {
		cfg := data.SafeOutputs.UpdateIssues
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		// Boolean pointer fields indicate which fields can be updated
		if cfg.Status != nil {
			handlerConfig["allow_status"] = true
		}
		if cfg.Title != nil {
			handlerConfig["allow_title"] = true
		}
		if cfg.Body != nil {
			handlerConfig["allow_body"] = true
		}
		config["update_issue"] = handlerConfig
	}

	if data.SafeOutputs.UpdateDiscussions != nil {
		cfg := data.SafeOutputs.UpdateDiscussions
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		// Boolean pointer fields indicate which fields can be updated
		if cfg.Title != nil {
			handlerConfig["allow_title"] = true
		}
		if cfg.Body != nil {
			handlerConfig["allow_body"] = true
		}
		if cfg.Labels != nil {
			handlerConfig["allow_labels"] = true
		}
		if len(cfg.AllowedLabels) > 0 {
			handlerConfig["allowed_labels"] = cfg.AllowedLabels
		}
		config["update_discussion"] = handlerConfig
	}

	// Only add the env var if there are handlers to configure
	if len(config) > 0 {
		configJSON, err := json.Marshal(config)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Failed to marshal handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
	}
}

// addAllSafeOutputConfigEnvVars adds environment variables for all enabled safe output types
// These are needed by individual handlers when called by the handler manager
func (c *Compiler) addAllSafeOutputConfigEnvVars(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	// Track if we've already added staged flag to avoid duplicates
	stagedFlagAdded := false

	// Create Issue env vars - now handled by config object
	if data.SafeOutputs.CreateIssues != nil {
		cfg := data.SafeOutputs.CreateIssues
		// Only add allowed_labels and allowed_repos env vars for backward compatibility with processSafeOutput helper
		*steps = append(*steps, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", cfg.AllowedLabels)...)
		*steps = append(*steps, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", cfg.AllowedRepos)...)
		// Add target repo slug if specified
		if cfg.TargetRepoSlug != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", cfg.TargetRepoSlug))
		} else if !c.trialMode && data.SafeOutputs.Staged && !stagedFlagAdded {
			*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
			stagedFlagAdded = true
		}
	}

	// Add Comment env vars
	if data.SafeOutputs.AddComments != nil {
		cfg := data.SafeOutputs.AddComments
		if cfg.Target != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_COMMENT_TARGET: %q\n", cfg.Target))
		}
		if cfg.Discussion != nil && *cfg.Discussion {
			*steps = append(*steps, "          GITHUB_AW_COMMENT_DISCUSSION: \"true\"\n")
		}
		if cfg.HideOlderComments {
			*steps = append(*steps, "          GH_AW_HIDE_OLDER_COMMENTS: \"true\"\n")
		}
	}

	// Add Labels env vars
	if data.SafeOutputs.AddLabels != nil {
		cfg := data.SafeOutputs.AddLabels
		*steps = append(*steps, buildLabelsEnvVar("GH_AW_LABELS_ALLOWED", cfg.Allowed)...)
		if cfg.Max > 0 {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_LABELS_MAX_COUNT: %d\n", cfg.Max))
		}
		if cfg.Target != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_LABELS_TARGET: %q\n", cfg.Target))
		}
		// Add target repo slug if specified
		if cfg.TargetRepoSlug != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", cfg.TargetRepoSlug))
		} else if !c.trialMode && data.SafeOutputs.Staged && !stagedFlagAdded {
			*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
			stagedFlagAdded = true
		}
	}

	// Update Issue env vars
	if data.SafeOutputs.UpdateIssues != nil {
		cfg := data.SafeOutputs.UpdateIssues
		// Add target repo slug if specified
		if cfg.TargetRepoSlug != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", cfg.TargetRepoSlug))
		} else if !c.trialMode && data.SafeOutputs.Staged && !stagedFlagAdded {
			*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
			stagedFlagAdded = true
		}
	}

	// Update Discussion env vars
	if data.SafeOutputs.UpdateDiscussions != nil {
		cfg := data.SafeOutputs.UpdateDiscussions
		// Add target repo slug if specified
		if cfg.TargetRepoSlug != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", cfg.TargetRepoSlug))
		} else if !c.trialMode && data.SafeOutputs.Staged && !stagedFlagAdded {
			*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
			stagedFlagAdded = true
			_ = stagedFlagAdded // Mark as used for linter
		}
		if cfg.Target != "" {
			*steps = append(*steps, fmt.Sprintf("          GH_AW_UPDATE_TARGET: %q\n", cfg.Target))
		}
		if cfg.Title != nil {
			*steps = append(*steps, "          GH_AW_UPDATE_TITLE: \"true\"\n")
		}
		if cfg.Body != nil {
			*steps = append(*steps, "          GH_AW_UPDATE_BODY: \"true\"\n")
		}
	}

	// Note: Most handlers read from the config.json file, so we may not need all env vars here
}
