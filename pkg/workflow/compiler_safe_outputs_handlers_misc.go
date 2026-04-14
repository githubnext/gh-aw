// This file provides workflow compilation functionality for gh-aw.
// This file (compiler_safe_outputs_handlers_misc.go) registers utility and project safe output handlers
// into the global handlerRegistry during package initialization.

package workflow

import "maps"

func init() {
	maps.Copy(handlerRegistry, map[string]handlerBuilder{
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
	})
}
