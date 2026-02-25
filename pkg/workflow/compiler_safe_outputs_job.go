package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var consolidatedSafeOutputsJobLog = logger.New("workflow:compiler_safe_outputs_job")

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
		consolidatedSafeOutputsJobLog.Print("No safe outputs configured, skipping consolidated job")
		return nil, nil, nil
	}

	consolidatedSafeOutputsJobLog.Print("Building consolidated safe outputs job with file mode")

	var steps []string
	var outputs = make(map[string]string)
	var safeOutputStepNames []string

	// Compute permissions based on configured safe outputs (principle of least privilege)
	permissions := ComputePermissionsForSafeOutputs(data.SafeOutputs)

	// Track whether threat detection job is enabled for step conditions
	threatDetectionEnabled := data.SafeOutputs.ThreatDetection != nil

	// Note: GitHub App token minting step is added later (after setup/downloads)
	// to ensure proper step ordering. See insertion logic below.

	// Add setup action to copy JavaScript files
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)

		// Enable safe-output-projects flag if project-related safe outputs are configured
		enableProjectSupport := c.hasProjectRelatedSafeOutputs(data.SafeOutputs)
		steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, enableProjectSupport)...)
	}

	// Add artifact download steps after setup
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Add patch artifact download if create-pull-request or push-to-pull-request-branch is enabled
	// Both of these safe outputs require the patch file to apply changes
	// Download from unified agent-artifacts artifact
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		consolidatedSafeOutputsJobLog.Print("Adding patch artifact download for create-pull-request or push-to-pull-request-branch")
		patchDownloadSteps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
			ArtifactName: "agent-artifacts",
			DownloadPath: "/tmp/gh-aw/",
			SetupEnvStep: false, // No environment variable needed, the script checks the file directly
			StepName:     "Download patch artifact",
		})
		steps = append(steps, patchDownloadSteps...)
	}

	// Add shared checkout and git config steps for PR operations
	// Both create-pull-request and push-to-pull-request-branch need these steps,
	// so we add them once with a combined condition to avoid duplication
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
		consolidatedSafeOutputsJobLog.Print("Adding shared checkout step for PR operations")
		checkoutSteps := c.buildSharedPRCheckoutSteps(data)
		steps = append(steps, checkoutSteps...)
	}

	// Note: Unlock step has been moved to dedicated unlock job
	// The safe_outputs job now depends on the unlock job, so the issue
	// will already be unlocked when this job runs

	// === Build safe output steps ===
	//
	// IMPORTANT: Step order matters for safe outputs that depend on each other.
	// The execution order ensures dependencies are satisfied:
	// 1. Handler Manager - processes create_issue, update_issue, add_comment, etc.
	// 2. Assign To Agent - assigns issue to agent (after handler managers complete)
	// 3. Create Agent Session - creates agent session (after assignment)
	//
	// Note: All project-related operations (create_project, update_project, create_project_status_update)
	// are now handled by the unified handler in the handler manager step.

	// Check if any handler-manager-supported types are enabled
	hasHandlerManagerTypes := data.SafeOutputs.CreateIssues != nil ||
		data.SafeOutputs.AddComments != nil ||
		data.SafeOutputs.CreateDiscussions != nil ||
		data.SafeOutputs.CloseIssues != nil ||
		data.SafeOutputs.CloseDiscussions != nil ||
		data.SafeOutputs.AddLabels != nil ||
		data.SafeOutputs.RemoveLabels != nil ||
		data.SafeOutputs.UpdateIssues != nil ||
		data.SafeOutputs.UpdateDiscussions != nil ||
		data.SafeOutputs.LinkSubIssue != nil ||
		data.SafeOutputs.UpdateRelease != nil ||
		data.SafeOutputs.CreatePullRequestReviewComments != nil ||
		data.SafeOutputs.SubmitPullRequestReview != nil ||
		data.SafeOutputs.ReplyToPullRequestReviewComment != nil ||
		data.SafeOutputs.ResolvePullRequestReviewThread != nil ||
		data.SafeOutputs.CreatePullRequests != nil ||
		data.SafeOutputs.PushToPullRequestBranch != nil ||
		data.SafeOutputs.UpdatePullRequests != nil ||
		data.SafeOutputs.ClosePullRequests != nil ||
		data.SafeOutputs.MarkPullRequestAsReadyForReview != nil ||
		data.SafeOutputs.HideComment != nil ||
		data.SafeOutputs.DispatchWorkflow != nil ||
		data.SafeOutputs.CreateCodeScanningAlerts != nil ||
		data.SafeOutputs.AutofixCodeScanningAlert != nil ||
		data.SafeOutputs.MissingTool != nil ||
		data.SafeOutputs.MissingData != nil

	// Note: All project-related operations are now handled by the unified handler.
	// The project handler manager has been removed.

	// 1. Handler Manager step (processes create_issue, update_issue, add_comment, etc.)
	// This processes all safe output types that are handled by the unified handler
	// Critical for workflows that create projects and then add issues/PRs to those projects
	if hasHandlerManagerTypes {
		consolidatedSafeOutputsJobLog.Print("Using handler manager for safe outputs")
		handlerManagerSteps := c.buildHandlerManagerStep(data)
		steps = append(steps, handlerManagerSteps...)
		safeOutputStepNames = append(safeOutputStepNames, "process_safe_outputs")

		// Add outputs from handler manager
		outputs["process_safe_outputs_temporary_id_map"] = "${{ steps.process_safe_outputs.outputs.temporary_id_map }}"
		outputs["process_safe_outputs_processed_count"] = "${{ steps.process_safe_outputs.outputs.processed_count }}"
		outputs["create_discussion_errors"] = "${{ steps.process_safe_outputs.outputs.create_discussion_errors }}"
		outputs["create_discussion_error_count"] = "${{ steps.process_safe_outputs.outputs.create_discussion_error_count }}"
		outputs["code_push_failure_errors"] = "${{ steps.process_safe_outputs.outputs.code_push_failure_errors }}"
		outputs["code_push_failure_count"] = "${{ steps.process_safe_outputs.outputs.code_push_failure_count }}"

		// Note: Permissions are now computed centrally by ComputePermissionsForSafeOutputs()
		// at the start of this function to ensure consistent permission calculation

		// If create-issue is configured with assignees: copilot, run a follow-up step to
		// assign the Copilot coding agent. The handler manager exports the list via
		// steps.process_safe_outputs.outputs.issues_to_assign_copilot.
		if data.SafeOutputs.CreateIssues != nil && hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees) {
			consolidatedSafeOutputsJobLog.Print("Adding copilot assignment step for created issues")
			steps = append(steps, "      - name: Assign Copilot to created issues\n")
			steps = append(steps, "        if: steps.process_safe_outputs.outputs.issues_to_assign_copilot != ''\n")
			steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
			steps = append(steps, "        env:\n")
			steps = append(steps, "          GH_AW_ISSUES_TO_ASSIGN_COPILOT: ${{ steps.process_safe_outputs.outputs.issues_to_assign_copilot }}\n")
			steps = append(steps, "        with:\n")
			c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, data.SafeOutputs.CreateIssues.GitHubToken)
			steps = append(steps, "          script: |\n")
			steps = append(steps, generateGitHubScriptWithRequire("assign_copilot_to_created_issues.cjs"))
		}
	}

	// 3. Assign To Agent step (runs after handler managers)
	if data.SafeOutputs.AssignToAgent != nil {
		stepConfig := c.buildAssignToAgentStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["assign_to_agent_assigned"] = "${{ steps.assign_to_agent.outputs.assigned }}"
		outputs["assign_to_agent_assignment_errors"] = "${{ steps.assign_to_agent.outputs.assignment_errors }}"
		outputs["assign_to_agent_assignment_error_count"] = "${{ steps.assign_to_agent.outputs.assignment_error_count }}"

		// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()
	}

	// 4. Create Agent Session step
	if data.SafeOutputs.CreateAgentSessions != nil {
		stepConfig := c.buildCreateAgentSessionStepConfig(data, mainJobName, threatDetectionEnabled)
		stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
		steps = append(steps, stepYAML...)
		safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

		outputs["create_agent_session_session_number"] = "${{ steps.create_agent_session.outputs.session_number }}"
		outputs["create_agent_session_session_url"] = "${{ steps.create_agent_session.outputs.session_url }}"

		// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()
	}

	// Note: Create Pull Request is now handled by the handler manager
	// The outputs and permissions are configured in the handler manager section above

	// Note: Mark Pull Request as Ready for Review is now handled by the handler manager
	// The permissions are configured in the handler manager section above

	// Note: Create Code Scanning Alert is now handled by the handler manager
	// The permissions are configured in the handler manager section above
	// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()

	// Note: Create Project Status Update is now handled by the handler manager
	// The permissions are configured in the handler manager section above
	// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()

	// Note: Add Reviewer is now handled by the handler manager
	// The outputs and permissions are configured in the handler manager section above
	if data.SafeOutputs.AddReviewer != nil {
		outputs["add_reviewer_reviewers_added"] = "${{ steps.process_safe_outputs.outputs.reviewers_added }}"
		// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()
	}

	// Note: Assign Milestone is now handled by the handler manager
	// The outputs and permissions are configured in the handler manager section above
	if data.SafeOutputs.AssignMilestone != nil {
		outputs["assign_milestone_milestone_assigned"] = "${{ steps.process_safe_outputs.outputs.milestone_assigned }}"
		// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()
	}

	// Note: Assign To User is now handled by the handler manager
	// The outputs and permissions are configured in the handler manager section above
	if data.SafeOutputs.AssignToUser != nil {
		outputs["assign_to_user_assigned"] = "${{ steps.process_safe_outputs.outputs.assigned }}"
		// Note: Permissions are computed centrally by ComputePermissionsForSafeOutputs()
	}

	// Note: Update Pull Request step - now handled by handler manager

	// Note: Push To Pull Request Branch step - now handled by handler manager

	// Note: Upload Assets - now handled as a separate job (see buildSafeOutputsJobs)
	// This was moved out of the consolidated job to allow proper git configuration
	// for pushing to orphaned branches

	// Note: Update Release step - now handled by handler manager
	// Note: Link Sub Issue step - now handled by handler manager
	// Note: Hide Comment step - now handled by handler manager

	// If no steps were added, return nil
	if len(safeOutputStepNames) == 0 {
		consolidatedSafeOutputsJobLog.Print("No safe output steps were added")
		return nil, nil, nil
	}

	// Add GitHub App token minting step at the beginning if app is configured
	if data.SafeOutputs.App != nil {
		appTokenSteps := c.buildGitHubAppTokenMintStep(data.SafeOutputs.App, permissions)
		// Calculate insertion index: after setup action (if present) and artifact downloads, but before checkout and safe output steps
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
		// Download from unified agent-artifacts artifact
		if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil {
			patchDownloadSteps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
				ArtifactName: "agent-artifacts",
				DownloadPath: "/tmp/gh-aw/",
				SetupEnvStep: false,
				StepName:     "Download patch artifact",
			})
			insertIndex += len(patchDownloadSteps)
		}

		// Note: App token step must be inserted BEFORE shared checkout steps
		// because those steps reference steps.safe-outputs-app-token.outputs.token

		// Insert app token steps
		var newSteps []string
		newSteps = append(newSteps, steps[:insertIndex]...)
		newSteps = append(newSteps, appTokenSteps...)
		newSteps = append(newSteps, steps[insertIndex:]...)
		steps = newSteps
	}

	// Add GitHub App token invalidation step at the end if app is configured
	if data.SafeOutputs.App != nil {
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}

	// Upload the safe output items manifest as an artifact (non-staged mode only).
	// This step runs even if previous steps fail, ensuring the audit trail
	// is always available for the audit command to display.
	// In staged mode, no items are actually created in GitHub so there is nothing to record.
	isStaged := c.trialMode || data.SafeOutputs.Staged
	if !isStaged {
		steps = append(steps, buildSafeOutputItemsManifestUploadStep()...)
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

	// Build dependencies — detection is now inline in the agent job, no separate dependency needed
	needs := []string{mainJobName}
	// Add activation job dependency for jobs that need it (create_pull_request, push_to_pull_request_branch, lock-for-agent)
	if data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil || data.LockForAgent {
		needs = append(needs, string(constants.ActivationJobName))
	}
	// Add unlock job dependency if lock-for-agent is enabled
	// This ensures the issue is unlocked before safe outputs run
	if data.LockForAgent {
		needs = append(needs, "unlock")
		consolidatedSafeOutputsJobLog.Print("Added unlock job dependency to safe_outputs job")
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

	consolidatedSafeOutputsJobLog.Printf("Built consolidated safe outputs job with %d steps", len(safeOutputStepNames))

	return job, safeOutputStepNames, nil
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
	if data.SafeOutputs != nil && (c.trialMode || data.SafeOutputs.Staged) {
		envVars["GH_AW_SAFE_OUTPUTS_STAGED"] = "\"true\""
	}

	// Set GH_AW_TARGET_REPO_SLUG - prefer trial target repo (applies to all steps)
	// Note: Individual steps with target-repo config will override this in their step-level env
	if c.trialMode && c.trialLogicalRepoSlug != "" {
		envVars["GH_AW_TARGET_REPO_SLUG"] = fmt.Sprintf("%q", c.trialLogicalRepoSlug)
	}

	// Add messages config if present (applies to all steps)
	if data.SafeOutputs != nil && data.SafeOutputs.Messages != nil {
		messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
		if err != nil {
			consolidatedSafeOutputsJobLog.Printf("Warning: failed to serialize messages config: %v", err)
		} else if messagesJSON != "" {
			envVars["GH_AW_SAFE_OUTPUT_MESSAGES"] = fmt.Sprintf("%q", messagesJSON)
		}
	}

	// Note: GH_AW_CI_TRIGGER_TOKEN is added at the step level (in buildHandlerManagerStep)
	// rather than job level, since only the Process Safe Outputs step needs it,
	// and only when create-pull-request or push-to-pull-request-branch is configured.

	// Note: Asset upload configuration is not needed here because upload_assets
	// is now handled as a separate job (see buildUploadAssetsJob)

	return envVars
}

// buildDetectionSuccessCondition builds the condition to check if detection passed.
// Detection runs inline in the agent job and outputs detection_success.
func buildDetectionSuccessCondition() ConditionNode {
	return BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.detection_success", constants.AgentJobName)),
		BuildStringLiteral("true"),
	)
}

// buildSafeOutputItemsManifestUploadStep builds the step that uploads the safe output
// items manifest as a GitHub Actions artifact. The step always runs (if: always()) so
// the manifest is available to the audit command even if some safe output steps fail.
func buildSafeOutputItemsManifestUploadStep() []string {
	return []string{
		"      - name: Upload safe output items manifest\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")),
		"        with:\n",
		"          name: safe-output-items\n",
		"          path: /tmp/safe-output-items.jsonl\n",
		"          if-no-files-found: warn\n",
	}
}
