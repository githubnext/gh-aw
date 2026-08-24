package workflow

func buildWorkflowDispatchAndReportingHandlerRegistry() map[string]handlerBuilder { //nolint:largefunc // Declarative handler registry.
	return map[string]handlerBuilder{
		"dispatch_workflow": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.DispatchWorkflow == nil {
				return nil
			}
			c := cfg.DispatchWorkflow
			builder := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddStringSlice("workflows", c.Workflows).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddTemplatableStringSlice("allowed_repos", c.AllowedRepos).
				AddTemplatableStringSlice("allowed_refs", c.AllowedRefs)

			// Add workflow_files map if it has entries
			if len(c.WorkflowFiles) > 0 {
				builder.AddDefault("workflow_files", c.WorkflowFiles)
			}

			// Add aw_context_workflows list if it has entries
			if len(c.AwContextWorkflows) > 0 {
				builder.AddStringSlice("aw_context_workflows", c.AwContextWorkflows)
			}

			builder.AddIfNotEmpty("target-ref", c.TargetRef)
			builder.AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "dispatch-workflow", c.GitHubToken))
			builder.AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
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
					AddIfNotEmpty("github-token", resolveHandlerGitHubTokenWithStepID(tool.GitHubApp, dispatchRepositoryToolAppTokenStepID(toolKey), tool.GitHubToken)).
					AddTemplatableBool("staged", templatableBoolPtrToStringPtr(tool.Staged)).
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

			builder.AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
			return builder.Build()
		},
		"missing_tool": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.MissingTool == nil {
				return nil
			}
			c := cfg.MissingTool
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "missing-tool", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"missing_data": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.MissingData == nil {
				return nil
			}
			c := cfg.MissingData
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "missing-data", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"report_incomplete": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.ReportIncomplete == nil {
				return nil
			}
			c := cfg.ReportIncomplete
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "report-incomplete", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"create_report_incomplete_issue": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.ReportIncomplete == nil {
				return nil
			}
			c := cfg.ReportIncomplete
			// If create-issue is explicitly false, skip generating the issue handler.
			// For nil (default) or "true", always include; for expressions, include
			// the handler and embed the expression so it is evaluated at runtime.
			if c.CreateIssue != nil && *c.CreateIssue == "false" {
				return nil
			}
			builder := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("title-prefix", c.TitlePrefix).
				AddStringSlice("labels", c.Labels).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "report-incomplete", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
			// When create-issue is a GitHub Actions expression, embed it in the handler config.
			// GitHub Actions evaluates the expression before the handler runs; the JavaScript
			// handler then parses the resolved value via parseBoolTemplatable at runtime.
			if c.CreateIssue != nil && isExpression(*c.CreateIssue) {
				builder = builder.AddTemplatableBool("create-issue", c.CreateIssue)
			}
			return builder.Build()
		},
	}
}
