package workflow

import (
	"encoding/json"
	"fmt"
)

func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	config := make(map[string]map[string]any)

	// Add config for each enabled safe output type with their options
	// Presence in config = enabled, so no need for "enabled": true field
	if data.SafeOutputs.CreateIssues != nil {
		cfg := data.SafeOutputs.CreateIssues
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.AllowedLabels) > 0 {
			handlerConfig["allowed_labels"] = cfg.AllowedLabels
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		if cfg.Expires > 0 {
			handlerConfig["expires"] = cfg.Expires
		}
		// Add labels, title_prefix to config
		if len(cfg.Labels) > 0 {
			handlerConfig["labels"] = cfg.Labels
		}
		if cfg.TitlePrefix != "" {
			handlerConfig["title_prefix"] = cfg.TitlePrefix
		}
		// Add assignees to config
		if len(cfg.Assignees) > 0 {
			handlerConfig["assignees"] = cfg.Assignees
		}
		// Add target-repo to config
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		// Add group flag to config
		if cfg.Group {
			handlerConfig["group"] = true
		}
		config["create_issue"] = handlerConfig
	}

	if data.SafeOutputs.AddComments != nil {
		cfg := data.SafeOutputs.AddComments
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if cfg.HideOlderComments {
			handlerConfig["hide_older_comments"] = true
		}
		// Add target-repo to config
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["add_comment"] = handlerConfig
	}

	if data.SafeOutputs.CreateDiscussions != nil {
		cfg := data.SafeOutputs.CreateDiscussions
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Category != "" {
			handlerConfig["category"] = cfg.Category
		}
		if cfg.TitlePrefix != "" {
			handlerConfig["title_prefix"] = cfg.TitlePrefix
		}
		if len(cfg.Labels) > 0 {
			handlerConfig["labels"] = cfg.Labels
		}
		if len(cfg.AllowedLabels) > 0 {
			handlerConfig["allowed_labels"] = cfg.AllowedLabels
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		if cfg.CloseOlderDiscussions {
			handlerConfig["close_older_discussions"] = true
		}
		if cfg.RequiredCategory != "" {
			handlerConfig["required_category"] = cfg.RequiredCategory
		}
		if cfg.Expires > 0 {
			handlerConfig["expires"] = cfg.Expires
		}
		// Add target-repo to config
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		config["create_discussion"] = handlerConfig
	}

	if data.SafeOutputs.CloseIssues != nil {
		cfg := data.SafeOutputs.CloseIssues
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if len(cfg.RequiredLabels) > 0 {
			handlerConfig["required_labels"] = cfg.RequiredLabels
		}
		if cfg.RequiredTitlePrefix != "" {
			handlerConfig["required_title_prefix"] = cfg.RequiredTitlePrefix
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["close_issue"] = handlerConfig
	}

	if data.SafeOutputs.CloseDiscussions != nil {
		cfg := data.SafeOutputs.CloseDiscussions
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if len(cfg.RequiredLabels) > 0 {
			handlerConfig["required_labels"] = cfg.RequiredLabels
		}
		if cfg.RequiredTitlePrefix != "" {
			handlerConfig["required_title_prefix"] = cfg.RequiredTitlePrefix
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["close_discussion"] = handlerConfig
	}

	if data.SafeOutputs.AddLabels != nil {
		cfg := data.SafeOutputs.AddLabels
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.Allowed) > 0 {
			handlerConfig["allowed"] = cfg.Allowed
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["add_labels"] = handlerConfig
	}

	if data.SafeOutputs.UpdateIssues != nil {
		cfg := data.SafeOutputs.UpdateIssues
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		// Boolean pointer fields indicate which fields can be updated
		if cfg.Status != nil {
			handlerConfig["allow_status"] = true
		}
		if cfg.Title != nil {
			handlerConfig["allow_title"] = true
		}
		if cfg.Body != nil {
			handlerConfig["allow_body"] = true
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["update_issue"] = handlerConfig
	}

	if data.SafeOutputs.UpdateDiscussions != nil {
		cfg := data.SafeOutputs.UpdateDiscussions
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		// Boolean pointer fields indicate which fields can be updated
		if cfg.Title != nil {
			handlerConfig["allow_title"] = true
		}
		if cfg.Body != nil {
			handlerConfig["allow_body"] = true
		}
		if cfg.Labels != nil {
			handlerConfig["allow_labels"] = true
		}
		if len(cfg.AllowedLabels) > 0 {
			handlerConfig["allowed_labels"] = cfg.AllowedLabels
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["update_discussion"] = handlerConfig
	}

	if data.SafeOutputs.LinkSubIssue != nil {
		cfg := data.SafeOutputs.LinkSubIssue
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.ParentRequiredLabels) > 0 {
			handlerConfig["parent_required_labels"] = cfg.ParentRequiredLabels
		}
		if cfg.ParentTitlePrefix != "" {
			handlerConfig["parent_title_prefix"] = cfg.ParentTitlePrefix
		}
		if len(cfg.SubRequiredLabels) > 0 {
			handlerConfig["sub_required_labels"] = cfg.SubRequiredLabels
		}
		if cfg.SubTitlePrefix != "" {
			handlerConfig["sub_title_prefix"] = cfg.SubTitlePrefix
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["link_sub_issue"] = handlerConfig
	}

	if data.SafeOutputs.UpdateRelease != nil {
		cfg := data.SafeOutputs.UpdateRelease
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		config["update_release"] = handlerConfig
	}

	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		cfg := data.SafeOutputs.CreatePullRequestReviewComments
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Side != "" {
			handlerConfig["side"] = cfg.Side
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["create_pull_request_review_comment"] = handlerConfig
	}

	if data.SafeOutputs.CreatePullRequests != nil {
		cfg := data.SafeOutputs.CreatePullRequests
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.TitlePrefix != "" {
			handlerConfig["title_prefix"] = cfg.TitlePrefix
		}
		if len(cfg.Labels) > 0 {
			handlerConfig["labels"] = cfg.Labels
		}
		if cfg.Draft != nil {
			handlerConfig["draft"] = *cfg.Draft
		}
		if cfg.IfNoChanges != "" {
			handlerConfig["if_no_changes"] = cfg.IfNoChanges
		}
		if cfg.AllowEmpty {
			handlerConfig["allow_empty"] = cfg.AllowEmpty
		}
		if cfg.Expires > 0 {
			handlerConfig["expires"] = cfg.Expires
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		// Add base branch (required for git operations)
		handlerConfig["base_branch"] = "${{ github.ref_name }}"
		// Add max patch size
		maxPatchSize := 1024 // default 1024 KB
		if data.SafeOutputs.MaximumPatchSize > 0 {
			maxPatchSize = data.SafeOutputs.MaximumPatchSize
		}
		handlerConfig["max_patch_size"] = maxPatchSize
		config["create_pull_request"] = handlerConfig
	}

	if data.SafeOutputs.PushToPullRequestBranch != nil {
		cfg := data.SafeOutputs.PushToPullRequestBranch
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if cfg.TitlePrefix != "" {
			handlerConfig["title_prefix"] = cfg.TitlePrefix
		}
		if len(cfg.Labels) > 0 {
			handlerConfig["labels"] = cfg.Labels
		}
		if cfg.IfNoChanges != "" {
			handlerConfig["if_no_changes"] = cfg.IfNoChanges
		}
		if cfg.CommitTitleSuffix != "" {
			handlerConfig["commit_title_suffix"] = cfg.CommitTitleSuffix
		}
		// Add base branch (required for git operations)
		handlerConfig["base_branch"] = "${{ github.ref_name }}"
		// Add max patch size
		maxPatchSize := 1024 // default 1024 KB
		if data.SafeOutputs.MaximumPatchSize > 0 {
			maxPatchSize = data.SafeOutputs.MaximumPatchSize
		}
		handlerConfig["max_patch_size"] = maxPatchSize
		config["push_to_pull_request_branch"] = handlerConfig
	}

	if data.SafeOutputs.UpdatePullRequests != nil {
		cfg := data.SafeOutputs.UpdatePullRequests
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		// Boolean pointer fields indicate which fields can be updated
		// Default to true if not specified (backward compatibility)
		if cfg.Title != nil {
			handlerConfig["allow_title"] = *cfg.Title
		} else {
			handlerConfig["allow_title"] = true
		}
		if cfg.Body != nil {
			handlerConfig["allow_body"] = *cfg.Body
		} else {
			handlerConfig["allow_body"] = true
		}
		// Add default operation if specified
		if cfg.Operation != nil {
			handlerConfig["default_operation"] = *cfg.Operation
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["update_pull_request"] = handlerConfig
	}

	if data.SafeOutputs.ClosePullRequests != nil {
		cfg := data.SafeOutputs.ClosePullRequests
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.Target != "" {
			handlerConfig["target"] = cfg.Target
		}
		if len(cfg.RequiredLabels) > 0 {
			handlerConfig["required_labels"] = cfg.RequiredLabels
		}
		if cfg.RequiredTitlePrefix != "" {
			handlerConfig["required_title_prefix"] = cfg.RequiredTitlePrefix
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["close_pull_request"] = handlerConfig
	}

	if data.SafeOutputs.HideComment != nil {
		cfg := data.SafeOutputs.HideComment
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.AllowedReasons) > 0 {
			handlerConfig["allowed_reasons"] = cfg.AllowedReasons
		}
		if cfg.TargetRepoSlug != "" {
			handlerConfig["target-repo"] = cfg.TargetRepoSlug
		}
		if len(cfg.AllowedRepos) > 0 {
			handlerConfig["allowed_repos"] = cfg.AllowedRepos
		}
		config["hide_comment"] = handlerConfig
	}

	if data.SafeOutputs.DispatchWorkflow != nil {
		cfg := data.SafeOutputs.DispatchWorkflow
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if len(cfg.Workflows) > 0 {
			handlerConfig["workflows"] = cfg.Workflows
		}
		config["dispatch_workflow"] = handlerConfig
	}

	// Note: CreateProjects and CreateProjectStatusUpdates are handled by the project handler manager
	// (see addProjectHandlerManagerConfigEnvVar) because they require GH_AW_PROJECT_GITHUB_TOKEN

	if data.SafeOutputs.MissingTool != nil {
		cfg := data.SafeOutputs.MissingTool
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		config["missing_tool"] = handlerConfig
	}

	if data.SafeOutputs.MissingData != nil {
		cfg := data.SafeOutputs.MissingData
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		config["missing_data"] = handlerConfig
	}

	if data.SafeOutputs.AutofixCodeScanningAlert != nil {
		cfg := data.SafeOutputs.AutofixCodeScanningAlert
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.GitHubToken != "" {
			handlerConfig["github-token"] = cfg.GitHubToken
		}
		config["autofix_code_scanning_alert"] = handlerConfig
	}

	// Only add the env var if there are handlers to configure
	if len(config) > 0 {
		configJSON, err := json.Marshal(config)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Failed to marshal handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
	}
}

// addProjectHandlerManagerConfigEnvVar adds the GH_AW_SAFE_OUTPUTS_PROJECT_HANDLER_CONFIG environment variable
// containing JSON configuration for project-related safe output handlers (create_project, create_project_status_update).
// These handlers require GH_AW_PROJECT_GITHUB_TOKEN and are processed separately from the main handler manager.
func (c *Compiler) addProjectHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	config := make(map[string]map[string]any)

	// Add config for project-related safe output types
	if data.SafeOutputs.CreateProjects != nil {
		cfg := data.SafeOutputs.CreateProjects
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.TargetOwner != "" {
			handlerConfig["target_owner"] = cfg.TargetOwner
		}
		if cfg.TitlePrefix != "" {
			handlerConfig["title_prefix"] = cfg.TitlePrefix
		}
		if cfg.GitHubToken != "" {
			handlerConfig["github-token"] = cfg.GitHubToken
		}
		if len(cfg.Views) > 0 {
			handlerConfig["views"] = cfg.Views
		}
		config["create_project"] = handlerConfig
	}

	if data.SafeOutputs.CreateProjectStatusUpdates != nil {
		cfg := data.SafeOutputs.CreateProjectStatusUpdates
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.GitHubToken != "" {
			handlerConfig["github-token"] = cfg.GitHubToken
		}
		config["create_project_status_update"] = handlerConfig
	}

	if data.SafeOutputs.UpdateProjects != nil {
		cfg := data.SafeOutputs.UpdateProjects
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.GitHubToken != "" {
			handlerConfig["github-token"] = cfg.GitHubToken
		}
		if len(cfg.Views) > 0 {
			handlerConfig["views"] = cfg.Views
		}
		if len(cfg.FieldDefinitions) > 0 {
			handlerConfig["field_definitions"] = cfg.FieldDefinitions
		}
		config["update_project"] = handlerConfig
	}

	if data.SafeOutputs.CopyProjects != nil {
		cfg := data.SafeOutputs.CopyProjects
		handlerConfig := make(map[string]any)
		if cfg.Max > 0 {
			handlerConfig["max"] = cfg.Max
		}
		if cfg.GitHubToken != "" {
			handlerConfig["github-token"] = cfg.GitHubToken
		}
		if cfg.SourceProject != "" {
			handlerConfig["source_project"] = cfg.SourceProject
		}
		if cfg.TargetOwner != "" {
			handlerConfig["target_owner"] = cfg.TargetOwner
		}
		config["copy_project"] = handlerConfig
	}

	// Only add the env var if there are project handlers to configure
	if len(config) > 0 {
		configJSON, err := json.Marshal(config)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Failed to marshal project handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_PROJECT_HANDLER_CONFIG: %q\n", configStr))
	}
}

// addAllSafeOutputConfigEnvVars adds environment variables for all enabled safe output types
