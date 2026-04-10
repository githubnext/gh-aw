package workflow

func init() {
	handlerRegistry["dispatch_workflow"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["dispatch_repository"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["call_workflow"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["noop"] = func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.NoOp == nil {
			return nil
		}
		c := cfg.NoOp
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringPtr("report-as-issue", c.ReportAsIssue).
			AddIfTrue("staged", c.Staged).
			Build()
	}

	handlerRegistry["missing_tool"] = func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MissingTool == nil {
			return nil
		}
		c := cfg.MissingTool
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	}

	handlerRegistry["missing_data"] = func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MissingData == nil {
			return nil
		}
		c := cfg.MissingData
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	}

	handlerRegistry["report_incomplete"] = func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReportIncomplete == nil {
			return nil
		}
		c := cfg.ReportIncomplete
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	}

	handlerRegistry["create_report_incomplete_issue"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["assign_to_agent"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["upload_asset"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["upload_artifact"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["autofix_code_scanning_alert"] = func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AutofixCodeScanningAlert == nil {
			return nil
		}
		c := cfg.AutofixCodeScanningAlert
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	}

	// Note: create_project, update_project and create_project_status_update are handled by the unified handler,
	// not the separate project handler manager, so they are included in this registry.
	handlerRegistry["create_project"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["update_project"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["create_project_status_update"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["update_release"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["create_code_scanning_alert"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}

	handlerRegistry["create_agent_session"] = func(cfg *SafeOutputsConfig) map[string]any {
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
	}
}
