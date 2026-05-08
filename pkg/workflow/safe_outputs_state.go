package workflow

import (
	"fmt"
)

// ========================================
// Safe Output State Inspection
// ========================================
//
// This file contains functions for querying, inspecting, and validating the
// state of a SafeOutputsConfig. hasAnySafeOutputEnabled and
// hasNonBuiltinSafeOutputsEnabled use direct nil-checks instead of reflection
// for performance (these functions are called on every compilation).
//
// NOTE: When adding a new pointer field to SafeOutputsConfig that represents
// a user-facing safe output action, add it to ALL of the following locations:
//   1. safeOutputFieldMapping (below) — drives imports, prompt generation, etc.
//   2. hasAnySafeOutputEnabled — performance-critical hot path
//   3. hasNonBuiltinSafeOutputsEnabled — if it is NOT a builtin (noop/missing-*)
//   4. hasSafeOutputType in imports.go — used for conflict detection

// safeOutputFieldMapping maps SafeOutputsConfig struct field names to their tool names.
// This map is used by imports, prompt generation, and other metadata operations.
// It is NOT used for existence checks — see hasAnySafeOutputEnabled and
// hasNonBuiltinSafeOutputsEnabled for the performance-critical direct-field versions.
var safeOutputFieldMapping = map[string]string{
	"CreateIssues":                    "create_issue",
	"CreateAgentSessions":             "create_agent_session",
	"CreateDiscussions":               "create_discussion",
	"UpdateDiscussions":               "update_discussion",
	"CloseDiscussions":                "close_discussion",
	"CloseIssues":                     "close_issue",
	"ClosePullRequests":               "close_pull_request",
	"AddComments":                     "add_comment",
	"CreatePullRequests":              "create_pull_request",
	"CreatePullRequestReviewComments": "create_pull_request_review_comment",
	"SubmitPullRequestReview":         "submit_pull_request_review",
	"ReplyToPullRequestReviewComment": "reply_to_pull_request_review_comment",
	"ResolvePullRequestReviewThread":  "resolve_pull_request_review_thread",
	"CreateCodeScanningAlerts":        "create_code_scanning_alert",
	"AutofixCodeScanningAlert":        "autofix_code_scanning_alert",
	"AddLabels":                       "add_labels",
	"RemoveLabels":                    "remove_labels",
	"AddReviewer":                     "add_reviewer",
	"AssignMilestone":                 "assign_milestone",
	"AssignToAgent":                   "assign_to_agent",
	"AssignToUser":                    "assign_to_user",
	"UnassignFromUser":                "unassign_from_user",
	"UpdateIssues":                    "update_issue",
	"UpdatePullRequests":              "update_pull_request",
	"MergePullRequest":                "merge_pull_request",
	"PushToPullRequestBranch":         "push_to_pull_request_branch",
	"UploadAssets":                    "upload_asset",
	"UploadArtifact":                  "upload_artifact",
	"UpdateRelease":                   "update_release",
	"UpdateProjects":                  "update_project",
	"CreateProjects":                  "create_project",
	"CreateProjectStatusUpdates":      "create_project_status_update",
	"LinkSubIssue":                    "link_sub_issue",
	"HideComment":                     "hide_comment",
	"DispatchWorkflow":                "dispatch_workflow",
	"DispatchRepository":              "dispatch_repository",
	"CallWorkflow":                    "call_workflow",
	"MissingTool":                     "missing_tool",
	"MissingData":                     "missing_data",
	"SetIssueType":                    "set_issue_type",
	"SetIssueField":                   "set_issue_field",
	"NoOp":                            "noop",
	"MarkPullRequestAsReadyForReview": "mark_pull_request_as_ready_for_review",
}

// hasAnySafeOutputEnabled reports whether any safe output field is non-nil.
// It uses direct struct-field nil checks instead of reflection for performance;
// this function is called on every compilation and is on the hot path.
//
// NOTE: keep this function in sync with safeOutputFieldMapping above and
// hasNonBuiltinSafeOutputsEnabled below when adding new safe output types.
func hasAnySafeOutputEnabled(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}

	// Check map fields separately
	if len(safeOutputs.Jobs) > 0 || len(safeOutputs.Scripts) > 0 || len(safeOutputs.Actions) > 0 {
		return true
	}

	// Direct nil checks — no reflection, no heap allocation (43 fields matching safeOutputFieldMapping
	// plus CommentMemory which is attached via tools.comment-memory and not in safeOutputFieldMapping).
	return safeOutputs.CreateIssues != nil ||
		safeOutputs.CreateAgentSessions != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.UpdateDiscussions != nil ||
		safeOutputs.CloseDiscussions != nil ||
		safeOutputs.CloseIssues != nil ||
		safeOutputs.ClosePullRequests != nil ||
		safeOutputs.MarkPullRequestAsReadyForReview != nil ||
		safeOutputs.AddComments != nil ||
		safeOutputs.CommentMemory != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.SubmitPullRequestReview != nil ||
		safeOutputs.ReplyToPullRequestReviewComment != nil ||
		safeOutputs.ResolvePullRequestReviewThread != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AutofixCodeScanningAlert != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.RemoveLabels != nil ||
		safeOutputs.AddReviewer != nil ||
		safeOutputs.AssignMilestone != nil ||
		safeOutputs.AssignToAgent != nil ||
		safeOutputs.AssignToUser != nil ||
		safeOutputs.UnassignFromUser != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.UpdatePullRequests != nil ||
		safeOutputs.MergePullRequest != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UploadAssets != nil ||
		safeOutputs.UploadArtifact != nil ||
		safeOutputs.UpdateRelease != nil ||
		safeOutputs.UpdateProjects != nil ||
		safeOutputs.CreateProjects != nil ||
		safeOutputs.CreateProjectStatusUpdates != nil ||
		safeOutputs.LinkSubIssue != nil ||
		safeOutputs.HideComment != nil ||
		safeOutputs.DispatchWorkflow != nil ||
		safeOutputs.DispatchRepository != nil ||
		safeOutputs.CallWorkflow != nil ||
		safeOutputs.MissingTool != nil ||
		safeOutputs.MissingData != nil ||
		safeOutputs.SetIssueType != nil ||
		safeOutputs.SetIssueField != nil ||
		safeOutputs.NoOp != nil // 43rd field
}

// hasNonBuiltinSafeOutputsEnabled reports whether any non-builtin safe output is configured.
// The builtin types (noop, missing-data, missing-tool) are excluded from this check
// because they are always auto-enabled and do not represent a meaningful output action.
//
// NOTE: keep this function in sync with safeOutputFieldMapping above and
// hasAnySafeOutputEnabled above when adding new safe output types.
func hasNonBuiltinSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}

	// Custom safe-jobs, scripts, and actions are always non-builtin
	if len(safeOutputs.Jobs) > 0 || len(safeOutputs.Scripts) > 0 || len(safeOutputs.Actions) > 0 {
		return true
	}

	// Direct nil checks for non-builtin pointer fields (40 fields = 43 total minus 3 builtins:
	// NoOp, MissingData, MissingTool). Includes CommentMemory which is attached via
	// tools.comment-memory and is not in safeOutputFieldMapping.
	return safeOutputs.CreateIssues != nil ||
		safeOutputs.CreateAgentSessions != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.UpdateDiscussions != nil ||
		safeOutputs.CloseDiscussions != nil ||
		safeOutputs.CloseIssues != nil ||
		safeOutputs.ClosePullRequests != nil ||
		safeOutputs.MarkPullRequestAsReadyForReview != nil ||
		safeOutputs.AddComments != nil ||
		safeOutputs.CommentMemory != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.SubmitPullRequestReview != nil ||
		safeOutputs.ReplyToPullRequestReviewComment != nil ||
		safeOutputs.ResolvePullRequestReviewThread != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AutofixCodeScanningAlert != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.RemoveLabels != nil ||
		safeOutputs.AddReviewer != nil ||
		safeOutputs.AssignMilestone != nil ||
		safeOutputs.AssignToAgent != nil ||
		safeOutputs.AssignToUser != nil ||
		safeOutputs.UnassignFromUser != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.UpdatePullRequests != nil ||
		safeOutputs.MergePullRequest != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UploadAssets != nil ||
		safeOutputs.UploadArtifact != nil ||
		safeOutputs.UpdateRelease != nil ||
		safeOutputs.UpdateProjects != nil ||
		safeOutputs.CreateProjects != nil ||
		safeOutputs.CreateProjectStatusUpdates != nil ||
		safeOutputs.LinkSubIssue != nil ||
		safeOutputs.HideComment != nil ||
		safeOutputs.DispatchWorkflow != nil ||
		safeOutputs.DispatchRepository != nil ||
		safeOutputs.CallWorkflow != nil ||
		safeOutputs.SetIssueType != nil ||
		safeOutputs.SetIssueField != nil // non-builtin safe output field
}

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	enabled := hasAnySafeOutputEnabled(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// applyDefaultCreateIssue injects a default create-issues safe output when no non-builtin
// safe outputs are configured (including when safe-outputs is nil / not configured at all).
// The injected config uses the workflow ID as the label and [workflowID] as the title prefix.
// The AutoInjectedCreateIssue flag is set so the prompt generator can add a specific
// instruction for the agent. This aligns create-issue with the other builtin safe outputs
// (noop, missing-tool, missing-data) that are always available.
func applyDefaultCreateIssue(workflowData *WorkflowData) {
	if hasNonBuiltinSafeOutputsEnabled(workflowData.SafeOutputs) {
		return
	}
	if workflowData.SafeOutputs == nil {
		workflowData.SafeOutputs = &SafeOutputsConfig{}
	}

	workflowID := workflowData.WorkflowID
	safeOutputsConfigLog.Printf("Auto-injecting create-issues for workflow %q (no non-builtin safe outputs configured)", workflowID)
	workflowData.SafeOutputs.CreateIssues = &CreateIssuesConfig{
		BaseSafeOutputConfig: BaseSafeOutputConfig{Max: defaultIntStr(1)},
		Labels:               []string{workflowID},
		TitlePrefix:          fmt.Sprintf("[%s]", workflowID),
	}
	workflowData.SafeOutputs.AutoInjectedCreateIssue = true
}
