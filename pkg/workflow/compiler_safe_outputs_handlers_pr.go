// This file provides workflow compilation functionality for gh-aw.
// This file (compiler_safe_outputs_handlers_pr.go) registers pull request safe output handlers
// into the global handlerRegistry during package initialization.

package workflow

import "maps"

func init() {
	maps.Copy(handlerRegistry, map[string]handlerBuilder{
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
				AddStringSlice("reviewers", c.Reviewers).
				AddStringSlice("assignees", c.Assignees).
				AddTemplatableBool("draft", c.Draft).
				AddIfNotEmpty("if_no_changes", c.IfNoChanges).
				AddTemplatableBool("allow_empty", c.AllowEmpty).
				AddTemplatableBool("auto_merge", c.AutoMerge).
				AddIfPositive("expires", c.Expires).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddDefault("max_patch_size", maxPatchSize).
				AddIfNotEmpty("github-token", c.GitHubToken).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddBoolPtr("fallback_as_issue", c.FallbackAsIssue).
				AddTemplatableBool("auto_close_issue", c.AutoCloseIssue).
				AddIfNotEmpty("base_branch", c.BaseBranch).
				AddStringPtr("protected_files_policy", c.ManifestFilesPolicy).
				AddStringSlice("protected_files", getAllManifestFiles()).
				AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
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
				AddIfNotEmpty("commit_title_suffix", c.CommitTitleSuffix).
				AddDefault("max_patch_size", maxPatchSize).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddIfNotEmpty("github-token", c.GitHubToken).
				AddIfTrue("staged", c.Staged).
				AddStringPtr("protected_files_policy", c.ManifestFilesPolicy).
				AddStringSlice("protected_files", getAllManifestFiles()).
				AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
				AddStringSlice("allowed_files", c.AllowedFiles).
				AddStringSlice("excluded_files", c.ExcludedFiles).
				AddIfNotEmpty("patch_format", c.PatchFormat).
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
				AddStringPtr("default_operation", c.Operation).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed_repos", c.AllowedRepos).
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
	})
}
