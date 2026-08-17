package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputsToolsComputationLog = logger.New("workflow:safe_outputs_tools_computation")

// addEnabledTool adds a single tool name to enabledTools if enabled is true.
func addEnabledTool(enabledTools map[string]struct{}, name string, enabled bool) {
	if enabled {
		enabledTools[name] = struct{}{}
	}
}

// addPredefinedTools adds all predefined safe-output tool names to enabledTools
// directly, without materializing an intermediate slice, since this runs on
// every compilation.
func addPredefinedTools(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) {
	addEnabledTool(enabledTools, "create_issue", safeOutputs.CreateIssues != nil)
	addEnabledTool(enabledTools, "create_agent_session", safeOutputs.CreateAgentSessions != nil)
	addEnabledTool(enabledTools, "create_discussion", safeOutputs.CreateDiscussions != nil)
	addEnabledTool(enabledTools, "update_discussion", safeOutputs.UpdateDiscussions != nil)
	addEnabledTool(enabledTools, "close_discussion", safeOutputs.CloseDiscussions != nil)
	addEnabledTool(enabledTools, "close_issue", safeOutputs.CloseIssues != nil)
	addEnabledTool(enabledTools, "close_pull_request", safeOutputs.ClosePullRequests != nil)
	addEnabledTool(enabledTools, "mark_pull_request_as_ready_for_review", safeOutputs.MarkPullRequestAsReadyForReview != nil)
	addEnabledTool(enabledTools, "approve_workflow_run", safeOutputs.ApproveWorkflowRun != nil)
	addEnabledTool(enabledTools, "dismiss_pull_request_review", safeOutputs.DismissPullRequestReview != nil)
	addEnabledTool(enabledTools, "add_comment", safeOutputs.AddComments != nil)
	addEnabledTool(enabledTools, "create_pull_request", safeOutputs.CreatePullRequests != nil)
	addEnabledTool(enabledTools, "create_pull_request_review_comment", safeOutputs.CreatePullRequestReviewComments != nil)
	addEnabledTool(enabledTools, "submit_pull_request_review", safeOutputs.SubmitPullRequestReview != nil)
	addEnabledTool(enabledTools, "reply_to_pull_request_review_comment", safeOutputs.ReplyToPullRequestReviewComment != nil)
	addEnabledTool(enabledTools, "resolve_pull_request_review_thread", safeOutputs.ResolvePullRequestReviewThread != nil)
	addEnabledTool(enabledTools, "create_code_scanning_alert", safeOutputs.CreateCodeScanningAlerts != nil)
	addEnabledTool(enabledTools, "autofix_code_scanning_alert", safeOutputs.AutofixCodeScanningAlert != nil)
	addEnabledTool(enabledTools, "create_check_run", safeOutputs.CreateCheckRun != nil)
	addEnabledTool(enabledTools, "add_labels", safeOutputs.AddLabels != nil)
	addEnabledTool(enabledTools, "remove_labels", safeOutputs.RemoveLabels != nil)
	addEnabledTool(enabledTools, "replace_label", safeOutputs.ReplaceLabel != nil)
	addEnabledTool(enabledTools, "add_reviewer", safeOutputs.AddReviewer != nil)
	addEnabledTool(enabledTools, "assign_milestone", safeOutputs.AssignMilestone != nil)
	addEnabledTool(enabledTools, "assign_to_agent", safeOutputs.AssignToAgent != nil)
	addEnabledTool(enabledTools, "assign_to_user", safeOutputs.AssignToUser != nil)
	addEnabledTool(enabledTools, "unassign_from_user", safeOutputs.UnassignFromUser != nil)
	addEnabledTool(enabledTools, "update_issue", safeOutputs.UpdateIssues != nil)
	addEnabledTool(enabledTools, "update_pull_request", safeOutputs.UpdatePullRequests != nil)
	addEnabledTool(enabledTools, "push_to_pull_request_branch", safeOutputs.PushToPullRequestBranch != nil)
	addEnabledTool(enabledTools, "upload_asset", safeOutputs.UploadAssets != nil)
	addEnabledTool(enabledTools, "upload_artifact", safeOutputs.UploadArtifact != nil)
	addEnabledTool(enabledTools, "missing_tool", safeOutputs.MissingTool != nil)
	addEnabledTool(enabledTools, "missing_data", safeOutputs.MissingData != nil)
	addEnabledTool(enabledTools, "update_release", safeOutputs.UpdateRelease != nil)
	addEnabledTool(enabledTools, "noop", safeOutputs.NoOp != nil)
	addEnabledTool(enabledTools, "link_sub_issue", safeOutputs.LinkSubIssue != nil)
	addEnabledTool(enabledTools, "hide_comment", safeOutputs.HideComment != nil)
	addEnabledTool(enabledTools, "set_issue_type", safeOutputs.SetIssueType != nil)
	addEnabledTool(enabledTools, "set_issue_field", safeOutputs.SetIssueField != nil)
	addEnabledTool(enabledTools, "update_project", safeOutputs.UpdateProjects != nil)
	addEnabledTool(enabledTools, "create_project_status_update", safeOutputs.CreateProjectStatusUpdates != nil)
	addEnabledTool(enabledTools, "create_project", safeOutputs.CreateProjects != nil)
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

	addPredefinedTools(enabledTools, data.SafeOutputs)

	// Add push_repo_memory tool if repo-memory is configured
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		enabledTools["push_repo_memory"] = struct{}{}
	}

	safeOutputsToolsComputationLog.Printf("Computed %d enabled safe output tool names", len(enabledTools))
	return enabledTools
}
