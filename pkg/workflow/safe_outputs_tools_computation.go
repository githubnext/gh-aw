package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputsToolsComputationLog = logger.New("workflow:safe_outputs_tools_computation")

// computeEnabledToolNames returns the set of predefined tool names that are enabled
// by the workflow's SafeOutputsConfig. Dynamic tools (dispatch-workflow, custom jobs,
// call-workflow) are excluded because they are generated separately.
func computeEnabledToolNames(data *WorkflowData) map[string]struct{} {
	enabledTools := make(map[string]struct{})
	if data.SafeOutputs == nil {
		safeOutputsToolsComputationLog.Print("No safe outputs configuration, returning empty tool set")
		return enabledTools
	}

	addSafeOutputCreationToolNames(enabledTools, data.SafeOutputs)
	addSafeOutputPRToolNames(enabledTools, data.SafeOutputs)
	addSafeOutputLabelAndAssignmentToolNames(enabledTools, data.SafeOutputs)
	addSafeOutputStatusAndProjectToolNames(enabledTools, data.SafeOutputs)
	addSafeOutputMemoryToolNames(enabledTools, data)

	safeOutputsToolsComputationLog.Printf("Computed %d enabled safe output tool names", len(enabledTools))
	return enabledTools
}

func addSafeOutputCreationToolNames(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) {
	if safeOutputs.CreateIssues != nil {
		enabledTools["create_issue"] = struct{}{}
	}
	if safeOutputs.CreateAgentSessions != nil {
		enabledTools["create_agent_session"] = struct{}{}
	}
	if safeOutputs.CreateDiscussions != nil {
		enabledTools["create_discussion"] = struct{}{}
	}
	if safeOutputs.UpdateDiscussions != nil {
		enabledTools["update_discussion"] = struct{}{}
	}
	if safeOutputs.CloseDiscussions != nil {
		enabledTools["close_discussion"] = struct{}{}
	}
	if safeOutputs.CloseIssues != nil {
		enabledTools["close_issue"] = struct{}{}
	}
	if safeOutputs.AddComments != nil {
		enabledTools["add_comment"] = struct{}{}
	}
	if safeOutputs.UploadAssets != nil {
		enabledTools["upload_asset"] = struct{}{}
	}
	if safeOutputs.UploadArtifact != nil {
		enabledTools["upload_artifact"] = struct{}{}
	}
}

func addSafeOutputPRToolNames(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) {
	if safeOutputs.ClosePullRequests != nil {
		enabledTools["close_pull_request"] = struct{}{}
	}
	if safeOutputs.MarkPullRequestAsReadyForReview != nil {
		enabledTools["mark_pull_request_as_ready_for_review"] = struct{}{}
	}
	if safeOutputs.DismissPullRequestReview != nil {
		enabledTools["dismiss_pull_request_review"] = struct{}{}
	}
	if safeOutputs.CreatePullRequests != nil {
		enabledTools["create_pull_request"] = struct{}{}
	}
	if safeOutputs.CreatePullRequestReviewComments != nil {
		enabledTools["create_pull_request_review_comment"] = struct{}{}
	}
	if safeOutputs.SubmitPullRequestReview != nil {
		enabledTools["submit_pull_request_review"] = struct{}{}
	}
	if safeOutputs.ReplyToPullRequestReviewComment != nil {
		enabledTools["reply_to_pull_request_review_comment"] = struct{}{}
	}
	if safeOutputs.ResolvePullRequestReviewThread != nil {
		enabledTools["resolve_pull_request_review_thread"] = struct{}{}
	}
	if safeOutputs.UpdatePullRequests != nil {
		enabledTools["update_pull_request"] = struct{}{}
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		enabledTools["push_to_pull_request_branch"] = struct{}{}
	}
}

func addSafeOutputLabelAndAssignmentToolNames(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) {
	if safeOutputs.AddLabels != nil {
		enabledTools["add_labels"] = struct{}{}
	}
	if safeOutputs.RemoveLabels != nil {
		enabledTools["remove_labels"] = struct{}{}
	}
	if safeOutputs.ReplaceLabel != nil {
		enabledTools["replace_label"] = struct{}{}
	}
	if safeOutputs.AddReviewer != nil {
		enabledTools["add_reviewer"] = struct{}{}
	}
	if safeOutputs.AssignMilestone != nil {
		enabledTools["assign_milestone"] = struct{}{}
	}
	if safeOutputs.AssignToAgent != nil {
		enabledTools["assign_to_agent"] = struct{}{}
	}
	if safeOutputs.AssignToUser != nil {
		enabledTools["assign_to_user"] = struct{}{}
	}
	if safeOutputs.UnassignFromUser != nil {
		enabledTools["unassign_from_user"] = struct{}{}
	}
}

func addSafeOutputStatusAndProjectToolNames(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) {
	if safeOutputs.CreateCodeScanningAlerts != nil {
		enabledTools["create_code_scanning_alert"] = struct{}{}
	}
	if safeOutputs.AutofixCodeScanningAlert != nil {
		enabledTools["autofix_code_scanning_alert"] = struct{}{}
	}
	if safeOutputs.CreateCheckRun != nil {
		enabledTools["create_check_run"] = struct{}{}
	}
	if safeOutputs.UpdateIssues != nil {
		enabledTools["update_issue"] = struct{}{}
	}
	if safeOutputs.MissingTool != nil {
		enabledTools["missing_tool"] = struct{}{}
	}
	if safeOutputs.MissingData != nil {
		enabledTools["missing_data"] = struct{}{}
	}
	if safeOutputs.UpdateRelease != nil {
		enabledTools["update_release"] = struct{}{}
	}
	if safeOutputs.NoOp != nil {
		enabledTools["noop"] = struct{}{}
	}
	if safeOutputs.LinkSubIssue != nil {
		enabledTools["link_sub_issue"] = struct{}{}
	}
	if safeOutputs.HideComment != nil {
		enabledTools["hide_comment"] = struct{}{}
	}
	if safeOutputs.SetIssueType != nil {
		enabledTools["set_issue_type"] = struct{}{}
	}
	if safeOutputs.SetIssueField != nil {
		enabledTools["set_issue_field"] = struct{}{}
	}
	if safeOutputs.UpdateProjects != nil {
		enabledTools["update_project"] = struct{}{}
	}
	if safeOutputs.CreateProjectStatusUpdates != nil {
		enabledTools["create_project_status_update"] = struct{}{}
	}
	if safeOutputs.CreateProjects != nil {
		enabledTools["create_project"] = struct{}{}
	}
}

func addSafeOutputMemoryToolNames(enabledTools map[string]struct{}, data *WorkflowData) {
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		enabledTools["push_repo_memory"] = struct{}{}
	}
}
