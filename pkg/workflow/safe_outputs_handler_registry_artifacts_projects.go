package workflow

func buildArtifactAndProjectHandlerRegistry() map[string]handlerBuilder { //nolint:largefunc // Declarative handler registry.
	return map[string]handlerBuilder{
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
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "upload-asset", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "upload-artifact", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
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
		"upload_code_coverage": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.UploadCodeCoverage == nil {
				return nil
			}
			c := cfg.UploadCodeCoverage
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "upload-code-coverage", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
}
