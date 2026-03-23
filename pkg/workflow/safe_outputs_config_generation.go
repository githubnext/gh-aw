package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Configuration Generation
// ========================================
//
// This file is responsible for transforming a SafeOutputsConfig into the
// normalized JSON configuration objects consumed by the safe-outputs MCP server.
//
// Helper functions for building per-tool config maps are in safe_outputs_config_helpers.go.
//
// # Dual Config Generation Systems
//
// There are two code paths that generate safe-output handler configuration:
//
//  1. generateSafeOutputsConfig() (this file) — produces the GH_AW_SAFE_OUTPUTS_CONFIG_PATH
//     (config.json) consumed by the safe-outputs MCP server at startup.
//     Uses ad-hoc generateMax*Config() helper functions from safe_outputs_config_helpers.go.
//
//  2. addHandlerManagerConfigEnvVar() in compiler_safe_outputs_config.go — produces the
//     GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG env var consumed by the handler manager at runtime.
//     Uses the handlerRegistry + fluent handlerConfigBuilder API.
//
// These two paths serve distinct runtime purposes and must be kept in sync when adding
// new handler fields. When adding a new field to a handler, update both this file and
// compiler_safe_outputs_config.go. See compiler_safe_outputs_config.go for the full
// field contract defined by the handlerRegistry.

// generateSafeOutputsConfig transforms workflow safe-outputs configuration into a
// JSON string consumed by the safe-outputs MCP server at runtime.
func generateSafeOutputsConfig(data *WorkflowData) string {
	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs == nil {
		safeOutputsConfigLog.Print("No safe outputs configuration found, returning empty config")
		return ""
	}
	safeOutputsConfigLog.Print("Generating safe outputs configuration for workflow")
	// Create a simplified config object for validation
	safeOutputsConfig := make(map[string]any)

	// Handle safe-outputs configuration if present
	if data.SafeOutputs != nil {
		if data.SafeOutputs.CreateIssues != nil {
			config := generateMaxWithAllowedLabelsConfig(
				data.SafeOutputs.CreateIssues.Max,
				1, // default max
				data.SafeOutputs.CreateIssues.AllowedLabels,
			)
			// Add group flag if enabled
			if data.SafeOutputs.CreateIssues.Group != nil && *data.SafeOutputs.CreateIssues.Group == "true" {
				config["group"] = true
			}
			// Add expires value if set (0 means explicitly disabled or not set)
			if data.SafeOutputs.CreateIssues.Expires > 0 {
				config["expires"] = data.SafeOutputs.CreateIssues.Expires
			}
			addStagedIfTrue(config, data.SafeOutputs.CreateIssues.Staged)
			safeOutputsConfig["create_issue"] = config
		}
		if data.SafeOutputs.CreateAgentSessions != nil {
			config := generateMaxConfig(
				data.SafeOutputs.CreateAgentSessions.Max,
				1, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.CreateAgentSessions.Staged)
			safeOutputsConfig["create_agent_session"] = config
		}
		if data.SafeOutputs.AddComments != nil {
			additionalFields := make(map[string]any)
			addStagedIfTrue(additionalFields, data.SafeOutputs.AddComments.Staged)
			// Note: AddCommentsConfig has Target, TargetRepoSlug, AllowedRepos but not embedded SafeOutputTargetConfig
			// So we need to construct the target config manually
			targetConfig := SafeOutputTargetConfig{
				Target:         data.SafeOutputs.AddComments.Target,
				TargetRepoSlug: data.SafeOutputs.AddComments.TargetRepoSlug,
				AllowedRepos:   data.SafeOutputs.AddComments.AllowedRepos,
			}
			safeOutputsConfig["add_comment"] = generateTargetConfigWithRepos(
				targetConfig,
				data.SafeOutputs.AddComments.Max,
				1, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.CreateDiscussions != nil {
			config := generateMaxWithAllowedLabelsConfig(
				data.SafeOutputs.CreateDiscussions.Max,
				1, // default max
				data.SafeOutputs.CreateDiscussions.AllowedLabels,
			)
			// Add expires value if set (0 means explicitly disabled or not set)
			if data.SafeOutputs.CreateDiscussions.Expires > 0 {
				config["expires"] = data.SafeOutputs.CreateDiscussions.Expires
			}
			addStagedIfTrue(config, data.SafeOutputs.CreateDiscussions.Staged)
			safeOutputsConfig["create_discussion"] = config
		}
		if data.SafeOutputs.CloseDiscussions != nil {
			config := generateMaxWithDiscussionFieldsConfig(
				data.SafeOutputs.CloseDiscussions.Max,
				1, // default max
				data.SafeOutputs.CloseDiscussions.RequiredCategory,
				data.SafeOutputs.CloseDiscussions.RequiredLabels,
				data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix,
			)
			addStagedIfTrue(config, data.SafeOutputs.CloseDiscussions.Staged)
			safeOutputsConfig["close_discussion"] = config
		}
		if data.SafeOutputs.CloseIssues != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.CloseIssues.RequiredLabels) > 0 {
				additionalFields["required_labels"] = data.SafeOutputs.CloseIssues.RequiredLabels
			}
			if data.SafeOutputs.CloseIssues.RequiredTitlePrefix != "" {
				additionalFields["required_title_prefix"] = data.SafeOutputs.CloseIssues.RequiredTitlePrefix
			}
			addStagedIfTrue(additionalFields, data.SafeOutputs.CloseIssues.Staged)
			safeOutputsConfig["close_issue"] = generateTargetConfigWithRepos(
				data.SafeOutputs.CloseIssues.SafeOutputTargetConfig,
				data.SafeOutputs.CloseIssues.Max,
				1, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.ClosePullRequests != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.ClosePullRequests.RequiredLabels) > 0 {
				additionalFields["required_labels"] = data.SafeOutputs.ClosePullRequests.RequiredLabels
			}
			if data.SafeOutputs.ClosePullRequests.RequiredTitlePrefix != "" {
				additionalFields["required_title_prefix"] = data.SafeOutputs.ClosePullRequests.RequiredTitlePrefix
			}
			if data.SafeOutputs.ClosePullRequests.GitHubToken != "" {
				additionalFields["github-token"] = data.SafeOutputs.ClosePullRequests.GitHubToken
			}
			addStagedIfTrue(additionalFields, data.SafeOutputs.ClosePullRequests.Staged)
			safeOutputsConfig["close_pull_request"] = generateTargetConfigWithRepos(
				data.SafeOutputs.ClosePullRequests.SafeOutputTargetConfig,
				data.SafeOutputs.ClosePullRequests.Max,
				1, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			config := generatePullRequestConfig(
				data.SafeOutputs.CreatePullRequests,
				1, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.CreatePullRequests.Staged)
			safeOutputsConfig["create_pull_request"] = config
		}
		if data.SafeOutputs.CreatePullRequestReviewComments != nil {
			config := generateMaxConfig(
				data.SafeOutputs.CreatePullRequestReviewComments.Max,
				10, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.CreatePullRequestReviewComments.Staged)
			safeOutputsConfig["create_pull_request_review_comment"] = config
		}
		if data.SafeOutputs.SubmitPullRequestReview != nil {
			config := generateMaxConfig(
				data.SafeOutputs.SubmitPullRequestReview.Max,
				1, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.SubmitPullRequestReview.Staged)
			safeOutputsConfig["submit_pull_request_review"] = config
		}
		if data.SafeOutputs.ReplyToPullRequestReviewComment != nil {
			additionalFields := newHandlerConfigBuilder().
				AddTemplatableBool("footer", data.SafeOutputs.ReplyToPullRequestReviewComment.Footer).
				AddIfTrue("staged", data.SafeOutputs.ReplyToPullRequestReviewComment.Staged).
				Build()
			safeOutputsConfig["reply_to_pull_request_review_comment"] = generateTargetConfigWithRepos(
				data.SafeOutputs.ReplyToPullRequestReviewComment.SafeOutputTargetConfig,
				data.SafeOutputs.ReplyToPullRequestReviewComment.Max,
				10, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.ResolvePullRequestReviewThread != nil {
			config := generateMaxConfig(
				data.SafeOutputs.ResolvePullRequestReviewThread.Max,
				10, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.ResolvePullRequestReviewThread.Staged)
			safeOutputsConfig["resolve_pull_request_review_thread"] = config
		}
		if data.SafeOutputs.CreateCodeScanningAlerts != nil {
			config := generateMaxConfig(
				data.SafeOutputs.CreateCodeScanningAlerts.Max,
				0, // default: unlimited
			)
			addStagedIfTrue(config, data.SafeOutputs.CreateCodeScanningAlerts.Staged)
			safeOutputsConfig["create_code_scanning_alert"] = config
		}
		if data.SafeOutputs.AutofixCodeScanningAlert != nil {
			config := generateMaxConfig(
				data.SafeOutputs.AutofixCodeScanningAlert.Max,
				10, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.AutofixCodeScanningAlert.Staged)
			safeOutputsConfig["autofix_code_scanning_alert"] = config
		}
		if data.SafeOutputs.AddLabels != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.AddLabels.Allowed) > 0 {
				additionalFields["allowed"] = data.SafeOutputs.AddLabels.Allowed
			}
			if len(data.SafeOutputs.AddLabels.Blocked) > 0 {
				additionalFields["blocked"] = data.SafeOutputs.AddLabels.Blocked
			}
			addStagedIfTrue(additionalFields, data.SafeOutputs.AddLabels.Staged)
			safeOutputsConfig["add_labels"] = generateTargetConfigWithRepos(
				data.SafeOutputs.AddLabels.SafeOutputTargetConfig,
				data.SafeOutputs.AddLabels.Max,
				3, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.RemoveLabels != nil {
			config := generateMaxWithAllowedConfig(
				data.SafeOutputs.RemoveLabels.Max,
				3, // default max
				data.SafeOutputs.RemoveLabels.Allowed,
			)
			addStagedIfTrue(config, data.SafeOutputs.RemoveLabels.Staged)
			safeOutputsConfig["remove_labels"] = config
		}
		if data.SafeOutputs.AddReviewer != nil {
			config := generateMaxWithReviewersConfig(
				data.SafeOutputs.AddReviewer.Max,
				3, // default max
				data.SafeOutputs.AddReviewer.Reviewers,
			)
			addStagedIfTrue(config, data.SafeOutputs.AddReviewer.Staged)
			safeOutputsConfig["add_reviewer"] = config
		}
		if data.SafeOutputs.AssignMilestone != nil {
			config := generateMaxWithAllowedConfig(
				data.SafeOutputs.AssignMilestone.Max,
				1, // default max
				data.SafeOutputs.AssignMilestone.Allowed,
			)
			addStagedIfTrue(config, data.SafeOutputs.AssignMilestone.Staged)
			safeOutputsConfig["assign_milestone"] = config
		}
		if data.SafeOutputs.AssignToAgent != nil {
			config := generateAssignToAgentConfig(
				data.SafeOutputs.AssignToAgent.Max,
				1, // default max
				data.SafeOutputs.AssignToAgent.DefaultAgent,
				data.SafeOutputs.AssignToAgent.Target,
				data.SafeOutputs.AssignToAgent.Allowed,
			)
			addStagedIfTrue(config, data.SafeOutputs.AssignToAgent.Staged)
			safeOutputsConfig["assign_to_agent"] = config
		}
		if data.SafeOutputs.AssignToUser != nil {
			config := generateMaxWithAllowedAndBlockedConfig(
				data.SafeOutputs.AssignToUser.Max,
				1, // default max
				data.SafeOutputs.AssignToUser.Allowed,
				data.SafeOutputs.AssignToUser.Blocked,
			)
			addStagedIfTrue(config, data.SafeOutputs.AssignToUser.Staged)
			safeOutputsConfig["assign_to_user"] = config
		}
		if data.SafeOutputs.UnassignFromUser != nil {
			config := generateMaxWithAllowedAndBlockedConfig(
				data.SafeOutputs.UnassignFromUser.Max,
				1, // default max
				data.SafeOutputs.UnassignFromUser.Allowed,
				data.SafeOutputs.UnassignFromUser.Blocked,
			)
			addStagedIfTrue(config, data.SafeOutputs.UnassignFromUser.Staged)
			safeOutputsConfig["unassign_from_user"] = config
		}
		if data.SafeOutputs.UpdateIssues != nil {
			config := generateMaxConfig(
				data.SafeOutputs.UpdateIssues.Max,
				1, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.UpdateIssues.Staged)
			safeOutputsConfig["update_issue"] = config
		}
		if data.SafeOutputs.UpdateDiscussions != nil {
			config := generateMaxWithAllowedLabelsConfig(
				data.SafeOutputs.UpdateDiscussions.Max,
				1, // default max
				data.SafeOutputs.UpdateDiscussions.AllowedLabels,
			)
			addStagedIfTrue(config, data.SafeOutputs.UpdateDiscussions.Staged)
			safeOutputsConfig["update_discussion"] = config
		}
		if data.SafeOutputs.UpdatePullRequests != nil {
			config := generateMaxConfig(
				data.SafeOutputs.UpdatePullRequests.Max,
				1, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.UpdatePullRequests.Staged)
			safeOutputsConfig["update_pull_request"] = config
		}
		if data.SafeOutputs.MarkPullRequestAsReadyForReview != nil {
			config := generateMaxConfig(
				data.SafeOutputs.MarkPullRequestAsReadyForReview.Max,
				10, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.MarkPullRequestAsReadyForReview.Staged)
			safeOutputsConfig["mark_pull_request_as_ready_for_review"] = config
		}
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			config := generateMaxWithTargetConfig(
				data.SafeOutputs.PushToPullRequestBranch.Max,
				1, // default max: 1
				data.SafeOutputs.PushToPullRequestBranch.Target,
			)
			addStagedIfTrue(config, data.SafeOutputs.PushToPullRequestBranch.Staged)
			safeOutputsConfig["push_to_pull_request_branch"] = config
		}
		if data.SafeOutputs.UploadAssets != nil {
			config := generateMaxConfig(
				data.SafeOutputs.UploadAssets.Max,
				0, // default: unlimited
			)
			addStagedIfTrue(config, data.SafeOutputs.UploadAssets.Staged)
			safeOutputsConfig["upload_asset"] = config
		}
		if data.SafeOutputs.MissingTool != nil {
			// Generate config for missing_tool with issue creation support
			missingToolConfig := make(map[string]any)

			// Add max if set
			if data.SafeOutputs.MissingTool.Max != nil {
				missingToolConfig["max"] = resolveMaxForConfig(data.SafeOutputs.MissingTool.Max, 0)
			}

			// Add issue creation config if enabled
			if data.SafeOutputs.MissingTool.CreateIssue {
				createIssueConfig := make(map[string]any)
				createIssueConfig["max"] = 1 // Only create one issue per workflow run

				if data.SafeOutputs.MissingTool.TitlePrefix != "" {
					createIssueConfig["title_prefix"] = data.SafeOutputs.MissingTool.TitlePrefix
				}

				if len(data.SafeOutputs.MissingTool.Labels) > 0 {
					createIssueConfig["labels"] = data.SafeOutputs.MissingTool.Labels
				}

				safeOutputsConfig["create_missing_tool_issue"] = createIssueConfig
			}

			safeOutputsConfig["missing_tool"] = missingToolConfig
		}
		if data.SafeOutputs.MissingData != nil {
			// Generate config for missing_data with issue creation support
			missingDataConfig := make(map[string]any)

			// Add max if set
			if data.SafeOutputs.MissingData.Max != nil {
				missingDataConfig["max"] = resolveMaxForConfig(data.SafeOutputs.MissingData.Max, 0)
			}

			// Add issue creation config if enabled
			if data.SafeOutputs.MissingData.CreateIssue {
				createIssueConfig := make(map[string]any)
				createIssueConfig["max"] = 1 // Only create one issue per workflow run

				if data.SafeOutputs.MissingData.TitlePrefix != "" {
					createIssueConfig["title_prefix"] = data.SafeOutputs.MissingData.TitlePrefix
				}

				if len(data.SafeOutputs.MissingData.Labels) > 0 {
					createIssueConfig["labels"] = data.SafeOutputs.MissingData.Labels
				}

				safeOutputsConfig["create_missing_data_issue"] = createIssueConfig
			}

			safeOutputsConfig["missing_data"] = missingDataConfig
		}
		if data.SafeOutputs.UpdateProjects != nil {
			additionalFields := make(map[string]any)
			addStagedIfTrue(additionalFields, data.SafeOutputs.UpdateProjects.Staged)
			safeOutputsConfig["update_project"] = generateTargetConfigWithRepos(
				SafeOutputTargetConfig{
					TargetRepoSlug: data.SafeOutputs.UpdateProjects.TargetRepoSlug,
					AllowedRepos:   data.SafeOutputs.UpdateProjects.AllowedRepos,
				},
				data.SafeOutputs.UpdateProjects.Max,
				10, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.CreateProjectStatusUpdates != nil {
			config := generateMaxConfig(
				data.SafeOutputs.CreateProjectStatusUpdates.Max,
				10, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.CreateProjectStatusUpdates.Staged)
			safeOutputsConfig["create_project_status_update"] = config
		}
		if data.SafeOutputs.CreateProjects != nil {
			config := generateMaxConfig(
				data.SafeOutputs.CreateProjects.Max,
				1, // default max
			)
			// Add target-owner if specified
			if data.SafeOutputs.CreateProjects.TargetOwner != "" {
				config["target_owner"] = data.SafeOutputs.CreateProjects.TargetOwner
			}
			// Add title-prefix if specified
			if data.SafeOutputs.CreateProjects.TitlePrefix != "" {
				config["title_prefix"] = data.SafeOutputs.CreateProjects.TitlePrefix
			}
			addStagedIfTrue(config, data.SafeOutputs.CreateProjects.Staged)
			safeOutputsConfig["create_project"] = config
		}
		if data.SafeOutputs.UpdateRelease != nil {
			config := generateMaxConfig(
				data.SafeOutputs.UpdateRelease.Max,
				1, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.UpdateRelease.Staged)
			safeOutputsConfig["update_release"] = config
		}
		if data.SafeOutputs.LinkSubIssue != nil {
			config := generateMaxConfig(
				data.SafeOutputs.LinkSubIssue.Max,
				5, // default max
			)
			addStagedIfTrue(config, data.SafeOutputs.LinkSubIssue.Staged)
			safeOutputsConfig["link_sub_issue"] = config
		}
		if data.SafeOutputs.NoOp != nil {
			safeOutputsConfig["noop"] = generateMaxConfig(
				data.SafeOutputs.NoOp.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.HideComment != nil {
			config := generateHideCommentConfig(
				data.SafeOutputs.HideComment.Max,
				5, // default max
				data.SafeOutputs.HideComment.AllowedReasons,
			)
			addStagedIfTrue(config, data.SafeOutputs.HideComment.Staged)
			safeOutputsConfig["hide_comment"] = config
		}
		if data.SafeOutputs.SetIssueType != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.SetIssueType.Allowed) > 0 {
				additionalFields["allowed"] = data.SafeOutputs.SetIssueType.Allowed
			}
			addStagedIfTrue(additionalFields, data.SafeOutputs.SetIssueType.Staged)
			safeOutputsConfig["set_issue_type"] = generateTargetConfigWithRepos(
				data.SafeOutputs.SetIssueType.SafeOutputTargetConfig,
				data.SafeOutputs.SetIssueType.Max,
				5, // default max
				additionalFields,
			)
		}
	}

	// Add safe-jobs configuration from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		safeOutputsConfigLog.Printf("Processing %d safe job configurations", len(data.SafeOutputs.Jobs))
		for jobName, jobConfig := range data.SafeOutputs.Jobs {
			safeOutputsConfigLog.Printf("Generating config for safe job: %s", jobName)
			safeJobConfig := map[string]any{}

			// Add description if present
			if jobConfig.Description != "" {
				safeJobConfig["description"] = jobConfig.Description
			}

			// Add output if present
			if jobConfig.Output != "" {
				safeJobConfig["output"] = jobConfig.Output
			}

			// Add inputs information
			if len(jobConfig.Inputs) > 0 {
				inputsConfig := make(map[string]any)
				for inputName, inputDef := range jobConfig.Inputs {
					inputConfig := map[string]any{
						"type":        inputDef.Type,
						"description": inputDef.Description,
						"required":    inputDef.Required,
					}
					if inputDef.Default != "" {
						inputConfig["default"] = inputDef.Default
					}
					if len(inputDef.Options) > 0 {
						inputConfig["options"] = inputDef.Options
					}
					inputsConfig[inputName] = inputConfig
				}
				safeJobConfig["inputs"] = inputsConfig
			}

			safeOutputsConfig[jobName] = safeJobConfig
		}
	}

	// Add safe-scripts configuration from SafeOutputs.Scripts
	// Scripts run in the handler loop, so they are registered the same way as jobs in the config
	if len(data.SafeOutputs.Scripts) > 0 {
		safeOutputsConfigLog.Printf("Processing %d safe script configurations", len(data.SafeOutputs.Scripts))
		for scriptName, scriptConfig := range data.SafeOutputs.Scripts {
			normalizedName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
			safeOutputsConfigLog.Printf("Generating config for safe script: %s (normalized: %s)", scriptName, normalizedName)
			safeScriptConfigMap := map[string]any{}

			// Add description if present
			if scriptConfig.Description != "" {
				safeScriptConfigMap["description"] = scriptConfig.Description
			}

			// Add inputs information
			if len(scriptConfig.Inputs) > 0 {
				inputsConfig := make(map[string]any)
				for inputName, inputDef := range scriptConfig.Inputs {
					inputConfig := map[string]any{
						"type":        inputDef.Type,
						"description": inputDef.Description,
						"required":    inputDef.Required,
					}
					if inputDef.Default != "" {
						inputConfig["default"] = inputDef.Default
					}
					if len(inputDef.Options) > 0 {
						inputConfig["options"] = inputDef.Options
					}
					inputsConfig[inputName] = inputConfig
				}
				safeScriptConfigMap["inputs"] = inputsConfig
			}

			safeOutputsConfig[normalizedName] = safeScriptConfigMap
		}
	}

	// Add mentions configuration
	if data.SafeOutputs.Mentions != nil {
		mentionsConfig := make(map[string]any)

		// Handle enabled flag (simple boolean mode)
		if data.SafeOutputs.Mentions.Enabled != nil {
			mentionsConfig["enabled"] = *data.SafeOutputs.Mentions.Enabled
		}

		// Handle allow-team-members
		if data.SafeOutputs.Mentions.AllowTeamMembers != nil {
			mentionsConfig["allowTeamMembers"] = *data.SafeOutputs.Mentions.AllowTeamMembers
		}

		// Handle allow-context
		if data.SafeOutputs.Mentions.AllowContext != nil {
			mentionsConfig["allowContext"] = *data.SafeOutputs.Mentions.AllowContext
		}

		// Handle allowed list
		if len(data.SafeOutputs.Mentions.Allowed) > 0 {
			mentionsConfig["allowed"] = data.SafeOutputs.Mentions.Allowed
		}

		// Handle max
		if data.SafeOutputs.Mentions.Max != nil {
			mentionsConfig["max"] = *data.SafeOutputs.Mentions.Max
		}

		// Only add mentions config if it has any fields
		if len(mentionsConfig) > 0 {
			safeOutputsConfig["mentions"] = mentionsConfig
		}
	}

	// Add dispatch-workflow configuration
	if data.SafeOutputs.DispatchWorkflow != nil {
		dispatchWorkflowConfig := map[string]any{}

		// Include workflows list
		if len(data.SafeOutputs.DispatchWorkflow.Workflows) > 0 {
			dispatchWorkflowConfig["workflows"] = data.SafeOutputs.DispatchWorkflow.Workflows
		}

		// Include workflow files mapping (file extension for each workflow)
		if len(data.SafeOutputs.DispatchWorkflow.WorkflowFiles) > 0 {
			dispatchWorkflowConfig["workflow_files"] = data.SafeOutputs.DispatchWorkflow.WorkflowFiles
		}

		// Include max count
		dispatchWorkflowConfig["max"] = resolveMaxForConfig(data.SafeOutputs.DispatchWorkflow.Max, 1)

		addStagedIfTrue(dispatchWorkflowConfig, data.SafeOutputs.DispatchWorkflow.Staged)

		// Only add if it has fields
		if len(dispatchWorkflowConfig) > 0 {
			safeOutputsConfig["dispatch_workflow"] = dispatchWorkflowConfig
		}
	}

	// Add dispatch_repository configuration
	if data.SafeOutputs.DispatchRepository != nil && len(data.SafeOutputs.DispatchRepository.Tools) > 0 {
		tools := make(map[string]any, len(data.SafeOutputs.DispatchRepository.Tools))
		for toolKey, tool := range data.SafeOutputs.DispatchRepository.Tools {
			toolCfg := map[string]any{
				"workflow":   tool.Workflow,
				"event_type": tool.EventType,
				"max":        resolveMaxForConfig(tool.Max, 1),
			}
			if tool.Repository != "" {
				toolCfg["repository"] = tool.Repository
			}
			if len(tool.AllowedRepositories) > 0 {
				toolCfg["allowed_repositories"] = tool.AllowedRepositories
			}
			if tool.GitHubToken != "" {
				toolCfg["github-token"] = tool.GitHubToken
			}
			if tool.Staged {
				toolCfg["staged"] = true
			}
			if tool.Description != "" {
				toolCfg["description"] = tool.Description
			}
			tools[toolKey] = toolCfg
		}
		safeOutputsConfig["dispatch_repository"] = map[string]any{"tools": tools}
	}

	// Add call-workflow configuration
	if data.SafeOutputs.CallWorkflow != nil {
		callWorkflowConfig := map[string]any{}

		// Include workflows list
		if len(data.SafeOutputs.CallWorkflow.Workflows) > 0 {
			callWorkflowConfig["workflows"] = data.SafeOutputs.CallWorkflow.Workflows
		}

		// Include workflow files mapping (relative path for each workflow)
		if len(data.SafeOutputs.CallWorkflow.WorkflowFiles) > 0 {
			callWorkflowConfig["workflow_files"] = data.SafeOutputs.CallWorkflow.WorkflowFiles
		}

		// Include max count
		callWorkflowConfig["max"] = resolveMaxForConfig(data.SafeOutputs.CallWorkflow.Max, 1)

		// Only add if it has fields
		if len(callWorkflowConfig) > 0 {
			safeOutputsConfig["call_workflow"] = callWorkflowConfig
		}
	}

	// Add max-bot-mentions if set (templatable integer)
	if data.SafeOutputs.MaxBotMentions != nil {
		v := *data.SafeOutputs.MaxBotMentions
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			safeOutputsConfig["max_bot_mentions"] = n
		} else if strings.HasPrefix(v, "${{") {
			safeOutputsConfig["max_bot_mentions"] = v
		}
	}

	// Add push_repo_memory config if repo-memory is configured
	// This enables the push_repo_memory MCP tool for early size validation during agent session
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		var memories []map[string]any
		for _, memory := range data.RepoMemoryConfig.Memories {
			memories = append(memories, map[string]any{
				"id":             memory.ID,
				"dir":            "/tmp/gh-aw/repo-memory/" + memory.ID,
				"max_file_size":  memory.MaxFileSize,
				"max_patch_size": memory.MaxPatchSize,
				"max_file_count": memory.MaxFileCount,
			})
		}
		safeOutputsConfig["push_repo_memory"] = map[string]any{
			"memories": memories,
		}
		safeOutputsConfigLog.Printf("Added push_repo_memory config with %d memory entries", len(memories))
	}

	configJSON, _ := json.Marshal(safeOutputsConfig)
	safeOutputsConfigLog.Printf("Safe outputs config generation complete: %d tool types configured", len(safeOutputsConfig))
	return string(configJSON)
}

// generateCustomJobToolDefinition creates an MCP tool definition for a custom safe-output job.
// Returns a map representing the tool definition in MCP format with name, description, and inputSchema.
func generateCustomJobToolDefinition(jobName string, jobConfig *SafeJobConfig) map[string]any {
	safeOutputsConfigLog.Printf("Generating tool definition for custom job: %s", jobName)

	description := jobConfig.Description
	if description == "" {
		description = fmt.Sprintf("Execute the %s custom job", jobName)
	}

	inputSchema := map[string]any{
		"type":                 "object",
		"properties":           make(map[string]any),
		"additionalProperties": false,
	}

	var requiredFields []string
	properties := inputSchema["properties"].(map[string]any)

	for inputName, inputDef := range jobConfig.Inputs {
		property := map[string]any{}

		if inputDef.Description != "" {
			property["description"] = inputDef.Description
		}

		// Convert type to JSON Schema type
		switch inputDef.Type {
		case "choice":
			// Choice inputs are strings with enum constraints
			property["type"] = "string"
			if len(inputDef.Options) > 0 {
				property["enum"] = inputDef.Options
			}
		case "boolean":
			property["type"] = "boolean"
		case "number":
			property["type"] = "number"
		default:
			// "string", empty string, or any unknown type defaults to string
			property["type"] = "string"
		}

		if inputDef.Default != nil {
			property["default"] = inputDef.Default
		}

		if inputDef.Required {
			requiredFields = append(requiredFields, inputName)
		}

		properties[inputName] = property
	}

	if len(requiredFields) > 0 {
		sort.Strings(requiredFields)
		inputSchema["required"] = requiredFields
	}

	safeOutputsConfigLog.Printf("Generated tool definition for %s with %d inputs, %d required",
		jobName, len(jobConfig.Inputs), len(requiredFields))

	return map[string]any{
		"name":        jobName,
		"description": description,
		"inputSchema": inputSchema,
	}
}
