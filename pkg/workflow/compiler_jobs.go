package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var compilerJobsLog = logger.New("workflow:compiler_jobs")

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
	compilerJobsLog.Printf("Building jobs for workflow: %s", markdownPath)

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
	hasSkipIfMatch := data.SkipIfMatch != nil
	compilerJobsLog.Printf("Job configuration: needsPermissionCheck=%v, hasStopTime=%v, hasSkipIfMatch=%v, hasCommand=%v", needsPermissionCheck, hasStopTime, hasSkipIfMatch, data.Command != "")

	// Determine if we need to add workflow_run repository safety check
	// Add the check if the agentic workflow declares a workflow_run trigger
	// This prevents cross-repository workflow_run attacks
	var workflowRunRepoSafety string
	if c.hasWorkflowRunTrigger(frontmatter) {
		workflowRunRepoSafety = c.buildWorkflowRunRepoSafetyCondition()
		compilerJobsLog.Print("Adding workflow_run repository safety check")
	}

	// Extract lock filename for timestamp check
	lockFilename := filepath.Base(strings.TrimSuffix(markdownPath, ".md") + ".lock.yml")

	// Build pre-activation job if needed (combines membership checks, stop-time validation, skip-if-match check, and command position check)
	var preActivationJobCreated bool
	hasCommandTrigger := data.Command != ""
	if needsPermissionCheck || hasStopTime || hasSkipIfMatch || hasCommandTrigger {
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
		activationJob, err := c.buildActivationJob(data, preActivationJobCreated, workflowRunRepoSafety, lockFilename)
		if err != nil {
			return fmt.Errorf("failed to build activation job: %w", err)
		}
		if err := c.jobManager.AddJob(activationJob); err != nil {
			return fmt.Errorf("failed to add activation job: %w", err)
		}
		activationJobCreated = true
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, activationJobCreated)
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
	if err := c.buildCustomJobs(data, activationJobCreated); err != nil {
		return fmt.Errorf("failed to build custom jobs: %w", err)
	}

	compilerJobsLog.Print("Successfully built all jobs for workflow")
	return nil
}

// buildSafeOutputsJobs creates all safe outputs jobs if configured
func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName, markdownPath string) error {
	if data.SafeOutputs == nil {
		return nil
	}
	compilerJobsLog.Print("Building safe outputs jobs")

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

	// Track safe output job names to establish dependencies for conclusion job
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
			// Add detection success check to the job condition
			createIssueJob.If = AddDetectionSuccessCheck(createIssueJob.If)
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
			// Add detection success check to the job condition
			createDiscussionJob.If = AddDetectionSuccessCheck(createDiscussionJob.If)
		}
		if err := c.jobManager.AddJob(createDiscussionJob); err != nil {
			return fmt.Errorf("failed to add create_discussion job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createDiscussionJob.Name)
		createDiscussionJobName = createDiscussionJob.Name
	}

	// Build close_discussion job if safe-outputs.close-discussion is configured
	if data.SafeOutputs.CloseDiscussions != nil {
		closeDiscussionJob, err := c.buildCreateOutputCloseDiscussionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build close_discussion job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			closeDiscussionJob.Needs = append(closeDiscussionJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			closeDiscussionJob.If = AddDetectionSuccessCheck(closeDiscussionJob.If)
		}
		if err := c.jobManager.AddJob(closeDiscussionJob); err != nil {
			return fmt.Errorf("failed to add close_discussion job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, closeDiscussionJob.Name)
	}

	// Build create_pull_request job if output.create-pull-request is configured
	// NOTE: This is built BEFORE add_comment so that add_comment can depend on it
	if data.SafeOutputs.CreatePullRequests != nil {
		createPullRequestJob, err := c.buildCreateOutputPullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pull_request job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always), activation job (for comment linking), AND detection job (if enabled)
		createPullRequestJob.Needs = append(createPullRequestJob.Needs, constants.ActivationJobName)
		if threatDetectionEnabled {
			createPullRequestJob.Needs = append(createPullRequestJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			createPullRequestJob.If = AddDetectionSuccessCheck(createPullRequestJob.If)
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
			// Add detection success check to the job condition
			createCommentJob.If = AddDetectionSuccessCheck(createCommentJob.If)
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
			// Add detection success check to the job condition
			createPRReviewCommentJob.If = AddDetectionSuccessCheck(createPRReviewCommentJob.If)
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
			// Add detection success check to the job condition
			createCodeScanningAlertJob.If = AddDetectionSuccessCheck(createCodeScanningAlertJob.If)
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
			// Add detection success check to the job condition
			addLabelsJob.If = AddDetectionSuccessCheck(addLabelsJob.If)
		}
		if err := c.jobManager.AddJob(addLabelsJob); err != nil {
			return fmt.Errorf("failed to add add_labels job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, addLabelsJob.Name)
	}

	// Build assign_milestone job if output.assign-milestone is configured
	if data.SafeOutputs.AssignMilestone != nil {
		assignMilestoneJob, err := c.buildAssignMilestoneJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build assign_milestone job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			assignMilestoneJob.Needs = append(assignMilestoneJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			assignMilestoneJob.If = AddDetectionSuccessCheck(assignMilestoneJob.If)
		}
		if err := c.jobManager.AddJob(assignMilestoneJob); err != nil {
			return fmt.Errorf("failed to add assign_milestone job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, assignMilestoneJob.Name)
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
			// Add detection success check to the job condition
			updateIssueJob.If = AddDetectionSuccessCheck(updateIssueJob.If)
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
		// Safe-output jobs should depend on agent job (always), activation job (for comment linking), AND detection job (if enabled)
		pushToBranchJob.Needs = append(pushToBranchJob.Needs, constants.ActivationJobName)
		if threatDetectionEnabled {
			pushToBranchJob.Needs = append(pushToBranchJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			pushToBranchJob.If = AddDetectionSuccessCheck(pushToBranchJob.If)
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
			// Add detection success check to the job condition
			missingToolJob.If = AddDetectionSuccessCheck(missingToolJob.If)
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
			// Add detection success check to the job condition
			uploadAssetsJob.If = AddDetectionSuccessCheck(uploadAssetsJob.If)
		}
		if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
			return fmt.Errorf("failed to add upload_assets job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, uploadAssetsJob.Name)
	}

	// Build update_release job if output.update-release is configured
	if data.SafeOutputs.UpdateRelease != nil {
		updateReleaseJob, err := c.buildCreateOutputUpdateReleaseJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_release job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			updateReleaseJob.Needs = append(updateReleaseJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			updateReleaseJob.If = AddDetectionSuccessCheck(updateReleaseJob.If)
		}
		if err := c.jobManager.AddJob(updateReleaseJob); err != nil {
			return fmt.Errorf("failed to add update_release job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, updateReleaseJob.Name)
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
			// Add detection success check to the job condition
			createAgentTaskJob.If = AddDetectionSuccessCheck(createAgentTaskJob.If)
		}
		if err := c.jobManager.AddJob(createAgentTaskJob); err != nil {
			return fmt.Errorf("failed to add create_agent_task job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createAgentTaskJob.Name)
	}

	// Build update_project job if safe-outputs.update-project is configured
	if data.SafeOutputs.UpdateProjects != nil {
		updateProjectJob, err := c.buildUpdateProjectJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_project job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			updateProjectJob.Needs = append(updateProjectJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(updateProjectJob); err != nil {
			return fmt.Errorf("failed to add update_project job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, updateProjectJob.Name)
	}

	// Note: noop processing is now handled inside the conclusion job, not as a separate job

	// Build conclusion job if add-comment is configured OR if command trigger is configured with reactions
	// This job runs last, after all safe output jobs, to update the activation comment on failure
	// The buildConclusionJob function itself will decide whether to create the job based on the configuration
	conclusionJob, err := c.buildConclusionJob(data, jobName, safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build conclusion job: %w", err)
	}
	if conclusionJob != nil {
		if err := c.jobManager.AddJob(conclusionJob); err != nil {
			return fmt.Errorf("failed to add conclusion job: %w", err)
		}
	}

	return nil
}

// buildPreActivationJob creates a unified pre-activation job that combines membership checks and stop-time validation
// This job exposes a single "activated" output that indicates whether the workflow should proceed
func (c *Compiler) buildPreActivationJob(data *WorkflowData, needsPermissionCheck bool) (*Job, error) {
	compilerJobsLog.Printf("Building pre-activation job: needsPermissionCheck=%v, hasStopTime=%v", needsPermissionCheck, data.StopTime != "")
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

	// Add skip-if-match check if configured
	if data.SkipIfMatch != nil {
		// Extract workflow name for the skip-if-match check
		workflowName := data.Name

		steps = append(steps, "      - name: Check skip-if-match query\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckSkipIfMatchStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_QUERY: %q\n", data.SkipIfMatch.Query))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, fmt.Sprintf("          GH_AW_SKIP_MAX_MATCHES: \"%d\"\n", data.SkipIfMatch.Max))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkSkipIfMatchScript)
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

	if data.SkipIfMatch != nil {
		// Add skip-if-match check condition
		skipCheckOk := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckSkipIfMatchStepID, constants.SkipCheckOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, skipCheckOk)
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

	// Pre-activation job uses the user's original if condition (data.If)
	// The workflow_run safety check is NOT applied here - it's only on the activation job
	jobIfCondition := data.If

	job := &Job{
		Name:        constants.PreActivationJobName,
		If:          jobIfCondition,
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// buildActivationJob creates the preamble activation job that acts as a barrier for runtime conditions
// The workflow_run repository safety check is applied exclusively to this job
func (c *Compiler) buildActivationJob(data *WorkflowData, preActivationJobCreated bool, workflowRunRepoSafety string, lockFilename string) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Team member check is now handled by the separate check_membership job
	// No inline role checks needed in the task job anymore

	// Add timestamp check for lock file vs source file using GitHub API
	// No checkout step needed - uses GitHub API to check commit times
	steps = append(steps, "      - name: Check workflow file timestamps\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_FILE: \"%s\"\n", lockFilename))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add the JavaScript script with proper indentation (using API-based version)
	formattedScript := FormatJavaScriptForYAML(checkWorkflowTimestampAPIScript)
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

	// Always declare comment_id and comment_repo outputs to avoid actionlint errors
	// These will be empty if no reaction is configured, and the scripts handle empty values gracefully
	// Use plain empty strings (quoted) to avoid triggering security scanners like zizmor
	if _, exists := outputs["comment_id"]; !exists {
		outputs["comment_id"] = `""`
	}
	if _, exists := outputs["comment_repo"]; !exists {
		outputs["comment_repo"] = `""`
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
		// No pre-activation check needed, use user's if condition
		activationCondition = data.If
	}

	// Apply workflow_run repository safety check exclusively to activation job
	// This check is combined with any existing activation condition
	if workflowRunRepoSafety != "" {
		activationCondition = c.combineJobIfConditions(activationCondition, workflowRunRepoSafety)
	}

	// Set permissions - activation job always needs contents:read for GitHub API access
	// Also add reaction permissions if reaction is configured and not "none"
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead, // Always needed for GitHub API access to check file commits
	}

	if data.AIReaction != "" && data.AIReaction != "none" {
		permsMap[PermissionDiscussions] = PermissionWrite
		permsMap[PermissionIssues] = PermissionWrite
		permsMap[PermissionPullRequests] = PermissionWrite
	}

	perms := NewPermissionsFromMap(permsMap)
	permissions := perms.RenderToYAML()

	// Set environment if manual-approval is configured
	var environment string
	if data.ManualApproval != "" {
		environment = fmt.Sprintf("environment: %s", data.ManualApproval)
	}

	job := &Job{
		Name:                       constants.ActivationJobName,
		If:                         activationCondition,
		HasWorkflowRunSafetyChecks: workflowRunRepoSafety != "", // Mark job as having workflow_run safety checks
		RunsOn:                     c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:                permissions,
		Environment:                environment,
		Steps:                      steps,
		Outputs:                    outputs,
		Needs:                      activationNeeds, // Depend on pre-activation job if it exists
	}

	return job, nil
}

// buildMainJob creates the main workflow job
func (c *Compiler) buildMainJob(data *WorkflowData, activationJobCreated bool) (*Job, error) {
	log.Printf("Building main job for workflow: %s", data.Name)
	var steps []string

	var jobCondition = data.If
	if activationJobCreated {
		jobCondition = "" // Main job depends on activation job, so no need for inline condition
	}

	// Note: workflow_run repository safety check is applied exclusively to activation job

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

	// Add custom jobs as dependencies if they exist
	// This allows the agent job to wait for custom jobs to complete before running
	if data.Jobs != nil {
		for jobName := range data.Jobs {
			depends = append(depends, jobName)
		}
	}

	// Build outputs for all engines (GH_AW_SAFE_OUTPUTS functionality)
	// Only include output if the workflow actually uses the safe-outputs feature
	var outputs map[string]string
	if data.SafeOutputs != nil {
		outputs = map[string]string{
			"output":       "${{ steps.collect_output.outputs.output }}",
			"output_types": "${{ steps.collect_output.outputs.output_types }}",
			"has_patch":    "${{ steps.collect_output.outputs.has_patch }}",
		}
	}

	// Build job-level environment variables for safe outputs
	var env map[string]string
	if data.SafeOutputs != nil {
		env = make(map[string]string)

		// Set GH_AW_SAFE_OUTPUTS to fixed path
		env["GH_AW_SAFE_OUTPUTS"] = "/tmp/gh-aw/safeoutputs/outputs.jsonl"

		// Config is written to /tmp/gh-aw/safeoutputs/config.json file, not passed as env var

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

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section
func (c *Compiler) buildCustomJobs(data *WorkflowData, activationJobCreated bool) error {
	compilerJobsLog.Printf("Building %d custom jobs", len(data.Jobs))
	for jobName, jobConfig := range data.Jobs {
		if configMap, ok := jobConfig.(map[string]any); ok {
			job := &Job{
				Name: jobName,
			}

			// Extract job dependencies
			hasExplicitNeeds := false
			if needs, hasNeeds := configMap["needs"]; hasNeeds {
				hasExplicitNeeds = true
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

			// If no explicit needs and activation job exists, automatically add activation as dependency
			// This ensures custom jobs wait for workflow validation before executing
			if !hasExplicitNeeds && activationJobCreated {
				job.Needs = append(job.Needs, constants.ActivationJobName)
				compilerJobsLog.Printf("Added automatic dependency: custom job '%s' now depends on '%s'", jobName, constants.ActivationJobName)
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

			// Extract permissions
			if permissions, hasPermissions := configMap["permissions"]; hasPermissions {
				if permsMap, ok := permissions.(map[string]any); ok {
					// Use gopkg.in/yaml.v3 to marshal permissions
					yamlBytes, err := yaml.Marshal(permsMap)
					if err != nil {
						return fmt.Errorf("failed to convert permissions to YAML for job '%s': %w", jobName, err)
					}
					// Indent the YAML properly for job-level permissions
					permsYAML := string(yamlBytes)
					lines := strings.Split(strings.TrimSpace(permsYAML), "\n")
					var formattedPerms strings.Builder
					formattedPerms.WriteString("permissions:\n")
					for _, line := range lines {
						formattedPerms.WriteString("      " + line + "\n")
					}
					job.Permissions = formattedPerms.String()
				}
			}

			// Check if this is a reusable workflow call
			if uses, hasUses := configMap["uses"]; hasUses {
				if usesStr, ok := uses.(string); ok {
					job.Uses = usesStr

					// Extract with parameters for reusable workflow
					if with, hasWith := configMap["with"]; hasWith {
						if withMap, ok := with.(map[string]any); ok {
							job.With = withMap
						}
					}

					// Extract secrets for reusable workflow
					if secrets, hasSecrets := configMap["secrets"]; hasSecrets {
						if secretsMap, ok := secrets.(map[string]any); ok {
							job.Secrets = make(map[string]string)
							for key, val := range secretsMap {
								if valStr, ok := val.(string); ok {
									job.Secrets[key] = valStr
								}
							}
						}
					}
				}
			} else {
				// Add basic steps if specified (only for non-reusable workflow jobs)
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
		log.Print("Skipping checkout step: custom steps already contain checkout")
		return false // Custom steps already have checkout
	}

	// Check condition 2: If custom agent file is specified (via imports), checkout is required
	if data.AgentFile != "" {
		log.Printf("Adding checkout step: custom agent file specified: %s", data.AgentFile)
		return true // Custom agent file requires checkout to access the file
	}

	// Check condition 3: If permissions don't grant contents access, don't add checkout
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		log.Print("Skipping checkout step: no contents read access in permissions")
		return false // No contents read access, so checkout is not needed
	}

	// If we get here, permissions allow contents access and custom steps (if any) don't contain checkout
	return true // Add checkout because it's needed and not already present
}
