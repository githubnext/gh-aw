package workflow

func buildIssueAndDiscussionHandlerRegistry() map[string]handlerBuilder { //nolint:largefunc // Declarative handler registry.
	return map[string]handlerBuilder{
		"create_issue": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.CreateIssues == nil {
				return nil
			}
			c := cfg.CreateIssues
			builder := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfTrue("require_temporary_id", c.RequireTemporaryID).
				AddStringSlice("allowed_labels", c.AllowedLabels).
				AddStringSlice("allowed_fields", c.AllowedFields).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddIfPositive("expires", c.Expires).
				AddStringSlice("labels", c.Labels).
				AddIfNotEmpty("title_prefix", c.TitlePrefix).
				AddStringSlice("assignees", c.Assignees).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddTemplatableBool("group", c.Group).
				// Shared CloseOlderConfig.Enabled is remapped here to this handler's
				// entity-specific env key name; the other create-* handlers below map the
				// same shared field to their own entity-specific keys.
				AddTemplatableBool("close_older_issues", c.Enabled).
				AddIfNotEmpty("close_older_key", c.Key).
				AddTemplatableBool("group_by_day", c.GroupByDay).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-issue", c.GitHubToken)).
				AddBoolPtr("normalize_closing_keywords", c.NormalizeClosingKeywords).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				AddTemplatableBoolOrInt("deduplicate_by_title", c.DeduplicateByTitle)
			return builder.Build()
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
				AddStringSlice("hide_older_comments_match", c.HideOlderCommentsMatch).
				AddBoolPtr("discussions", c.Discussions).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddTemplatableStringSlice("allowed_repos", c.AllowedRepos).
				AddTemplatableStringSlice("allows_comment_ids", c.AllowedCommentIDs).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "add-comment", c.GitHubToken)).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddBoolPtr("normalize_closing_keywords", c.NormalizeClosingKeywords).
				AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddIfPositive("min_body_length", c.MinBodyLength).
				AddStringSlice("labels", c.Labels).
				AddStringSlice("allowed_labels", c.AllowedLabels).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				// entity-specific env key name per shared CloseOlderConfig field (see create-issue handler above)
				AddTemplatableBool("close_older_discussions", c.Enabled).
				AddIfNotEmpty("close_older_key", c.Key).
				AddIfNotEmpty("required_category", c.RequiredCategory).
				AddIfPositive("expires", c.Expires).
				AddBoolPtr("fallback_to_issue", c.FallbackToIssue).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-discussion", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddStringSlice("allowed_state_reason", c.AllowedStateReason).
				AddBoolPtr("allow_body", c.AllowBody).
				AddBoolPtr("issue_intent", c.IssueIntent).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "close-issue", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddBoolPtr("allow_body", c.AllowBody).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "close-discussion", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddBoolPtr("issue_intent", c.IssueIntent).
				AddIfNotEmpty("target", c.Target).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "add-labels", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "remove-labels", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"replace_label": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.ReplaceLabel == nil {
				return nil
			}
			c := cfg.ReplaceLabel
			transitions := make([]map[string]string, len(c.AllowedTransitions))
			for i, t := range c.AllowedTransitions {
				transitions[i] = map[string]string{"from": t.From, "to": t.To}
			}
			config := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddStringSlice("allowed_add", c.AllowedAdd).
				AddStringSlice("allowed_remove", c.AllowedRemove).
				AddStringSlice("blocked", c.Blocked).
				AddMapSlice("allowed_transitions", transitions).
				AddIfNotEmpty("target", c.Target).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "replace-label", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
			// If config is empty, it means replace_label was explicitly configured with no options
			// (null config), which means "allow any labels". Return non-nil empty map to
			// indicate the handler is enabled.
			if len(config) == 0 {
				return make(map[string]any)
			}
			return config
		},
		"update_issue": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.UpdateIssues == nil {
				return nil
			}
			c := cfg.UpdateIssues
			builder := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("target", c.Target).
				AddIfNotEmpty("title_prefix", c.TitlePrefix).
				AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix)
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
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "update-issue", c.GitHubToken)).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "update-discussion", c.GitHubToken)).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddTemplatableStringSlice("allowed_repos", c.AllowedRepos).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "link-sub-issue", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
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
				AddIfNotEmpty("target", c.Target).AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "assign-to-user", c.GitHubToken)).
				AddTemplatableBool("unassign_first", c.UnassignFirst).
				AddBoolPtr("issue_intent", c.IssueIntent).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "unassign-from-user", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddBoolPtr("issue_intent", c.IssueIntent).
				AddIfNotEmpty("target", c.Target).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "set-issue-type", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
			// If config is empty, it means set_issue_type was explicitly configured with no options
			// (null config), which means "allow any type". Return non-nil empty map to
			// indicate the handler is enabled.
			if len(config) == 0 {
				return make(map[string]any)
			}
			return config
		},
		"set_issue_field": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.SetIssueField == nil {
				return nil
			}
			c := cfg.SetIssueField
			config := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddStringSlice("allowed_fields", c.AllowedFields).
				AddBoolPtr("issue_intent", c.IssueIntent).
				AddIfNotEmpty("target", c.Target).AddStringSlice("required_labels", c.RequiredLabels).
				AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "set-issue-field", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
			if len(config) == 0 {
				return make(map[string]any)
			}
			return config
		},
	}
}
