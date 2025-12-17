package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var compilerJobsSafeOutputsLog = logger.New("workflow:compiler_jobs_safe_outputs")

// This file contains the safe outputs job building logic extracted from compiler_jobs.go.
// It handles the construction of all safe output jobs including threat detection,
// create/update/close operations for issues, discussions, and pull requests.

func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName, markdownPath string) error {
	if data.SafeOutputs == nil {
		compilerJobsSafeOutputsLog.Print("No safe outputs configured, skipping safe outputs jobs")
		return nil
	}
	compilerJobsSafeOutputsLog.Print("Building safe outputs jobs")

	// Track whether threat detection job is enabled
	threatDetectionEnabled := false

	// Build threat detection job if enabled
	if data.SafeOutputs.ThreatDetection != nil {
		compilerJobsSafeOutputsLog.Print("Building threat detection job")
		detectionJob, err := c.buildThreatDetectionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build detection job: %w", err)
		}
		if err := c.jobManager.AddJob(detectionJob); err != nil {
			return fmt.Errorf("failed to add detection job: %w", err)
		}
		compilerJobsSafeOutputsLog.Printf("Successfully added threat detection job: %s", constants.DetectionJobName)
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
		createDiscussionJob, err := c.buildCreateOutputDiscussionJob(data, jobName, createIssueJobName)
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

	// Build close_issue job if safe-outputs.close-issue is configured
	if data.SafeOutputs.CloseIssues != nil {
		closeIssueJob, err := c.buildCreateOutputCloseIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build close_issue job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			closeIssueJob.Needs = append(closeIssueJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			closeIssueJob.If = AddDetectionSuccessCheck(closeIssueJob.If)
		}
		if err := c.jobManager.AddJob(closeIssueJob); err != nil {
			return fmt.Errorf("failed to add close_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, closeIssueJob.Name)
	}

	// Build close_pull_request job if safe-outputs.close-pull-request is configured
	if data.SafeOutputs.ClosePullRequests != nil {
		closePullRequestJob, err := c.buildCreateOutputClosePullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build close_pull_request job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			closePullRequestJob.Needs = append(closePullRequestJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			closePullRequestJob.If = AddDetectionSuccessCheck(closePullRequestJob.If)
		}
		if err := c.jobManager.AddJob(closePullRequestJob); err != nil {
			return fmt.Errorf("failed to add close_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, closePullRequestJob.Name)
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

	// Build add_reviewer job if output.add-reviewer is configured
	if data.SafeOutputs.AddReviewer != nil {
		addReviewerJob, err := c.buildAddReviewerJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build add_reviewer job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			addReviewerJob.Needs = append(addReviewerJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			addReviewerJob.If = AddDetectionSuccessCheck(addReviewerJob.If)
		}
		if err := c.jobManager.AddJob(addReviewerJob); err != nil {
			return fmt.Errorf("failed to add add_reviewer job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, addReviewerJob.Name)
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

	// Build assign_to_agent job if output.assign-to-agent is configured
	if data.SafeOutputs.AssignToAgent != nil {
		assignToAgentJob, err := c.buildAssignToAgentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build assign_to_agent job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			assignToAgentJob.Needs = append(assignToAgentJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			assignToAgentJob.If = AddDetectionSuccessCheck(assignToAgentJob.If)
		}
		if err := c.jobManager.AddJob(assignToAgentJob); err != nil {
			return fmt.Errorf("failed to add assign_to_agent job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, assignToAgentJob.Name)
	}

	// Build assign_to_user job if output.assign-to-user is configured
	if data.SafeOutputs.AssignToUser != nil {
		assignToUserJob, err := c.buildAssignToUserJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build assign_to_user job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			assignToUserJob.Needs = append(assignToUserJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			assignToUserJob.If = AddDetectionSuccessCheck(assignToUserJob.If)
		}
		if err := c.jobManager.AddJob(assignToUserJob); err != nil {
			return fmt.Errorf("failed to add assign_to_user job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, assignToUserJob.Name)
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

	// Build update_pull_request job if output.update-pull-request is configured
	if data.SafeOutputs.UpdatePullRequests != nil {
		updatePullRequestJob, err := c.buildCreateOutputUpdatePullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_pull_request job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			updatePullRequestJob.Needs = append(updatePullRequestJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			updatePullRequestJob.If = AddDetectionSuccessCheck(updatePullRequestJob.If)
		}
		if err := c.jobManager.AddJob(updatePullRequestJob); err != nil {
			return fmt.Errorf("failed to add update_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, updatePullRequestJob.Name)
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

	// Note: missing_tool processing is now handled inside the conclusion job, not as a separate job

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

	// Build link_sub_issue job if safe-outputs.link-sub-issue is configured
	if data.SafeOutputs.LinkSubIssue != nil {
		linkSubIssueJob, err := c.buildLinkSubIssueJob(data, jobName, createIssueJobName)
		if err != nil {
			return fmt.Errorf("failed to build link_sub_issue job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			linkSubIssueJob.Needs = append(linkSubIssueJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			linkSubIssueJob.If = AddDetectionSuccessCheck(linkSubIssueJob.If)
		}
		if err := c.jobManager.AddJob(linkSubIssueJob); err != nil {
			return fmt.Errorf("failed to add link_sub_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, linkSubIssueJob.Name)
	}

	// Build hide_comment job if safe-outputs.hide-comment is configured
	if data.SafeOutputs.HideComment != nil {
		hideCommentJob, err := c.buildHideCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build hide_comment job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			hideCommentJob.Needs = append(hideCommentJob.Needs, constants.DetectionJobName)
			// Add detection success check to the job condition
			hideCommentJob.If = AddDetectionSuccessCheck(hideCommentJob.If)
		}
		if err := c.jobManager.AddJob(hideCommentJob); err != nil {
			return fmt.Errorf("failed to add hide_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, hideCommentJob.Name)
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

	// Build safe-jobs if configured
	// Safe-jobs should depend on agent job (always) AND detection job (if threat detection is enabled)
	// These custom safe-jobs should also be included in the conclusion job's dependencies
	safeJobNames, err := c.buildSafeJobs(data, threatDetectionEnabled)
	if err != nil {
		return fmt.Errorf("failed to build safe-jobs: %w", err)
	}
	// Add custom safe-job names to the list of safe output jobs
	safeOutputJobNames = append(safeOutputJobNames, safeJobNames...)
	compilerJobsSafeOutputsLog.Printf("Added %d custom safe-job names to conclusion dependencies", len(safeJobNames))

	// Build conclusion job if add-comment is configured OR if command trigger is configured with reactions
	// This job runs last, after all safe output jobs (and push_repo_memory if configured), to update the activation comment on failure
	// The buildConclusionJob function itself will decide whether to create the job based on the configuration
	conclusionJob, err := c.buildConclusionJob(data, jobName, safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build conclusion job: %w", err)
	}
	if conclusionJob != nil {
		// If push_repo_memory job exists, conclusion should depend on it
		// Check if the job was already created (it's created in buildJobs)
		if _, exists := c.jobManager.GetJob("push_repo_memory"); exists {
			conclusionJob.Needs = append(conclusionJob.Needs, "push_repo_memory")
			compilerJobsSafeOutputsLog.Printf("Added push_repo_memory dependency to conclusion job")
		}
		if err := c.jobManager.AddJob(conclusionJob); err != nil {
			return fmt.Errorf("failed to add conclusion job: %w", err)
		}
	}

	return nil
}
