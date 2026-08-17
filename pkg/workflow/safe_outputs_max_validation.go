package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var safeOutputsMaxValidationLog = logger.New("workflow:safe_outputs_max_validation")

// isInvalidMaxValue returns true if n is not a valid max field value.
// Valid values are positive integers (n > 0) or -1 (unlimited).
// Invalid values are 0 and negative integers except -1.
func isInvalidMaxValue(n int) bool {
	if n == -1 {
		return false // -1 = unlimited, explicitly allowed by spec
	}
	return n <= 0
}

// maxInvalidErrSuffix is the common suffix of max validation error messages.
const maxInvalidErrSuffix = "\n\nThe max field controls how many times this safe output can be triggered.\nProvide a positive integer (e.g., max: 1 or max: 5) or -1 for unlimited"

// checkMaxField validates a single safe-output max field value.
// Returns an error if the max value is invalid (0 or negative, except -1).
// Returns nil if the max pointer is nil, the value is an expression, or is valid.
func checkMaxField(toolName string, maxPtr *string) error {
	if maxPtr == nil || isExpression(*maxPtr) {
		return nil
	}
	n, err := strconv.Atoi(*maxPtr)
	if err != nil {
		return nil
	}
	if isInvalidMaxValue(n) {
		toolDisplayName := strings.ReplaceAll(toolName, "_", "-")
		safeOutputsMaxValidationLog.Printf("Invalid max value %d for %s", n, toolDisplayName)
		return fmt.Errorf(
			"safe-outputs.%s: max must be a positive integer or -1 (unlimited), got %d%s",
			toolDisplayName, n, maxInvalidErrSuffix,
		)
	}
	return nil
}

type maxFieldCheck struct {
	toolName string
	maxPtr   *string
}

func validateStandardMaxFields(config *SafeOutputsConfig) error {
	checks := buildStandardMaxChecks(config)

	for _, check := range checks {
		if err := checkMaxField(check.toolName, check.maxPtr); err != nil {
			return err
		}
	}

	return nil
}

func buildStandardMaxChecks(config *SafeOutputsConfig) []maxFieldCheck {
	checks := []maxFieldCheck{}
	appendChecksGroup1(config, &checks)
	appendChecksGroup2(config, &checks)
	appendChecksGroup3(config, &checks)
	appendChecksGroup4(config, &checks)
	appendChecksGroup5(config, &checks)
	return checks
}

func appendChecksGroup1(config *SafeOutputsConfig, checks *[]maxFieldCheck) {
	if config.AddComments != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "add_comment", maxPtr: config.AddComments.Max})
	}
	if config.AddLabels != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "add_labels", maxPtr: config.AddLabels.Max})
	}
	if config.AddReviewer != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "add_reviewer", maxPtr: config.AddReviewer.Max})
	}
	if config.AssignMilestone != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "assign_milestone", maxPtr: config.AssignMilestone.Max})
	}
	if config.AssignToAgent != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "assign_to_agent", maxPtr: config.AssignToAgent.Max})
	}
	if config.AssignToUser != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "assign_to_user", maxPtr: config.AssignToUser.Max})
	}
	if config.AutofixCodeScanningAlert != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "autofix_code_scanning_alert", maxPtr: config.AutofixCodeScanningAlert.Max})
	}
	if config.CallWorkflow != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "call_workflow", maxPtr: config.CallWorkflow.Max})
	}
	if config.CloseDiscussions != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "close_discussion", maxPtr: config.CloseDiscussions.Max})
	}
}

func appendChecksGroup2(config *SafeOutputsConfig, checks *[]maxFieldCheck) {
	if config.CloseIssues != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "close_issue", maxPtr: config.CloseIssues.Max})
	}
	if config.ClosePullRequests != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "close_pull_request", maxPtr: config.ClosePullRequests.Max})
	}
	if config.CreateAgentSessions != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_agent_session", maxPtr: config.CreateAgentSessions.Max})
	}
	if config.CreateCheckRun != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_check_run", maxPtr: config.CreateCheckRun.Max})
	}
	if config.CreateCodeScanningAlerts != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_code_scanning_alert", maxPtr: config.CreateCodeScanningAlerts.Max})
	}
	if config.CreateDiscussions != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_discussion", maxPtr: config.CreateDiscussions.Max})
	}
	if config.CreateIssues != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_issue", maxPtr: config.CreateIssues.Max})
	}
	if config.CreateProjectStatusUpdates != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_project_status_update", maxPtr: config.CreateProjectStatusUpdates.Max})
	}
	if config.CreateProjects != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_project", maxPtr: config.CreateProjects.Max})
	}
}

func appendChecksGroup3(config *SafeOutputsConfig, checks *[]maxFieldCheck) {
	if config.CreatePullRequestReviewComments != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_pull_request_review_comment", maxPtr: config.CreatePullRequestReviewComments.Max})
	}
	if config.CreatePullRequests != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "create_pull_request", maxPtr: config.CreatePullRequests.Max})
	}
	if config.DispatchWorkflow != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "dispatch_workflow", maxPtr: config.DispatchWorkflow.Max})
	}
	if config.DismissPullRequestReview != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "dismiss_pull_request_review", maxPtr: config.DismissPullRequestReview.Max})
	}
	if config.HideComment != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "hide_comment", maxPtr: config.HideComment.Max})
	}
	if config.LinkSubIssue != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "link_sub_issue", maxPtr: config.LinkSubIssue.Max})
	}
	if config.MarkPullRequestAsReadyForReview != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "mark_pull_request_as_ready_for_review", maxPtr: config.MarkPullRequestAsReadyForReview.Max})
	}
	if config.ApproveWorkflowRun != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "approve_workflow_run", maxPtr: config.ApproveWorkflowRun.Max})
	}
	if config.MergePullRequest != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "merge_pull_request", maxPtr: config.MergePullRequest.Max})
	}
}

func appendChecksGroup4(config *SafeOutputsConfig, checks *[]maxFieldCheck) {
	if config.MissingData != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "missing_data", maxPtr: config.MissingData.Max})
	}
	if config.MissingTool != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "missing_tool", maxPtr: config.MissingTool.Max})
	}
	if config.NoOp != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "noop", maxPtr: config.NoOp.Max})
	}
	if config.PushToPullRequestBranch != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "push_to_pull_request_branch", maxPtr: config.PushToPullRequestBranch.Max})
	}
	if config.RemoveLabels != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "remove_labels", maxPtr: config.RemoveLabels.Max})
	}
	if config.ReplaceLabel != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "replace_label", maxPtr: config.ReplaceLabel.Max})
	}
	if config.ReplyToPullRequestReviewComment != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "reply_to_pull_request_review_comment", maxPtr: config.ReplyToPullRequestReviewComment.Max})
	}
	if config.ResolvePullRequestReviewThread != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "resolve_pull_request_review_thread", maxPtr: config.ResolvePullRequestReviewThread.Max})
	}
	if config.SetIssueType != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "set_issue_type", maxPtr: config.SetIssueType.Max})
	}
}

func appendChecksGroup5(config *SafeOutputsConfig, checks *[]maxFieldCheck) {
	if config.SetIssueField != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "set_issue_field", maxPtr: config.SetIssueField.Max})
	}
	if config.SubmitPullRequestReview != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "submit_pull_request_review", maxPtr: config.SubmitPullRequestReview.Max})
	}
	if config.UnassignFromUser != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "unassign_from_user", maxPtr: config.UnassignFromUser.Max})
	}
	if config.UpdateDiscussions != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "update_discussion", maxPtr: config.UpdateDiscussions.Max})
	}
	if config.UpdateIssues != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "update_issue", maxPtr: config.UpdateIssues.Max})
	}
	if config.UpdateProjects != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "update_project", maxPtr: config.UpdateProjects.Max})
	}
	if config.UpdatePullRequests != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "update_pull_request", maxPtr: config.UpdatePullRequests.Max})
	}
	if config.UpdateRelease != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "update_release", maxPtr: config.UpdateRelease.Max})
	}
	if config.UploadArtifact != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "upload_artifact", maxPtr: config.UploadArtifact.Max})
	}
	if config.UploadAssets != nil {
		*checks = append(*checks, maxFieldCheck{toolName: "upload_asset", maxPtr: config.UploadAssets.Max})
	}
}

func validateDispatchRepositoryMax(config *SafeOutputsConfig) error {
	if config.DispatchRepository == nil {
		return nil
	}

	sortedToolNames := sliceutil.SortedKeys(config.DispatchRepository.Tools)
	for _, toolName := range sortedToolNames {
		tool := config.DispatchRepository.Tools[toolName]
		if tool == nil || tool.Max == nil || isExpression(*tool.Max) {
			continue
		}

		n, err := strconv.Atoi(*tool.Max)
		if err != nil {
			continue
		}

		if isInvalidMaxValue(n) {
			safeOutputsMaxValidationLog.Printf("Invalid max value %d for dispatch_repository tool %s", n, toolName)
			return fmt.Errorf(
				"safe-outputs.dispatch_repository.%s: max must be a positive integer or -1 (unlimited), got %d%s",
				toolName, n, maxInvalidErrSuffix,
			)
		}
	}

	return nil
}

// validateSafeOutputsMax validates that all max fields in safe-outputs configs hold valid values.
// Valid values are positive integers (n > 0) or -1 (unlimited per spec).
// 0 and other negative values are rejected.
// GitHub Actions expressions (e.g. "${{ inputs.max }}") are not evaluable at compile time
// and are therefore skipped.
//
// This function uses direct struct field access instead of reflection for performance;
// it is on the hot path and called on every compilation. The field ordering matches
// the sorted safeOutputFieldMapping keys for deterministic error reporting.
func validateSafeOutputsMax(config *SafeOutputsConfig) error {
	if config == nil {
		return nil
	}

	safeOutputsMaxValidationLog.Print("Validating safe-outputs max fields")
	if err := validateStandardMaxFields(config); err != nil {
		return err
	}
	if err := validateDispatchRepositoryMax(config); err != nil {
		return err
	}

	safeOutputsMaxValidationLog.Print("Safe-outputs max fields validation passed")
	return nil
}
