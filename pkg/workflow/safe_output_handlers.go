package workflow

import "reflect"

type safeOutputHandlerDescriptor struct {
	Key               string
	Aliases           []string
	StructField       string
	ToolName          string
	Builtin           bool
	NewConfig         func() any
	PermissionBuilder func(*SafeOutputsConfig) *Permissions
}

var safeOutputHandlers = []safeOutputHandlerDescriptor{
	{
		Key:         "create-issue",
		StructField: "CreateIssues",
		ToolName:    "create_issue",
		NewConfig:   func() any { return &CreateIssuesConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateIssues") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "create-agent-session",
		Aliases:     []string{"create-agent-task"},
		StructField: "CreateAgentSessions",
		ToolName:    "create_agent_session",
		NewConfig:   func() any { return &CreateAgentSessionConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateAgentSessions") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "create-discussion",
		StructField: "CreateDiscussions",
		ToolName:    "create_discussion",
		NewConfig:   func() any { return &CreateDiscussionsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateDiscussions") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWriteDiscussionsWrite()
		},
	},
	{
		Key:         "update-discussion",
		StructField: "UpdateDiscussions",
		ToolName:    "update_discussion",
		NewConfig:   func() any { return &UpdateDiscussionsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UpdateDiscussions") {
				return nil
			}
			return NewPermissionsContentsReadDiscussionsWrite()
		},
	},
	{
		Key:         "close-discussion",
		StructField: "CloseDiscussions",
		ToolName:    "close_discussion",
		NewConfig:   func() any { return &CloseDiscussionsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CloseDiscussions") {
				return nil
			}
			return NewPermissionsContentsReadDiscussionsWrite()
		},
	},
	{
		Key:         "close-issue",
		StructField: "CloseIssues",
		ToolName:    "close_issue",
		NewConfig:   func() any { return &CloseIssuesConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CloseIssues") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "close-pull-request",
		StructField: "ClosePullRequests",
		ToolName:    "close_pull_request",
		NewConfig:   func() any { return &ClosePullRequestsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "ClosePullRequests") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "mark-pull-request-as-ready-for-review",
		StructField: "MarkPullRequestAsReadyForReview",
		ToolName:    "mark_pull_request_as_ready_for_review",
		NewConfig:   func() any { return &MarkPullRequestAsReadyForReviewConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "MarkPullRequestAsReadyForReview") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "add-comment",
		StructField: "AddComments",
		ToolName:    "add_comment",
		NewConfig:   func() any { return &AddCommentsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AddComments") {
				return nil
			}
			return buildAddCommentPermissions(safeOutputs.AddComments)
		},
	},
	{
		Key:         "comment-memory",
		StructField: "CommentMemory",
		NewConfig:   func() any { return &CommentMemoryConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CommentMemory") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "create-pull-request",
		StructField: "CreatePullRequests",
		ToolName:    "create_pull_request",
		NewConfig:   func() any { return &CreatePullRequestsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreatePullRequests") {
				return nil
			}
			if getFallbackAsIssue(safeOutputs.CreatePullRequests) {
				permissions := NewPermissionsContentsWriteIssuesWritePRWrite()
				if safeOutputs.CreatePullRequests.AllowWorkflows {
					permissions.Set(PermissionWorkflows, PermissionWrite)
				}
				return permissions
			}
			permissions := NewPermissionsContentsWritePRWrite()
			if safeOutputs.CreatePullRequests.AllowWorkflows {
				permissions.Set(PermissionWorkflows, PermissionWrite)
			}
			return permissions
		},
	},
	{
		Key:         "create-pull-request-review-comment",
		StructField: "CreatePullRequestReviewComments",
		ToolName:    "create_pull_request_review_comment",
		NewConfig:   func() any { return &CreatePullRequestReviewCommentsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreatePullRequestReviewComments") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "submit-pull-request-review",
		StructField: "SubmitPullRequestReview",
		ToolName:    "submit_pull_request_review",
		NewConfig:   func() any { return &SubmitPullRequestReviewConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "SubmitPullRequestReview") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "reply-to-pull-request-review-comment",
		StructField: "ReplyToPullRequestReviewComment",
		ToolName:    "reply_to_pull_request_review_comment",
		NewConfig:   func() any { return &ReplyToPullRequestReviewCommentConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "ReplyToPullRequestReviewComment") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "resolve-pull-request-review-thread",
		StructField: "ResolvePullRequestReviewThread",
		ToolName:    "resolve_pull_request_review_thread",
		NewConfig:   func() any { return &ResolvePullRequestReviewThreadConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "ResolvePullRequestReviewThread") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "create-code-scanning-alert",
		StructField: "CreateCodeScanningAlerts",
		ToolName:    "create_code_scanning_alert",
		NewConfig:   func() any { return &CreateCodeScanningAlertsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateCodeScanningAlerts") {
				return nil
			}
			return NewPermissionsContentsReadSecurityEventsWrite()
		},
	},
	{
		Key:         "autofix-code-scanning-alert",
		StructField: "AutofixCodeScanningAlert",
		ToolName:    "autofix_code_scanning_alert",
		NewConfig:   func() any { return &AutofixCodeScanningAlertConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AutofixCodeScanningAlert") {
				return nil
			}
			return NewPermissionsContentsReadSecurityEventsWriteActionsRead()
		},
	},
	{
		Key:         "create-check-run",
		StructField: "CreateCheckRun",
		ToolName:    "create_check_run",
		NewConfig:   func() any { return &CreateCheckRunConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateCheckRun") {
				return nil
			}
			return NewPermissionsContentsReadChecksWrite()
		},
	},
	{
		Key:         "add-labels",
		StructField: "AddLabels",
		ToolName:    "add_labels",
		NewConfig:   func() any { return &AddLabelsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AddLabels") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWritePRWrite()
		},
	},
	{
		Key:         "remove-labels",
		StructField: "RemoveLabels",
		ToolName:    "remove_labels",
		NewConfig:   func() any { return &RemoveLabelsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "RemoveLabels") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWritePRWrite()
		},
	},
	{
		Key:         "add-reviewer",
		StructField: "AddReviewer",
		ToolName:    "add_reviewer",
		NewConfig:   func() any { return &AddReviewerConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AddReviewer") {
				return nil
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "assign-milestone",
		StructField: "AssignMilestone",
		ToolName:    "assign_milestone",
		NewConfig:   func() any { return &AssignMilestoneConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AssignMilestone") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "assign-to-agent",
		StructField: "AssignToAgent",
		ToolName:    "assign_to_agent",
		NewConfig:   func() any { return &AssignToAgentConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AssignToAgent") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "assign-to-user",
		StructField: "AssignToUser",
		ToolName:    "assign_to_user",
		NewConfig:   func() any { return &AssignToUserConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "AssignToUser") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "unassign-from-user",
		StructField: "UnassignFromUser",
		ToolName:    "unassign_from_user",
		NewConfig:   func() any { return &UnassignFromUserConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UnassignFromUser") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "update-issue",
		StructField: "UpdateIssues",
		ToolName:    "update_issue",
		NewConfig:   func() any { return &UpdateIssuesConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UpdateIssues") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "update-pull-request",
		StructField: "UpdatePullRequests",
		ToolName:    "update_pull_request",
		NewConfig:   func() any { return &UpdatePullRequestsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UpdatePullRequests") {
				return nil
			}
			if safeOutputs.UpdatePullRequests.UpdateBranch != nil && *safeOutputs.UpdatePullRequests.UpdateBranch {
				return NewPermissionsContentsWritePRWrite()
			}
			return NewPermissionsContentsReadPRWrite()
		},
	},
	{
		Key:         "merge-pull-request",
		StructField: "MergePullRequest",
		ToolName:    "merge_pull_request",
		NewConfig:   func() any { return &MergePullRequestConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "MergePullRequest") {
				return nil
			}
			return NewPermissionsContentsWritePRWrite()
		},
	},
	{
		Key:         "push-to-pull-request-branch",
		StructField: "PushToPullRequestBranch",
		ToolName:    "push_to_pull_request_branch",
		NewConfig:   func() any { return &PushToPullRequestBranchConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "PushToPullRequestBranch") {
				return nil
			}
			permissions := NewPermissions()
			if getPushFallbackAsPullRequest(safeOutputs.PushToPullRequestBranch) {
				permissions.Merge(NewPermissionsContentsWritePRWrite())
			} else {
				permissions.Merge(NewPermissionsContentsWrite())
			}
			if safeOutputs.PushToPullRequestBranch.AllowWorkflows {
				permissions.Set(PermissionWorkflows, PermissionWrite)
			}
			if getCheckBranchProtection(safeOutputs.PushToPullRequestBranch) {
				permissions.Set(PermissionAdministration, PermissionRead)
			}
			return permissions
		},
	},
	{
		Key:         "upload-asset",
		StructField: "UploadAssets",
		ToolName:    "upload_asset",
		NewConfig:   func() any { return &UploadAssetsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UploadAssets") {
				return nil
			}
			return NewPermissionsContentsRead()
		},
	},
	{
		Key:         "upload-artifact",
		StructField: "UploadArtifact",
		ToolName:    "upload_artifact",
	},
	{
		Key:         "update-release",
		StructField: "UpdateRelease",
		ToolName:    "update_release",
		NewConfig:   func() any { return &UpdateReleaseConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UpdateRelease") {
				return nil
			}
			return NewPermissionsContentsWrite()
		},
	},
	{
		Key:         "update-project",
		StructField: "UpdateProjects",
		ToolName:    "update_project",
		NewConfig:   func() any { return &UpdateProjectConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "UpdateProjects") {
				return nil
			}
			permissions := NewPermissionsContentsReadProjectsWrite()
			permissions.Set(PermissionIssues, PermissionRead)
			return permissions
		},
	},
	{
		Key:         "create-project",
		StructField: "CreateProjects",
		ToolName:    "create_project",
		NewConfig:   func() any { return &CreateProjectsConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateProjects") {
				return nil
			}
			permissions := NewPermissionsContentsReadProjectsWrite()
			permissions.Set(PermissionIssues, PermissionRead)
			return permissions
		},
	},
	{
		Key:         "create-project-status-update",
		StructField: "CreateProjectStatusUpdates",
		ToolName:    "create_project_status_update",
		NewConfig:   func() any { return &CreateProjectStatusUpdateConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "CreateProjectStatusUpdates") {
				return nil
			}
			return NewPermissionsContentsReadProjectsWrite()
		},
	},
	{
		Key:         "link-sub-issue",
		StructField: "LinkSubIssue",
		ToolName:    "link_sub_issue",
		NewConfig:   func() any { return &LinkSubIssueConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "LinkSubIssue") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "hide-comment",
		StructField: "HideComment",
		ToolName:    "hide_comment",
		NewConfig:   func() any { return &HideCommentConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "HideComment") {
				return nil
			}
			if safeOutputs.HideComment.Discussions != nil && !*safeOutputs.HideComment.Discussions {
				return NewPermissionsContentsReadIssuesWrite()
			}
			return NewPermissionsContentsReadIssuesWriteDiscussionsWrite()
		},
	},
	{
		Key:         "dispatch-workflow",
		StructField: "DispatchWorkflow",
		ToolName:    "dispatch_workflow",
		NewConfig:   func() any { return &DispatchWorkflowConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "DispatchWorkflow") {
				return nil
			}
			return NewPermissionsActionsWrite()
		},
	},
	{
		Key:         "dispatch_repository",
		StructField: "DispatchRepository",
		ToolName:    "dispatch_repository",
		NewConfig:   func() any { return &DispatchRepositoryConfig{} },
	},
	{
		Key:         "call-workflow",
		StructField: "CallWorkflow",
		ToolName:    "call_workflow",
		NewConfig:   func() any { return &CallWorkflowConfig{} },
	},
	{
		Key:         "missing-tool",
		StructField: "MissingTool",
		ToolName:    "missing_tool",
		Builtin:     true,
	},
	{
		Key:         "missing-data",
		StructField: "MissingData",
		ToolName:    "missing_data",
		Builtin:     true,
	},
	{
		Key:         "set-issue-type",
		StructField: "SetIssueType",
		ToolName:    "set_issue_type",
		NewConfig:   func() any { return &SetIssueTypeConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "SetIssueType") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "set-issue-field",
		StructField: "SetIssueField",
		ToolName:    "set_issue_field",
		NewConfig:   func() any { return &SetIssueFieldConfig{} },
		PermissionBuilder: func(safeOutputs *SafeOutputsConfig) *Permissions {
			if !isSafeOutputHandlerEnabledAndUnstaged(safeOutputs, "SetIssueField") {
				return nil
			}
			return NewPermissionsContentsReadIssuesWrite()
		},
	},
	{
		Key:         "noop",
		StructField: "NoOp",
		ToolName:    "noop",
		Builtin:     true,
	},
	{
		Key:         "report-incomplete",
		StructField: "ReportIncomplete",
	},
	{
		Key:         "threat-detection",
		StructField: "ThreatDetection",
	},
}

var safeOutputHandlersByKey = buildSafeOutputHandlersByKey()

func buildSafeOutputHandlersByKey() map[string]safeOutputHandlerDescriptor {
	result := make(map[string]safeOutputHandlerDescriptor, safeAllocationCapacity(len(safeOutputHandlers), 1))
	for _, handler := range safeOutputHandlers {
		if handler.Key != "" {
			result[handler.Key] = handler
		}
		for _, alias := range handler.Aliases {
			result[alias] = handler
		}
	}
	return result
}

func buildSafeOutputFieldMapping() map[string]string {
	mapping := make(map[string]string)
	for _, handler := range safeOutputHandlers {
		if handler.ToolName == "" {
			continue
		}
		mapping[handler.StructField] = handler.ToolName
	}
	return mapping
}

func getSafeOutputHandlerByKey(key string) (safeOutputHandlerDescriptor, bool) {
	handler, ok := safeOutputHandlersByKey[key]
	return handler, ok
}

func safeOutputPointerFieldValue(config *SafeOutputsConfig, fieldName string) (reflect.Value, bool) {
	if config == nil {
		return reflect.Value{}, false
	}

	value := reflect.ValueOf(config)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return reflect.Value{}, false
	}

	field := value.Elem().FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.Pointer {
		return reflect.Value{}, false
	}

	return field, true
}

func hasSafeOutputFieldSet(config *SafeOutputsConfig, fieldName string) bool {
	field, ok := safeOutputPointerFieldValue(config, fieldName)
	return ok && !field.IsNil()
}

func setSafeOutputField(config *SafeOutputsConfig, fieldName string, value any) bool {
	field, ok := safeOutputPointerFieldValue(config, fieldName)
	if !ok {
		return false
	}

	newValue := reflect.ValueOf(value)
	if !newValue.IsValid() || !newValue.Type().AssignableTo(field.Type()) {
		return false
	}

	field.Set(newValue)
	return true
}

func mergeSafeOutputFieldIfNil(result, imported *SafeOutputsConfig, fieldName string) {
	resultField, ok := safeOutputPointerFieldValue(result, fieldName)
	if !ok || !resultField.IsNil() {
		return
	}

	importedField, ok := safeOutputPointerFieldValue(imported, fieldName)
	if !ok || importedField.IsNil() {
		return
	}

	resultField.Set(importedField)
}

func isSafeOutputHandlerEnabledAndUnstaged(safeOutputs *SafeOutputsConfig, fieldName string) bool {
	field, ok := safeOutputPointerFieldValue(safeOutputs, fieldName)
	if !ok || field.IsNil() {
		return false
	}

	return !isHandlerStaged(safeOutputs.Staged, safeOutputHandlerStaged(field))
}

func safeOutputHandlerStaged(field reflect.Value) bool {
	if !field.IsValid() || field.IsNil() {
		return false
	}

	elem := field.Elem()
	if !elem.IsValid() || elem.Kind() != reflect.Struct {
		return false
	}

	if staged := elem.FieldByName("Staged"); staged.IsValid() && staged.Kind() == reflect.Bool {
		return staged.Bool()
	}

	baseConfig := elem.FieldByName("BaseSafeOutputConfig")
	if !baseConfig.IsValid() || baseConfig.Kind() != reflect.Struct {
		return false
	}

	staged := baseConfig.FieldByName("Staged")
	if !staged.IsValid() || staged.Kind() != reflect.Bool {
		return false
	}

	return staged.Bool()
}
