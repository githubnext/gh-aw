package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputsPermissionsLog = logger.New("workflow:safe_outputs_permissions")

// ComputePermissionsForSafeOutputs computes the minimal required permissions
// based on the configured safe-outputs. This function is used by both the
// consolidated safe outputs job and the conclusion job to ensure they only
// request the permissions they actually need.
//
// This implements the principle of least privilege by only including
// permissions that are required by the configured safe outputs.
func ComputePermissionsForSafeOutputs(safeOutputs *SafeOutputsConfig) *Permissions {
	if safeOutputs == nil {
		safeOutputsPermissionsLog.Print("No safe outputs configured, returning empty permissions")
		return NewPermissions()
	}

	permissions := NewPermissions()

	// Merge permissions for all handler-managed types
	if safeOutputs.CreateIssues != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for create-issue")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.CreateDiscussions != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for create-discussion")
		permissions.Merge(NewPermissionsContentsReadIssuesWriteDiscussionsWrite())
	}
	if safeOutputs.AddComments != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for add-comment")
		// Check if discussions permission should be excluded (discussions: false)
		// Default (nil or true) includes discussions:write for GitHub Apps with Discussions permission
		// Note: PR comments are issue comments, so only issues:write is needed, not pull_requests:write
		if safeOutputs.AddComments.Discussions != nil && !*safeOutputs.AddComments.Discussions {
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		} else {
			permissions.Merge(NewPermissionsContentsReadIssuesWriteDiscussionsWrite())
		}
	}
	if safeOutputs.CloseIssues != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for close-issue")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.CloseDiscussions != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for close-discussion")
		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}
	if safeOutputs.AddLabels != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for add-labels")
		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWrite())
	}
	if safeOutputs.RemoveLabels != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for remove-labels")
		permissions.Merge(NewPermissionsContentsReadIssuesWritePRWrite())
	}
	if safeOutputs.UpdateIssues != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for update-issue")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.UpdateDiscussions != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for update-discussion")
		permissions.Merge(NewPermissionsContentsReadDiscussionsWrite())
	}
	if safeOutputs.LinkSubIssue != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for link-sub-issue")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.UpdateRelease != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for update-release")
		permissions.Merge(NewPermissionsContentsWrite())
	}
	if safeOutputs.CreatePullRequestReviewComments != nil || safeOutputs.SubmitPullRequestReview != nil ||
		safeOutputs.ReplyToPullRequestReviewComment != nil || safeOutputs.ResolvePullRequestReviewThread != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for PR review operations")
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if safeOutputs.CreatePullRequests != nil {
		// Check fallback-as-issue setting to determine permissions
		if getFallbackAsIssue(safeOutputs.CreatePullRequests) {
			safeOutputsPermissionsLog.Print("Adding permissions for create-pull-request with fallback-as-issue")
			permissions.Merge(NewPermissionsContentsWriteIssuesWritePRWrite())
		} else {
			safeOutputsPermissionsLog.Print("Adding permissions for create-pull-request")
			permissions.Merge(NewPermissionsContentsWritePRWrite())
		}
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for push-to-pull-request-branch")
		permissions.Merge(NewPermissionsContentsWritePRWrite())
	}
	if safeOutputs.UpdatePullRequests != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for update-pull-request")
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if safeOutputs.ClosePullRequests != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for close-pull-request")
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if safeOutputs.MarkPullRequestAsReadyForReview != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for mark-pull-request-as-ready-for-review")
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if safeOutputs.HideComment != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for hide-comment")
		// Check if discussions permission should be excluded (discussions: false)
		// Default (nil or true) includes discussions:write for GitHub Apps with Discussions permission
		// Note: Hiding comments (issue/PR/discussion) only needs issues:write, not pull_requests:write
		if safeOutputs.HideComment.Discussions != nil && !*safeOutputs.HideComment.Discussions {
			permissions.Merge(NewPermissionsContentsReadIssuesWrite())
		} else {
			permissions.Merge(NewPermissionsContentsReadIssuesWriteDiscussionsWrite())
		}
	}
	if safeOutputs.DispatchWorkflow != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for dispatch-workflow")
		permissions.Merge(NewPermissionsActionsWrite())
	}
	// Project-related types
	if safeOutputs.CreateProjects != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for create-project")
		permissions.Merge(NewPermissionsContentsReadProjectsWrite())
	}
	if safeOutputs.UpdateProjects != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for update-project")
		permissions.Merge(NewPermissionsContentsReadProjectsWrite())
	}
	if safeOutputs.CreateProjectStatusUpdates != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for create-project-status-update")
		permissions.Merge(NewPermissionsContentsReadProjectsWrite())
	}
	if safeOutputs.AssignToAgent != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for assign-to-agent")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.CreateAgentSessions != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for create-agent-session")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.CreateCodeScanningAlerts != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for create-code-scanning-alert")
		permissions.Merge(NewPermissionsContentsReadSecurityEventsWrite())
	}
	if safeOutputs.AutofixCodeScanningAlert != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for autofix-code-scanning-alert")
		permissions.Merge(NewPermissionsContentsReadSecurityEventsWriteActionsRead())
	}
	if safeOutputs.AssignToUser != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for assign-to-user")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.UnassignFromUser != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for unassign-from-user")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.AssignMilestone != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for assign-milestone")
		permissions.Merge(NewPermissionsContentsReadIssuesWrite())
	}
	if safeOutputs.AddReviewer != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for add-reviewer")
		permissions.Merge(NewPermissionsContentsReadPRWrite())
	}
	if safeOutputs.UploadAssets != nil {
		safeOutputsPermissionsLog.Print("Adding permissions for upload-asset")
		permissions.Merge(NewPermissionsContentsWrite())
	}

	// NoOp and MissingTool don't require write permissions beyond what's already included
	// They only need to comment if add-comment is already configured

	safeOutputsPermissionsLog.Printf("Computed permissions with %d scopes", len(permissions.permissions))
	return permissions
}

// SafeOutputsConfigFromKeys builds a minimal SafeOutputsConfig from a list of safe-output
// key names (e.g. "create-issue", "add-comment"). Only the fields needed for permission
// computation are populated. This is used by external callers (e.g. the interactive wizard)
// that want to call ComputePermissionsForSafeOutputs without constructing a full config.
func SafeOutputsConfigFromKeys(keys []string) *SafeOutputsConfig {
	config := &SafeOutputsConfig{}
	for _, key := range keys {
		switch key {
		case "create-issue":
			config.CreateIssues = &CreateIssuesConfig{}
		case "create-agent-session":
			config.CreateAgentSessions = &CreateAgentSessionConfig{}
		case "create-discussion":
			config.CreateDiscussions = &CreateDiscussionsConfig{}
		case "update-discussion":
			config.UpdateDiscussions = &UpdateDiscussionsConfig{}
		case "close-discussion":
			config.CloseDiscussions = &CloseDiscussionsConfig{}
		case "add-comment":
			config.AddComments = &AddCommentsConfig{}
		case "close-issue":
			config.CloseIssues = &CloseIssuesConfig{}
		case "close-pull-request":
			config.ClosePullRequests = &ClosePullRequestsConfig{}
		case "create-pull-request":
			config.CreatePullRequests = &CreatePullRequestsConfig{}
		case "create-pull-request-review-comment":
			config.CreatePullRequestReviewComments = &CreatePullRequestReviewCommentsConfig{}
		case "submit-pull-request-review":
			config.SubmitPullRequestReview = &SubmitPullRequestReviewConfig{}
		case "reply-to-pull-request-review-comment":
			config.ReplyToPullRequestReviewComment = &ReplyToPullRequestReviewCommentConfig{}
		case "resolve-pull-request-review-thread":
			config.ResolvePullRequestReviewThread = &ResolvePullRequestReviewThreadConfig{}
		case "create-code-scanning-alert":
			config.CreateCodeScanningAlerts = &CreateCodeScanningAlertsConfig{}
		case "autofix-code-scanning-alert":
			config.AutofixCodeScanningAlert = &AutofixCodeScanningAlertConfig{}
		case "add-labels":
			config.AddLabels = &AddLabelsConfig{}
		case "remove-labels":
			config.RemoveLabels = &RemoveLabelsConfig{}
		case "add-reviewer":
			config.AddReviewer = &AddReviewerConfig{}
		case "assign-milestone":
			config.AssignMilestone = &AssignMilestoneConfig{}
		case "assign-to-agent":
			config.AssignToAgent = &AssignToAgentConfig{}
		case "assign-to-user":
			config.AssignToUser = &AssignToUserConfig{}
		case "unassign-from-user":
			config.UnassignFromUser = &UnassignFromUserConfig{}
		case "update-issue":
			config.UpdateIssues = &UpdateIssuesConfig{}
		case "update-pull-request":
			config.UpdatePullRequests = &UpdatePullRequestsConfig{}
		case "push-to-pull-request-branch":
			config.PushToPullRequestBranch = &PushToPullRequestBranchConfig{}
		case "upload-asset":
			config.UploadAssets = &UploadAssetsConfig{}
		case "update-release":
			config.UpdateRelease = &UpdateReleaseConfig{}
		case "hide-comment":
			config.HideComment = &HideCommentConfig{}
		case "link-sub-issue":
			config.LinkSubIssue = &LinkSubIssueConfig{}
		case "update-project":
			config.UpdateProjects = &UpdateProjectConfig{}
		case "create-project":
			config.CreateProjects = &CreateProjectsConfig{}
		case "create-project-status-update":
			config.CreateProjectStatusUpdates = &CreateProjectStatusUpdateConfig{}
		case "mark-pull-request-as-ready-for-review":
			config.MarkPullRequestAsReadyForReview = &MarkPullRequestAsReadyForReviewConfig{}
		}
	}
	return config
}
