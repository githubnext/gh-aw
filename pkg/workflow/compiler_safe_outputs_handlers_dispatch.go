// This file provides workflow compilation functionality for gh-aw.
// This file (compiler_safe_outputs_handlers_dispatch.go) registers dispatch and workflow safe output handlers
// into the global handlerRegistry during package initialization.

package workflow

import "maps"

func init() {
	maps.Copy(handlerRegistry, map[string]handlerBuilder{
		"dispatch_workflow": func(cfg *SafeOutputsConfig) map[string]any {
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
					AddIfNotEmpty("github-token", tool.GitHubToken).
					AddIfTrue("staged", tool.Staged).
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

			builder.AddIfTrue("staged", c.Staged)
			return builder.Build()
		},
	})
}
