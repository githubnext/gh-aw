package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputsToolsComputationLog = logger.New("workflow:safe_outputs_tools_computation")

type toolEnabledCheck struct {
	name    string
	enabled bool
}

func addEnabledTools(enabledTools map[string]struct{}, checks ...toolEnabledCheck) {
	for _, check := range checks {
		if check.enabled {
			enabledTools[check.name] = struct{}{}
		}
	}
}

func predefinedToolChecks(safeOutputs *SafeOutputsConfig) []toolEnabledCheck {
	return []toolEnabledCheck{
		{name: "create_issue", enabled: safeOutputs.CreateIssues != nil},
		{name: "create_agent_session", enabled: safeOutputs.CreateAgentSessions != nil},
		{name: "create_discussion", enabled: safeOutputs.CreateDiscussions != nil},
		{name: "update_discussion", enabled: safeOutputs.UpdateDiscussions != nil},
		{name: "close_discussion", enabled: safeOutputs.CloseDiscussions != nil},
		{name: "close_issue", enabled: safeOutputs.CloseIssues != nil},
		{name: "close_pull_request", enabled: safeOutputs.ClosePullRequests != nil},
		{name: "mark_pull_request_as_ready_for_review", enabled: safeOutputs.MarkPullRequestAsReadyForReview != nil},
		{name: "approve_workflow_run", enabled: safeOutputs.ApproveWorkflowRun != nil},
		{name: "dismiss_pull_request_review", enabled: safeOutputs.DismissPullRequestReview != nil},
		{name: "add_comment", enabled: safeOutputs.AddComments != nil},
		{name: "create_pull_request", enabled: safeOutputs.CreatePullRequests != nil},
		{name: "create_pull_request_review_comment", enabled: safeOutputs.CreatePullRequestReviewComments != nil},
		{name: "submit_pull_request_review", enabled: safeOutputs.SubmitPullRequestReview != nil},
		{name: "reply_to_pull_request_review_comment", enabled: safeOutputs.ReplyToPullRequestReviewComment != nil},
		{name: "resolve_pull_request_review_thread", enabled: safeOutputs.ResolvePullRequestReviewThread != nil},
		{name: "create_code_scanning_alert", enabled: safeOutputs.CreateCodeScanningAlerts != nil},
		{name: "autofix_code_scanning_alert", enabled: safeOutputs.AutofixCodeScanningAlert != nil},
		{name: "create_check_run", enabled: safeOutputs.CreateCheckRun != nil},
		{name: "add_labels", enabled: safeOutputs.AddLabels != nil},
		{name: "remove_labels", enabled: safeOutputs.RemoveLabels != nil},
		{name: "replace_label", enabled: safeOutputs.ReplaceLabel != nil},
		{name: "add_reviewer", enabled: safeOutputs.AddReviewer != nil},
		{name: "assign_milestone", enabled: safeOutputs.AssignMilestone != nil},
		{name: "assign_to_agent", enabled: safeOutputs.AssignToAgent != nil},
		{name: "assign_to_user", enabled: safeOutputs.AssignToUser != nil},
		{name: "unassign_from_user", enabled: safeOutputs.UnassignFromUser != nil},
		{name: "update_issue", enabled: safeOutputs.UpdateIssues != nil},
		{name: "update_pull_request", enabled: safeOutputs.UpdatePullRequests != nil},
		{name: "push_to_pull_request_branch", enabled: safeOutputs.PushToPullRequestBranch != nil},
		{name: "upload_asset", enabled: safeOutputs.UploadAssets != nil},
		{name: "upload_artifact", enabled: safeOutputs.UploadArtifact != nil},
		{name: "missing_tool", enabled: safeOutputs.MissingTool != nil},
		{name: "missing_data", enabled: safeOutputs.MissingData != nil},
		{name: "update_release", enabled: safeOutputs.UpdateRelease != nil},
		{name: "noop", enabled: safeOutputs.NoOp != nil},
		{name: "link_sub_issue", enabled: safeOutputs.LinkSubIssue != nil},
		{name: "hide_comment", enabled: safeOutputs.HideComment != nil},
		{name: "set_issue_type", enabled: safeOutputs.SetIssueType != nil},
		{name: "set_issue_field", enabled: safeOutputs.SetIssueField != nil},
		{name: "update_project", enabled: safeOutputs.UpdateProjects != nil},
		{name: "create_project_status_update", enabled: safeOutputs.CreateProjectStatusUpdates != nil},
		{name: "create_project", enabled: safeOutputs.CreateProjects != nil},
	}
}

// computeEnabledToolNames returns the set of predefined tool names that are enabled
// by the workflow's SafeOutputsConfig. Dynamic tools (dispatch-workflow, custom jobs,
// call-workflow) are excluded because they are generated separately.
func computeEnabledToolNames(data *WorkflowData) map[string]struct {
} {
	enabledTools := make(map[string]struct{})
	if data.SafeOutputs == nil {
		safeOutputsToolsComputationLog.Print("No safe outputs configuration, returning empty tool set")
		return enabledTools
	}

	addEnabledTools(enabledTools, predefinedToolChecks(data.SafeOutputs)...)

	// Add push_repo_memory tool if repo-memory is configured
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		enabledTools["push_repo_memory"] = struct{}{}
	}

	safeOutputsToolsComputationLog.Printf("Computed %d enabled safe output tool names", len(enabledTools))
	return enabledTools
}
