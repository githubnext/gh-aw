package workflow

// discussionHandlerRegistry contains discussion lifecycle handler builders.
var discussionHandlerRegistry = map[string]handlerBuilder{
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
			// entity-specific env key name per shared CloseOlderConfig field (see the create_issue handler in safe_outputs_handler_registry_issues.go)
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
}
