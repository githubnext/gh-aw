package workflow

// commentHandlerRegistry contains comment handler builders.
var commentHandlerRegistry = map[string]handlerBuilder{
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
	"hide_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.HideComment == nil {
			return nil
		}
		c := cfg.HideComment
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_reasons", c.AllowedReasons).AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "hide-comment", c.GitHubToken)).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
			Build()
	},
}
