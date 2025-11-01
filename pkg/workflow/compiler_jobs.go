package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// This file contains job building functions extracted from compiler.go
// These functions are responsible for constructing the various jobs that make up
// a compiled agentic workflow, including activation, main, safe outputs, and custom jobs.

func (c *Compiler) isActivationJobNeeded() bool {
	// Activation job is always needed to perform the timestamp check
	// It also handles:
	// 1. Command is configured (for team member checking)
	// 2. Text output is needed (for compute-text action)
	// 3. If condition is specified (to handle runtime conditions)
	// 4. Permission checks are needed (consolidated team member validation)
	return true
}

// buildJobs creates all jobs for the workflow and adds them to the job manager
func (c *Compiler) buildJobs(data *WorkflowData, markdownPath string) error {
	// Try to read frontmatter to determine event types for safe events check
	// This is used for the enhanced permission checking logic
	var frontmatter map[string]any
	if content, err := os.ReadFile(markdownPath); err == nil {
		if result, err := parser.ExtractFrontmatterFromContent(string(content)); err == nil {
			frontmatter = result.Frontmatter
		}
	}
	// If frontmatter cannot be read, we'll fall back to the basic permission check logic

	// Main job ID is always constants.AgentJobName

	// Determine if permission checks or stop-time checks are needed
	needsPermissionCheck := c.needsRoleCheck(data, frontmatter)
	hasStopTime := data.StopTime != ""

	// Build pre-activation job if needed (combines membership checks, stop-time validation, and command position check)
	var preActivationJobCreated bool
	hasCommandTrigger := data.Command != ""
	if needsPermissionCheck || hasStopTime || hasCommandTrigger {
		preActivationJob, err := c.buildPreActivationJob(data, needsPermissionCheck)
		if err != nil {
			return fmt.Errorf("failed to build %s job: %w", constants.PreActivationJobName, err)
		}
		if err := c.jobManager.AddJob(preActivationJob); err != nil {
			return fmt.Errorf("failed to add %s job: %w", constants.PreActivationJobName, err)
		}
		preActivationJobCreated = true
	}

	// Build activation job if needed (preamble job that handles runtime conditions)
	// If pre-activation job exists, activation job depends on it and checks the "activated" output
	var activationJobCreated bool

	if c.isActivationJobNeeded() {
		activationJob, err := c.buildActivationJob(data, preActivationJobCreated)
		if err != nil {
			return fmt.Errorf("failed to build activation job: %w", err)
		}
		if err := c.jobManager.AddJob(activationJob); err != nil {
			return fmt.Errorf("failed to add activation job: %w", err)
		}
		activationJobCreated = true
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, activationJobCreated, frontmatter)
	if err != nil {
		return fmt.Errorf("failed to build main job: %w", err)
	}
	if err := c.jobManager.AddJob(mainJob); err != nil {
		return fmt.Errorf("failed to add main job: %w", err)
	}

	// Build safe outputs jobs if configured
	if err := c.buildSafeOutputsJobs(data, constants.AgentJobName, markdownPath); err != nil {
		return fmt.Errorf("failed to build safe outputs jobs: %w", err)
	}

	// Build safe-jobs if configured
	// Safe-jobs should depend on agent job (always) AND detection job (if threat detection is enabled)
	threatDetectionEnabledForSafeJobs := data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil
	if err := c.buildSafeJobs(data, threatDetectionEnabledForSafeJobs); err != nil {
		return fmt.Errorf("failed to build safe-jobs: %w", err)
	}

	// Build additional custom jobs from frontmatter jobs section
	if err := c.buildCustomJobs(data); err != nil {
		return fmt.Errorf("failed to build custom jobs: %w", err)
	}

	return nil
}

// buildSafeOutputsJobs creates all safe outputs jobs if configured
func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName, markdownPath string) error {
	if data.SafeOutputs == nil {
		return nil
	}

	// Track whether threat detection job is enabled
	threatDetectionEnabled := false

	// Build threat detection job if enabled
	if data.SafeOutputs.ThreatDetection != nil {
		detectionJob, err := c.buildThreatDetectionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build detection job: %w", err)
		}
		if err := c.jobManager.AddJob(detectionJob); err != nil {
			return fmt.Errorf("failed to add detection job: %w", err)
		}
		threatDetectionEnabled = true
	}

	// Track safe output job names to establish dependencies for update_reaction job
	var safeOutputJobNames []string

	// Track which jobs create_issue, create_discussion, and create_pull_request were created
	// so that add_comment can depend on them and reference their outputs
	var createIssueJobName string
	var createDiscussionJobName string
	var createPullRequestJobName string

	// Build create_issue job if output.create_issue is configured
	if data.SafeOutputs.CreateIssues != nil {
		createIssueJob, err := c.buildCreateOutputIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_issue job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createIssueJob.Needs = append(createIssueJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createIssueJob); err != nil {
			return fmt.Errorf("failed to add create_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createIssueJob.Name)
		createIssueJobName = createIssueJob.Name
	}

	// Build create_discussion job if output.create_discussion is configured
	if data.SafeOutputs.CreateDiscussions != nil {
		createDiscussionJob, err := c.buildCreateOutputDiscussionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_discussion job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createDiscussionJob.Needs = append(createDiscussionJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createDiscussionJob); err != nil {
			return fmt.Errorf("failed to add create_discussion job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createDiscussionJob.Name)
		createDiscussionJobName = createDiscussionJob.Name
	}

	// Build create_pull_request job if output.create-pull-request is configured
	// NOTE: This is built BEFORE add_comment so that add_comment can depend on it
	if data.SafeOutputs.CreatePullRequests != nil {
		createPullRequestJob, err := c.buildCreateOutputPullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pull_request job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createPullRequestJob.Needs = append(createPullRequestJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createPullRequestJob); err != nil {
			return fmt.Errorf("failed to add create_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createPullRequestJob.Name)
		createPullRequestJobName = createPullRequestJob.Name
	}

	// Build add_comment job if output.add-comment is configured
	if data.SafeOutputs.AddComments != nil {
		createCommentJob, err := c.buildCreateOutputAddCommentJob(data, jobName, createIssueJobName, createDiscussionJobName, createPullRequestJobName)
		if err != nil {
			return fmt.Errorf("failed to build add_comment job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createCommentJob.Needs = append(createCommentJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createCommentJob); err != nil {
			return fmt.Errorf("failed to add add_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createCommentJob.Name)
	}

	// Build create_pr_review_comment job if output.create-pull-request-review-comment is configured
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		createPRReviewCommentJob, err := c.buildCreateOutputPullRequestReviewCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pr_review_comment job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createPRReviewCommentJob.Needs = append(createPRReviewCommentJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createPRReviewCommentJob); err != nil {
			return fmt.Errorf("failed to add create_pr_review_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createPRReviewCommentJob.Name)
	}

	// Build create_code_scanning_alert job if output.create-code-scanning-alert is configured
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		// Extract the workflow filename without extension for rule ID prefix
		workflowFilename := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
		createCodeScanningAlertJob, err := c.buildCreateOutputCodeScanningAlertJob(data, jobName, workflowFilename)
		if err != nil {
			return fmt.Errorf("failed to build create_code_scanning_alert job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createCodeScanningAlertJob.Needs = append(createCodeScanningAlertJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createCodeScanningAlertJob); err != nil {
			return fmt.Errorf("failed to add create_code_scanning_alert job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createCodeScanningAlertJob.Name)
	}

	// Build add_labels job if output.add-labels is configured (including null/empty)
	if data.SafeOutputs.AddLabels != nil {
		addLabelsJob, err := c.buildAddLabelsJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build add_labels job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			addLabelsJob.Needs = append(addLabelsJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(addLabelsJob); err != nil {
			return fmt.Errorf("failed to add add_labels job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, addLabelsJob.Name)
	}

	// Build update_issue job if output.update-issue is configured
	if data.SafeOutputs.UpdateIssues != nil {
		updateIssueJob, err := c.buildCreateOutputUpdateIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_issue job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			updateIssueJob.Needs = append(updateIssueJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(updateIssueJob); err != nil {
			return fmt.Errorf("failed to add update_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, updateIssueJob.Name)
	}

	// Build push_to_pull_request_branch job if output.push-to-pull-request-branch is configured
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		pushToBranchJob, err := c.buildCreateOutputPushToPullRequestBranchJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build push_to_pull_request_branch job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			pushToBranchJob.Needs = append(pushToBranchJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(pushToBranchJob); err != nil {
			return fmt.Errorf("failed to add push_to_pull_request_branch job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, pushToBranchJob.Name)
	}

	// Build missing_tool job (always enabled when SafeOutputs exists)
	if data.SafeOutputs.MissingTool != nil {
		missingToolJob, err := c.buildCreateOutputMissingToolJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build missing_tool job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			missingToolJob.Needs = append(missingToolJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(missingToolJob); err != nil {
			return fmt.Errorf("failed to add missing_tool job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, missingToolJob.Name)
	}

	// Build upload_assets job if output.upload-asset is configured
	if data.SafeOutputs.UploadAssets != nil {
		uploadAssetsJob, err := c.buildUploadAssetsJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build upload_assets job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			uploadAssetsJob.Needs = append(uploadAssetsJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
			return fmt.Errorf("failed to add upload_assets job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, uploadAssetsJob.Name)
	}

	// Build create_agent_task job if output.create-agent-task is configured
	if data.SafeOutputs.CreateAgentTasks != nil {
		createAgentTaskJob, err := c.buildCreateOutputAgentTaskJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_agent_task job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createAgentTaskJob.Needs = append(createAgentTaskJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createAgentTaskJob); err != nil {
			return fmt.Errorf("failed to add create_agent_task job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createAgentTaskJob.Name)
	}

	// Build update_reaction job if add-comment is configured OR if command trigger is configured with reactions
	// This job runs last, after all safe output jobs, to update the activation comment on failure
	// The buildUpdateReactionJob function itself will decide whether to create the job based on the configuration
	updateReactionJob, err := c.buildUpdateReactionJob(data, jobName, safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build update_reaction job: %w", err)
	}
	if updateReactionJob != nil {
		if err := c.jobManager.AddJob(updateReactionJob); err != nil {
			return fmt.Errorf("failed to add update_reaction job: %w", err)
		}
	}

	return nil
}

// buildPreActivationJob creates a unified pre-activation job that combines membership checks and stop-time validation
// This job exposes a single "activated" output that indicates whether the workflow should proceed
func (c *Compiler) buildPreActivationJob(data *WorkflowData, needsPermissionCheck bool) (*Job, error) {
	var steps []string
	var permissions string

	// Add team member check if permission checks are needed
	if needsPermissionCheck {
		steps = c.generateMembershipCheck(data, steps)
	}

	// Add stop-time check if configured
	if data.StopTime != "" {
		// Extract workflow name for the stop-time check
		workflowName := data.Name

		steps = append(steps, "      - name: Check stop-time limit\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckStopTimeStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_STOP_TIME: %s\n", data.StopTime))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkStopTimeScript)
		steps = append(steps, formattedScript...)
	}

	// Add command position check if this is a command workflow
	if data.Command != "" {
		steps = append(steps, "      - name: Check command position\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckCommandPositionStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_COMMAND: %s\n", data.Command))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkCommandPositionScript)
		steps = append(steps, formattedScript...)
	}

	// Generate the activated output expression using expression builders
	var activatedNode ConditionNode

	// Build condition nodes for each check
	var conditions []ConditionNode

	if needsPermissionCheck {
		// Add membership check condition
		membershipCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckMembershipStepID, constants.IsTeamMemberOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, membershipCheck)
	}

	if data.StopTime != "" {
		// Add stop-time check condition
		stopTimeCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckStopTimeStepID, constants.StopTimeOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, stopTimeCheck)
	}

	if data.Command != "" {
		// Add command position check condition
		commandPositionCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckCommandPositionStepID, constants.CommandPositionOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, commandPositionCheck)
	}

	// Build the final expression
	if len(conditions) == 0 {
		// This should never happen - it means pre-activation job was created without any checks
		// If we reach this point, it's a developer error in the compiler logic
		return nil, fmt.Errorf("developer error: pre-activation job created without permission check or stop-time configuration")
	} else if len(conditions) == 1 {
		// Single condition
		activatedNode = conditions[0]
	} else {
		// Multiple conditions - combine with AND
		activatedNode = conditions[0]
		for i := 1; i < len(conditions); i++ {
			activatedNode = buildAnd(activatedNode, conditions[i])
		}
	}

	// Render the expression with ${{ }} wrapper
	activatedExpression := fmt.Sprintf("${{ %s }}", activatedNode.Render())

	outputs := map[string]string{
		"activated": activatedExpression,
	}

	job := &Job{
		Name:        constants.PreActivationJobName,
		If:          data.If, // Use the existing condition (which may include alias checks)
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// buildActivationJob creates the preamble activation job that acts as a barrier for runtime conditions
func (c *Compiler) buildActivationJob(data *WorkflowData, preActivationJobCreated bool) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Team member check is now handled by the separate check_membership job
	// No inline role checks needed in the task job anymore

	// Add timestamp check for lock file vs source file
	steps = append(steps, "      - name: Check workflow file timestamps\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add the JavaScript script with proper indentation
	formattedScript := FormatJavaScriptForYAML(checkWorkflowTimestampScript)
	steps = append(steps, formattedScript...)

	// Use inlined compute-text script only if needed (no shared action)
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: compute-text\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Inline the JavaScript directly instead of using shared action
		steps = append(steps, FormatJavaScriptForYAML(getComputeTextScript())...)

		// Set up outputs
		outputs["text"] = "${{ steps.compute-text.outputs.text }}"
	}

	// Add reaction step if ai-reaction is configured and not "none"
	if data.AIReaction != "" && data.AIReaction != "none" {
		reactionCondition := buildReactionCondition()

		steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
		steps = append(steps, "        id: react\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", reactionCondition.Render()))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

		// Add environment variables
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_REACTION: %s\n", data.AIReaction))
		if data.Command != "" {
			steps = append(steps, fmt.Sprintf("          GH_AW_COMMAND: %s\n", data.Command))
		}
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add each line of the script with proper indentation
		formattedScript := FormatJavaScriptForYAML(addReactionAndEditCommentScript)
		steps = append(steps, formattedScript...)

		// Add reaction outputs
		outputs["reaction_id"] = "${{ steps.react.outputs.reaction-id }}"
		outputs["comment_id"] = "${{ steps.react.outputs.comment-id }}"
		outputs["comment_url"] = "${{ steps.react.outputs.comment-url }}"
		outputs["comment_repo"] = "${{ steps.react.outputs.comment-repo }}"
	}

	// If no steps have been added, add a dummy step to make the job valid
	// This can happen when the activation job is created only for an if condition
	if len(steps) == 0 {
		steps = append(steps, "      - run: echo \"Activation success\"\n")
	}

	// Build the conditional expression that validates activation status and other conditions
	var activationNeeds []string
	var activationCondition string

	if preActivationJobCreated {
		// Activation job depends on pre-activation job and checks the "activated" output
		activationNeeds = []string{constants.PreActivationJobName}
		activatedExpr := BuildEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.%s", constants.PreActivationJobName, constants.ActivatedOutput)),
			BuildStringLiteral("true"),
		)
		if data.If != "" {
			// Strip ${{ }} wrapper from data.If before combining
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := buildAnd(activatedExpr, ifExpr)
			activationCondition = combinedExpr.Render()
		} else {
			activationCondition = activatedExpr.Render()
		}
	} else {
		// No pre-activation check needed
		activationCondition = data.If
	}

	// Set permissions - add reaction permissions if reaction is configured and not "none"
	var permissions string
	if data.AIReaction != "" && data.AIReaction != "none" {
		perms := NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
			PermissionDiscussions:  PermissionWrite,
			PermissionIssues:       PermissionWrite,
			PermissionPullRequests: PermissionWrite,
		})
		permissions = perms.RenderToYAML()
	}

	job := &Job{
		Name:        constants.ActivationJobName,
		If:          activationCondition,
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
		Needs:       activationNeeds, // Depend on pre-activation job if it exists
	}

	return job, nil
}

// buildMainJob creates the main workflow job
func (c *Compiler) buildMainJob(data *WorkflowData, activationJobCreated bool, frontmatter map[string]any) (*Job, error) {
	var steps []string

	var jobCondition = data.If

	// Check if forks are allowed from frontmatter
	allowForks := c.getAllowForksFromFrontmatter(frontmatter)

	// Add fork prevention for pull_request triggers BEFORE clearing conditions for activation job
	// Agentic workflows should NEVER execute from forked repositories for security reasons
	// unless explicitly enabled with forks: true in frontmatter
	if c.hasPullRequestTrigger(data.On) && !allowForks {
		// Build fork prevention condition: pull request must be from the same repository
		forkPreventionCondition := BuildNotFromFork().Render()

		// Combine with existing condition if present
		if jobCondition != "" {
			// Build a combined condition tree
			existingCondition := jobCondition
			conditionTree := buildConditionTree(existingCondition, forkPreventionCondition)
			jobCondition = conditionTree.Render()
		} else {
			// No existing condition, just use fork prevention
			jobCondition = forkPreventionCondition
		}
	}

	// Note: Even if activationJobCreated is true, we keep the fork prevention condition
	// Fork prevention is a security requirement that should always be enforced
	// The activation job dependency ensures proper workflow ordering, but doesn't replace security checks

	// Permission checks are now handled by the separate check_membership job
	// No role checks needed in the main job

	// Build step content using the generateMainJobSteps helper method
	// but capture it into a string instead of writing directly
	var stepBuilder strings.Builder
	c.generateMainJobSteps(&stepBuilder, data)

	// Split the steps content into individual step entries
	stepsContent := stepBuilder.String()
	if stepsContent != "" {
		steps = append(steps, stepsContent)
	}

	var depends []string
	if activationJobCreated {
		depends = []string{constants.ActivationJobName} // Depend on the activation job only if it exists
	}

	// Build outputs for all engines (GH_AW_SAFE_OUTPUTS functionality)
	// Only include output if the workflow actually uses the safe-outputs feature
	var outputs map[string]string
	if data.SafeOutputs != nil {
		outputs = map[string]string{
			"output":       "${{ steps.collect_output.outputs.output }}",
			"output_types": "${{ steps.collect_output.outputs.output_types }}",
		}
	}

	// Build job-level environment variables for safe outputs
	var env map[string]string
	if data.SafeOutputs != nil {
		env = make(map[string]string)

		// Set GH_AW_SAFE_OUTPUTS to fixed path
		env["GH_AW_SAFE_OUTPUTS"] = "/tmp/gh-aw/safeoutputs/outputs.jsonl"

		// Set GH_AW_SAFE_OUTPUTS_CONFIG with the safe outputs configuration
		safeOutputConfig := generateSafeOutputsConfig(data)
		if safeOutputConfig != "" {
			// The JSON string needs to be properly quoted for YAML
			env["GH_AW_SAFE_OUTPUTS_CONFIG"] = fmt.Sprintf("%q", safeOutputConfig)
		}

		// Add asset-related environment variables if upload-assets is configured
		if data.SafeOutputs.UploadAssets != nil {
			env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", data.SafeOutputs.UploadAssets.MaxSizeKB)
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
		}
	}

	// Generate agent concurrency configuration
	agentConcurrency := GenerateJobConcurrencyConfig(data)

	job := &Job{
		Name:        constants.AgentJobName,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Container:   c.indentYAMLLines(data.Container, "    "),
		Services:    c.indentYAMLLines(data.Services, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Concurrency: c.indentYAMLLines(agentConcurrency, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}

	return job, nil
}

// extractJobsFromFrontmatter extracts job configuration from frontmatter
func (c *Compiler) extractJobsFromFrontmatter(frontmatter map[string]any) map[string]any {
	if jobs, exists := frontmatter["jobs"]; exists {
		if jobsMap, ok := jobs.(map[string]any); ok {
			return jobsMap
		}
	}
	return make(map[string]any)
}

// getAllowForksFromFrontmatter checks if forks are allowed from the frontmatter configuration
// Returns true if forks: true is set under any pull_request trigger, false otherwise (default is false)
func (c *Compiler) getAllowForksFromFrontmatter(frontmatter map[string]any) bool {
	if frontmatter == nil {
		return false // Default: forks not allowed
	}

	// Check if there's an "on" section in the frontmatter
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return false
	}

	// Check if "on" is an object (not a string)
	onMap, isOnMap := onValue.(map[string]any)
	if !isOnMap {
		return false
	}

	// Check all pull request related triggers for the forks field
	prTriggers := []string{"pull_request", "pull_request_target", "pull_request_review", "pull_request_review_comment"}

	for _, trigger := range prTriggers {
		if triggerValue, hasTrigger := onMap[trigger]; hasTrigger {
			// Check if trigger is an object with fork settings
			if triggerMap, isTriggerMap := triggerValue.(map[string]any); isTriggerMap {
				// Check for "forks" field (boolean)
				if forksValue, hasForks := triggerMap["forks"]; hasForks {
					// Convert to boolean
					if forksBool, isBool := forksValue.(bool); isBool {
						if forksBool {
							return true // Forks explicitly allowed
						}
					}
				}
			}
		}
	}

	return false // Default: forks not allowed
}

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section
func (c *Compiler) buildCustomJobs(data *WorkflowData) error {
	for jobName, jobConfig := range data.Jobs {
		if configMap, ok := jobConfig.(map[string]any); ok {
			job := &Job{
				Name: jobName,
			}

			// Extract job dependencies
			if needs, hasNeeds := configMap["needs"]; hasNeeds {
				if needsList, ok := needs.([]any); ok {
					for _, need := range needsList {
						if needStr, ok := need.(string); ok {
							job.Needs = append(job.Needs, needStr)
						}
					}
				} else if needStr, ok := needs.(string); ok {
					// Single dependency as string
					job.Needs = append(job.Needs, needStr)
				}
			}

			// Extract other job properties
			if runsOn, hasRunsOn := configMap["runs-on"]; hasRunsOn {
				if runsOnStr, ok := runsOn.(string); ok {
					job.RunsOn = fmt.Sprintf("runs-on: %s", runsOnStr)
				}
			}

			if ifCond, hasIf := configMap["if"]; hasIf {
				if ifStr, ok := ifCond.(string); ok {
					job.If = c.extractExpressionFromIfString(ifStr)
				}
			}

			// Add basic steps if specified
			if steps, hasSteps := configMap["steps"]; hasSteps {
				if stepsList, ok := steps.([]any); ok {
					for _, step := range stepsList {
						if stepMap, ok := step.(map[string]any); ok {
							// Apply action pinning before converting to YAML
							stepMap = ApplyActionPinToStep(stepMap, data)

							stepYAML, err := c.convertStepToYAML(stepMap)
							if err != nil {
								return fmt.Errorf("failed to convert step to YAML for job '%s': %w", jobName, err)
							}
							job.Steps = append(job.Steps, stepYAML)
						}
					}
				}
			}

			if err := c.jobManager.AddJob(job); err != nil {
				return fmt.Errorf("failed to add custom job '%s': %w", jobName, err)
			}
		}
	}

	return nil
}

// shouldAddCheckoutStep determines if the checkout step should be added based on permissions and custom steps
func (c *Compiler) shouldAddCheckoutStep(data *WorkflowData) bool {
	// Check condition 1: If custom steps already contain checkout, don't add another one
	if data.CustomSteps != "" && ContainsCheckout(data.CustomSteps) {
		return false // Custom steps already have checkout
	}

	// Check condition 2: If custom agent file is specified (via imports), checkout is required
	if data.AgentFile != "" {
		return true // Custom agent file requires checkout to access the file
	}

	// Check condition 3: If permissions don't grant contents access, don't add checkout
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		return false // No contents read access, so checkout is not needed
	}

	// If we get here, permissions allow contents access and custom steps (if any) don't contain checkout
	return true // Add checkout because it's needed and not already present
}
