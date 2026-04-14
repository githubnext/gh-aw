// This file provides workflow compilation functionality for gh-aw.
// This file (compiler_safe_outputs_handlers_issue.go) registers issue, comment, discussion, and label safe output handlers
// into the global handlerRegistry during package initialization.

package workflow

import "maps"

func init() {
	maps.Copy(handlerRegistry, map[string]handlerBuilder{
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
	})
}
