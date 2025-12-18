package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var individualJobsLog = logger.New("workflow:compiler_safe_output_jobs_individual")

// buildIndividualSafeOutputJobs builds individual jobs for each safe output type.
// This is the default mode that creates separate jobs for create_issue, add_comment, etc.
// Returns the list of job names that were created.
func (c *Compiler) buildIndividualSafeOutputJobs(data *WorkflowData, jobName, markdownPath string, threatDetectionEnabled bool) ([]string, error) {
	individualJobsLog.Print("Building individual safe output jobs")

	var safeOutputJobNames []string

	// Track which jobs create_issue, create_discussion, and create_pull_request were created
	var createIssueJobName string
	var createDiscussionJobName string
	var createPullRequestJobName string

	// Build create_issue job
	if data.SafeOutputs.CreateIssues != nil {
		job, err := c.buildCreateOutputIssueJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build create_issue job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add create_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
		createIssueJobName = job.Name
	}

	// Build create_discussion job
	if data.SafeOutputs.CreateDiscussions != nil {
		job, err := c.buildCreateOutputDiscussionJob(data, jobName, createIssueJobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build create_discussion job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add create_discussion job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
		createDiscussionJobName = job.Name
	}

	// Build close_discussion job
	if data.SafeOutputs.CloseDiscussions != nil {
		job, err := c.buildCreateOutputCloseDiscussionJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build close_discussion job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add close_discussion job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build close_issue job
	if data.SafeOutputs.CloseIssues != nil {
		job, err := c.buildCreateOutputCloseIssueJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build close_issue job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add close_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build close_pull_request job
	if data.SafeOutputs.ClosePullRequests != nil {
		job, err := c.buildCreateOutputClosePullRequestJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build close_pull_request job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add close_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build create_pull_request job
	if data.SafeOutputs.CreatePullRequests != nil {
		job, err := c.buildCreateOutputPullRequestJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build create_pull_request job: %w", err)
		}
		job.Needs = append(job.Needs, constants.ActivationJobName)
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add create_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
		createPullRequestJobName = job.Name
	}

	// Build add_comment job
	if data.SafeOutputs.AddComments != nil {
		job, err := c.buildCreateOutputCommentJob(data, jobName, createIssueJobName, createDiscussionJobName, createPullRequestJobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build add_comment job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add add_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build create_pr_review_comment job
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		job, err := c.buildCreateOutputPRReviewCommentJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build create_pr_review_comment job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add create_pr_review_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build create_code_scanning_alert job
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		job, err := c.buildCreateOutputCodeScanningAlertJob(data, jobName, markdownPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build create_code_scanning_alert job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add create_code_scanning_alert job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build add_labels job
	if data.SafeOutputs.AddLabels != nil {
		job, err := c.buildCreateOutputLabelJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build add_labels job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add add_labels job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build add_reviewer job
	if data.SafeOutputs.AddReviewer != nil {
		job, err := c.buildAddReviewerJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build add_reviewer job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add add_reviewer job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build assign_milestone job
	if data.SafeOutputs.AssignMilestone != nil {
		job, err := c.buildAssignMilestoneJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build assign_milestone job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add assign_milestone job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build assign_to_agent job
	if data.SafeOutputs.AssignToAgent != nil {
		job, err := c.buildAssignToAgentJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build assign_to_agent job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add assign_to_agent job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build assign_to_user job
	if data.SafeOutputs.AssignToUser != nil {
		job, err := c.buildAssignToUserJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build assign_to_user job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add assign_to_user job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build update_issue job
	if data.SafeOutputs.UpdateIssues != nil {
		job, err := c.buildCreateOutputUpdateIssueJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build update_issue job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add update_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build update_pull_request job
	if data.SafeOutputs.UpdatePullRequests != nil {
		job, err := c.buildCreateOutputUpdatePullRequestJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build update_pull_request job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add update_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build push_to_pull_request_branch job
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		job, err := c.buildCreateOutputPushToPullRequestBranchJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build push_to_pull_request_branch job: %w", err)
		}
		job.Needs = append(job.Needs, constants.ActivationJobName)
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add push_to_pull_request_branch job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build upload_assets job
	if data.SafeOutputs.UploadAssets != nil {
		job, err := c.buildCreateOutputUploadAssetJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build upload_assets job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add upload_assets job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build update_release job
	if data.SafeOutputs.UpdateRelease != nil {
		job, err := c.buildCreateOutputUpdateReleaseJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build update_release job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add update_release job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build link_sub_issue job
	if data.SafeOutputs.LinkSubIssue != nil {
		job, err := c.buildLinkSubIssueJob(data, jobName, createIssueJobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build link_sub_issue job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add link_sub_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build hide_comment job
	if data.SafeOutputs.HideComment != nil {
		job, err := c.buildHideCommentJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build hide_comment job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add hide_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build create_agent_task job
	if data.SafeOutputs.CreateAgentTasks != nil {
		job, err := c.buildCreateOutputAgentTaskJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build create_agent_task job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
			job.If = AddDetectionSuccessCheck(job.If)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add create_agent_task job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	// Build update_project job
	if data.SafeOutputs.UpdateProjects != nil {
		job, err := c.buildUpdateProjectJob(data, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to build update_project job: %w", err)
		}
		if threatDetectionEnabled {
			job.Needs = append(job.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add update_project job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, job.Name)
	}

	individualJobsLog.Printf("Built %d individual safe output jobs", len(safeOutputJobNames))
	return safeOutputJobNames, nil
}
