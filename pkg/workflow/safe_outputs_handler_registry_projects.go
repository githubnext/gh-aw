package workflow

// projectHandlerRegistry contains project board handler builders.
var projectHandlerRegistry = map[string]handlerBuilder{
	"create_project": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateProjects == nil {
			return nil
		}
		c := cfg.CreateProjects
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target_owner", c.TargetOwner).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-project", c.GitHubToken))
		if len(c.Views) > 0 {
			builder.AddDefault("views", c.Views)
		}
		if len(c.FieldDefinitions) > 0 {
			builder.AddDefault("field_definitions", c.FieldDefinitions)
		}
		builder.AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
		return builder.Build()
	},
	"update_project": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateProjects == nil {
			return nil
		}
		c := cfg.UpdateProjects
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "update-project", c.GitHubToken)).
			AddIfNotEmpty("project", c.Project).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos)
		if len(c.Views) > 0 {
			builder.AddDefault("views", c.Views)
		}
		if len(c.FieldDefinitions) > 0 {
			builder.AddDefault("field_definitions", c.FieldDefinitions)
		}
		builder.AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
		return builder.Build()
	},
	"create_project_status_update": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateProjectStatusUpdates == nil {
			return nil
		}
		c := cfg.CreateProjectStatusUpdates
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-project-status-update", c.GitHubToken)).
			AddIfNotEmpty("project", c.Project).
			AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
			Build()
	},
}
