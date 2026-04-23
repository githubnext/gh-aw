package workflow

// handlerRegistry maps handler names to their builder functions.
// Each entry is keyed by the handler name used in GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
// and returns a config map (nil means the handler is disabled).
var handlerRegistry = map[string]handlerBuilder{
	"create_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateIssues == nil {
			return nil
		}
		c := cfg.CreateIssues
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfPositive("expires", c.Expires).
			AddStringSlice("labels", c.Labels).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddStringSlice("assignees", c.Assignees).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableBool("group", c.Group).
			AddTemplatableBool("close_older_issues", c.CloseOlderIssues).
			AddIfNotEmpty("close_older_key", c.CloseOlderKey).
			AddTemplatableBool("group_by_day", c.GroupByDay).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"add_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AddComments == nil {
			return nil
		}
		c := cfg.AddComments
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddTemplatableBool("hide_older_comments", c.HideOlderComments).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"comment_memory": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CommentMemory == nil {
			return nil
		}
		c := cfg.CommentMemory
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("memory_id", c.MemoryID).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_discussion": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateDiscussions == nil {
			return nil
		}
		c := cfg.CreateDiscussions
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("category", c.Category).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddStringSlice("labels", c.Labels).
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddTemplatableBool("close_older_discussions", c.CloseOlderDiscussions).
			AddIfNotEmpty("close_older_key", c.CloseOlderKey).
			AddIfNotEmpty("required_category", c.RequiredCategory).
			AddIfPositive("expires", c.Expires).
			AddBoolPtr("fallback_to_issue", c.FallbackToIssue).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"close_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CloseIssues == nil {
			return nil
		}
		c := cfg.CloseIssues
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("state_reason", c.StateReason).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"close_discussion": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CloseDiscussions == nil {
			return nil
		}
		c := cfg.CloseDiscussions
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"add_labels": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AddLabels == nil {
			return nil
		}
		c := cfg.AddLabels
		config := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
		// If config is empty, it means add_labels was explicitly configured with no options
		// (null config), which means "allow any labels". Return non-nil empty map to
		// indicate the handler is enabled.
		if len(config) == 0 {
			// Return empty map so handler is included in config
			return make(map[string]any)
		}
		return config
	},
	"remove_labels": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.RemoveLabels == nil {
			return nil
		}
		c := cfg.RemoveLabels
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"add_reviewer": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AddReviewer == nil {
			return nil
		}
		c := cfg.AddReviewer
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Reviewers).
			AddStringSlice("allowed_team_reviewers", c.TeamReviewers).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"assign_milestone": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignMilestone == nil {
			return nil
		}
		c := cfg.AssignMilestone
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			AddIfTrue("auto_create", c.AutoCreate).
			Build()
	},
	"mark_pull_request_as_ready_for_review": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MarkPullRequestAsReadyForReview == nil {
			return nil
		}
		c := cfg.MarkPullRequestAsReadyForReview
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_code_scanning_alert": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateCodeScanningAlerts == nil {
			return nil
		}
		c := cfg.CreateCodeScanningAlerts
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("driver", c.Driver).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_agent_session": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateAgentSessions == nil {
			return nil
		}
		c := cfg.CreateAgentSessions
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("base", c.Base).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"update_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateIssues == nil {
			return nil
		}
		c := cfg.UpdateIssues
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("title_prefix", c.TitlePrefix)
		// Boolean pointer fields indicate which fields can be updated
		if c.Status != nil {
			builder.AddDefault("allow_status", true)
		}
		if c.Title != nil {
			builder.AddDefault("allow_title", true)
		}
		// Body uses boolean value mode - add the actual boolean value
		builder.AddBoolPtrOrDefault("allow_body", c.Body, true)
		return builder.
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"update_discussion": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateDiscussions == nil {
			return nil
		}
		c := cfg.UpdateDiscussions
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target)
		// Boolean pointer fields indicate which fields can be updated
		if c.Title != nil {
			builder.AddDefault("allow_title", true)
		}
		if c.Body != nil {
			builder.AddDefault("allow_body", true)
		}
		if c.Labels != nil {
			builder.AddDefault("allow_labels", true)
		}
		return builder.
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"link_sub_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.LinkSubIssue == nil {
			return nil
		}
		c := cfg.LinkSubIssue
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("parent_required_labels", c.ParentRequiredLabels).
			AddIfNotEmpty("parent_title_prefix", c.ParentTitlePrefix).
			AddStringSlice("sub_required_labels", c.SubRequiredLabels).
			AddIfNotEmpty("sub_title_prefix", c.SubTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"update_release": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateRelease == nil {
			return nil
		}
		c := cfg.UpdateRelease
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_pull_request_review_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreatePullRequestReviewComments == nil {
			return nil
		}
		c := cfg.CreatePullRequestReviewComments
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("side", c.Side).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"submit_pull_request_review": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.SubmitPullRequestReview == nil {
			return nil
		}
		c := cfg.SubmitPullRequestReview
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddStringSlice("allowed_events", c.AllowedEvents).
			AddIfTrue("supersede_older_reviews", c.SupersedeOlderReviews).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddStringPtr("footer", getEffectiveFooterString(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"reply_to_pull_request_review_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReplyToPullRequestReviewComment == nil {
			return nil
		}
		c := cfg.ReplyToPullRequestReviewComment
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"resolve_pull_request_review_thread": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ResolvePullRequestReviewThread == nil {
			return nil
		}
		c := cfg.ResolvePullRequestReviewThread
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreatePullRequests == nil {
			return nil
		}
		c := cfg.CreatePullRequests
		maxPatchSize := 1024 // default 1024 KB
		if cfg.MaximumPatchSize > 0 {
			maxPatchSize = cfg.MaximumPatchSize
		}
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddStringSlice("labels", c.Labels).
			AddStringSlice("fallback_labels", c.FallbackLabels).
			AddStringSlice("reviewers", c.Reviewers).
			AddStringSlice("team_reviewers", c.TeamReviewers).
			AddStringSlice("assignees", c.Assignees).
			AddTemplatableBool("draft", c.Draft).
			AddIfNotEmpty("if_no_changes", c.IfNoChanges).
			AddTemplatableBool("allow_empty", c.AllowEmpty).
			AddTemplatableBool("auto_merge", c.AutoMerge).
			AddIfPositive("expires", c.Expires).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddStringSlice("allowed_base_branches", c.AllowedBaseBranches).
			AddDefault("max_patch_size", maxPatchSize).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddBoolPtr("fallback_as_issue", c.FallbackAsIssue).
			AddTemplatableBool("auto_close_issue", c.AutoCloseIssue).
			AddIfNotEmpty("base_branch", c.BaseBranch).
			AddStringPtr("protected_files_policy", c.ManifestFilesPolicy).
			AddStringSlice("protected_files", getAllManifestFiles()).
			AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
			AddStringSlice("_protected_files_exclude", c.ProtectedFilesExclude).
			AddStringSlice("allowed_files", c.AllowedFiles).
			AddStringSlice("excluded_files", c.ExcludedFiles).
			AddIfTrue("preserve_branch_name", c.PreserveBranchName).
			AddIfNotEmpty("patch_format", c.PatchFormat).
			AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"push_to_pull_request_branch": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.PushToPullRequestBranch == nil {
			return nil
		}
		c := cfg.PushToPullRequestBranch
		maxPatchSize := 1024 // default 1024 KB
		if cfg.MaximumPatchSize > 0 {
			maxPatchSize = cfg.MaximumPatchSize
		}
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddStringSlice("labels", c.Labels).
			AddIfNotEmpty("if_no_changes", c.IfNoChanges).
			AddIfTrue("ignore_missing_branch_failure", c.IgnoreMissingBranchFailure).
			AddIfNotEmpty("commit_title_suffix", c.CommitTitleSuffix).
			AddDefault("max_patch_size", maxPatchSize).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			AddStringPtr("protected_files_policy", c.ManifestFilesPolicy).
			AddStringSlice("protected_files", getAllManifestFiles()).
			AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
			AddStringSlice("_protected_files_exclude", c.ProtectedFilesExclude).
			AddStringSlice("allowed_files", c.AllowedFiles).
			AddStringSlice("excluded_files", c.ExcludedFiles).
			AddIfNotEmpty("patch_format", c.PatchFormat).
			AddBoolPtr("fallback_as_pull_request", c.FallbackAsPullRequest).
			Build()
	},
	"update_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdatePullRequests == nil {
			return nil
		}
		c := cfg.UpdatePullRequests
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddBoolPtrOrDefault("allow_title", c.Title, true).
			AddBoolPtrOrDefault("allow_body", c.Body, true).
			AddBoolPtrOrDefault("update_branch", c.UpdateBranch, false).
			AddStringPtr("default_operation", c.Operation).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"merge_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MergePullRequest == nil {
			return nil
		}
		c := cfg.MergePullRequest
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddStringSlice("allowed_branches", c.AllowedBranches).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"close_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ClosePullRequests == nil {
			return nil
		}
		c := cfg.ClosePullRequests
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"hide_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.HideComment == nil {
			return nil
		}
		c := cfg.HideComment
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_reasons", c.AllowedReasons).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"dispatch_workflow": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.DispatchWorkflow == nil {
			return nil
		}
		c := cfg.DispatchWorkflow
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("workflows", c.Workflows).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug)

		// Add workflow_files map if it has entries
		if len(c.WorkflowFiles) > 0 {
			builder.AddDefault("workflow_files", c.WorkflowFiles)
		}

		// Add aw_context_workflows list if it has entries
		if len(c.AwContextWorkflows) > 0 {
			builder.AddStringSlice("aw_context_workflows", c.AwContextWorkflows)
		}

		builder.AddIfNotEmpty("target-ref", c.TargetRef)
		builder.AddIfNotEmpty("github-token", c.GitHubToken)
		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"dispatch_repository": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.DispatchRepository == nil || len(cfg.DispatchRepository.Tools) == 0 {
			return nil
		}
		// Serialize each tool as a sub-map
		tools := make(map[string]any, len(cfg.DispatchRepository.Tools))
		for toolKey, tool := range cfg.DispatchRepository.Tools {
			toolConfig := newHandlerConfigBuilder().
				AddIfNotEmpty("workflow", tool.Workflow).
				AddIfNotEmpty("event_type", tool.EventType).
				AddIfNotEmpty("repository", tool.Repository).
				AddStringSlice("allowed_repositories", tool.AllowedRepositories).
				AddTemplatableInt("max", tool.Max).
				AddIfNotEmpty("github-token", tool.GitHubToken).
				AddIfTrue("staged", tool.Staged).
				Build()
			tools[toolKey] = toolConfig
		}
		return map[string]any{"tools": tools}
	},
	"call_workflow": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CallWorkflow == nil {
			return nil
		}
		c := cfg.CallWorkflow
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("workflows", c.Workflows)

		// Add workflow_files map if it has entries
		if len(c.WorkflowFiles) > 0 {
			builder.AddDefault("workflow_files", c.WorkflowFiles)
		}

		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"missing_tool": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MissingTool == nil {
			return nil
		}
		c := cfg.MissingTool
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"missing_data": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MissingData == nil {
			return nil
		}
		c := cfg.MissingData
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"noop": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.NoOp == nil {
			return nil
		}
		c := cfg.NoOp
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringPtr("report-as-issue", c.ReportAsIssue).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"report_incomplete": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReportIncomplete == nil {
			return nil
		}
		c := cfg.ReportIncomplete
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_report_incomplete_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReportIncomplete == nil {
			return nil
		}
		c := cfg.ReportIncomplete
		if !c.CreateIssue {
			return nil
		}
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("title-prefix", c.TitlePrefix).
			AddStringSlice("labels", c.Labels).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"assign_to_agent": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignToAgent == nil {
			return nil
		}
		c := cfg.AssignToAgent
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("name", c.DefaultAgent).
			AddIfNotEmpty("model", c.DefaultModel).
			AddIfNotEmpty("custom-agent", c.DefaultCustomAgent).
			AddIfNotEmpty("custom-instructions", c.DefaultCustomInstructions).
			AddStringSlice("allowed", c.Allowed).
			AddIfTrue("ignore-if-error", c.IgnoreIfError).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed-repos", c.AllowedRepos).
			AddIfNotEmpty("pull-request-repo", c.PullRequestRepoSlug).
			AddStringSlice("allowed-pull-request-repos", c.AllowedPullRequestRepos).
			AddIfNotEmpty("base-branch", c.BaseBranch).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"upload_asset": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UploadAssets == nil {
			return nil
		}
		c := cfg.UploadAssets
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("branch", c.BranchName).
			AddIfPositive("max-size", c.MaxSizeKB).
			AddStringSlice("allowed-exts", c.AllowedExts).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"upload_artifact": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UploadArtifact == nil {
			return nil
		}
		c := cfg.UploadArtifact
		b := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfPositive("max-uploads", c.MaxUploads).
			AddTemplatableInt("retention-days", c.RetentionDays).
			AddTemplatableBool("skip-archive", c.SkipArchive).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged)
		if c.MaxSizeBytes > 0 {
			b = b.AddDefault("max-size-bytes", c.MaxSizeBytes)
		}
		if len(c.AllowedPaths) > 0 {
			b = b.AddStringSlice("allowed-paths", c.AllowedPaths)
		}
		if c.Defaults != nil {
			if c.Defaults.IfNoFiles != "" {
				b = b.AddIfNotEmpty("default-if-no-files", c.Defaults.IfNoFiles)
			}
		}
		if c.Filters != nil {
			if len(c.Filters.Include) > 0 {
				b = b.AddStringSlice("filters-include", c.Filters.Include)
			}
			if len(c.Filters.Exclude) > 0 {
				b = b.AddStringSlice("filters-exclude", c.Filters.Exclude)
			}
		}
		return b.Build()
	},
	"autofix_code_scanning_alert": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AutofixCodeScanningAlert == nil {
			return nil
		}
		c := cfg.AutofixCodeScanningAlert
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	// Note: create_project, update_project and create_project_status_update are handled by the unified handler,
	// not the separate project handler manager, so they are included in this registry.
	"create_project": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateProjects == nil {
			return nil
		}
		c := cfg.CreateProjects
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target_owner", c.TargetOwner).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddIfNotEmpty("github-token", c.GitHubToken)
		if len(c.Views) > 0 {
			builder.AddDefault("views", c.Views)
		}
		if len(c.FieldDefinitions) > 0 {
			builder.AddDefault("field_definitions", c.FieldDefinitions)
		}
		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"update_project": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateProjects == nil {
			return nil
		}
		c := cfg.UpdateProjects
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfNotEmpty("project", c.Project).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos)
		if len(c.Views) > 0 {
			builder.AddDefault("views", c.Views)
		}
		if len(c.FieldDefinitions) > 0 {
			builder.AddDefault("field_definitions", c.FieldDefinitions)
		}
		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"assign_to_user": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignToUser == nil {
			return nil
		}
		c := cfg.AssignToUser
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("unassign_first", c.UnassignFirst).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"unassign_from_user": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UnassignFromUser == nil {
			return nil
		}
		c := cfg.UnassignFromUser
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_project_status_update": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateProjectStatusUpdates == nil {
			return nil
		}
		c := cfg.CreateProjectStatusUpdates
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfNotEmpty("project", c.Project).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"set_issue_type": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.SetIssueType == nil {
			return nil
		}
		c := cfg.SetIssueType
		config := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
		// If config is empty, it means set_issue_type was explicitly configured with no options
		// (null config), which means "allow any type". Return non-nil empty map to
		// indicate the handler is enabled.
		if len(config) == 0 {
			return make(map[string]any)
		}
		return config
	},
}
