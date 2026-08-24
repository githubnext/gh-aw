package workflow

func buildRepositoryAutomationHandlerRegistry() map[string]handlerBuilder { //nolint:largefunc // Declarative handler registry.
	return map[string]handlerBuilder{
		"approve_workflow_run": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.ApproveWorkflowRun == nil {
				return nil
			}
			c := cfg.ApproveWorkflowRun
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddDefault("comment", c.Comment).
				AddStringSlice("allowed_repos", c.AllowedRepos).
				AddTemplatableJSONSlice("allowed_pull_requests", c.AllowedPullRequests).
				AddStringSlice("allowed_workflows", c.AllowedWorkflows).
				AddStringSlice("protected_files", getAllManifestFiles()).
				AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
				AddDefault("protect_top_level_dot_folders", true).
				AddStringSlice("_protected_files_exclude", c.ProtectedFilesExclude).
				AddIfNotEmpty("github-token", resolveApproveWorkflowRunGitHubToken(cfg, c)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
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
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-code-scanning-alert", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"create_check_run": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.CreateCheckRun == nil {
				return nil
			}
			c := cfg.CreateCheckRun
			builder := newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("target", c.Target).
				AddIfNotEmpty("name", c.Name).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged))
			if c.Output != nil {
				builder.
					AddIfNotEmpty("output_title", c.Output.Title).
					AddIfNotEmpty("output_summary", c.Output.Summary)
			}
			// Use resolveHandlerGitHubToken so the per-handler github-app pattern is consistent
			// with all other handlers: when github-app is set the compiler mints a dedicated
			// {key}-app-token step; otherwise fall back to the explicit github-token.
			builder.AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-check-run", c.GitHubToken))
			return builder.Build()
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
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "create-agent-session", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"update_release": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.UpdateRelease == nil {
				return nil
			}
			c := cfg.UpdateRelease
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "update-release", c.GitHubToken)).
				AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
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
				AddBoolPtr("issue_intent", c.IssueIntent).
				AddIfTrue("ignore-if-error", c.IgnoreIfError).
				AddIfNotEmpty("target", c.Target).
				AddIfNotEmpty("target-repo", c.TargetRepoSlug).
				AddStringSlice("allowed-repos", c.AllowedRepos).
				AddIfNotEmpty("pull-request-repo", c.PullRequestRepoSlug).
				AddStringSlice("allowed-pull-request-repos", c.AllowedPullRequestRepos).
				AddIfNotEmpty("base-branch", c.BaseBranch).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "assign-to-agent", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
		"autofix_code_scanning_alert": func(cfg *SafeOutputsConfig) map[string]any {
			if cfg.AutofixCodeScanningAlert == nil {
				return nil
			}
			c := cfg.AutofixCodeScanningAlert
			return newHandlerConfigBuilder().
				AddTemplatableInt("max", c.Max).
				AddIfNotEmpty("github-token", resolveHandlerGitHubToken(c.GitHubApp, "autofix-code-scanning-alert", c.GitHubToken)).
				AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).
				Build()
		},
	}
}
