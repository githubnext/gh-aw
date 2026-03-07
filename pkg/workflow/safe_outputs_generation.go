package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Configuration Generation
// ========================================

// generateCustomJobToolDefinition creates an MCP tool definition for a custom safe-output job
// Returns a map representing the tool definition in MCP format with name, description, and inputSchema
func generateCustomJobToolDefinition(jobName string, jobConfig *SafeJobConfig) map[string]any {
	safeOutputsConfigLog.Printf("Generating tool definition for custom job: %s", jobName)

	// Build the tool definition
	tool := map[string]any{
		"name": jobName,
	}

	// Add description if present
	if jobConfig.Description != "" {
		tool["description"] = jobConfig.Description
	} else {
		// Provide a default description if none is specified
		tool["description"] = fmt.Sprintf("Execute the %s custom job", jobName)
	}

	// Build the input schema
	inputSchema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}

	// Track required fields
	var requiredFields []string

	// Add each input to the schema
	if len(jobConfig.Inputs) > 0 {
		properties := inputSchema["properties"].(map[string]any)

		for inputName, inputDef := range jobConfig.Inputs {
			property := map[string]any{}

			// Add description
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
			case "string", "":
				// Default to string if type is not specified
				property["type"] = "string"
			default:
				// For any unknown type, default to string
				property["type"] = "string"
			}

			// Add default value if present
			if inputDef.Default != nil {
				property["default"] = inputDef.Default
			}

			// Track required fields
			if inputDef.Required {
				requiredFields = append(requiredFields, inputName)
			}

			properties[inputName] = property
		}
	}

	// Add required fields array if any inputs are required
	if len(requiredFields) > 0 {
		sort.Strings(requiredFields)
		inputSchema["required"] = requiredFields
	}

	// Prevent additional properties to maintain schema strictness
	inputSchema["additionalProperties"] = false

	tool["inputSchema"] = inputSchema

	safeOutputsConfigLog.Printf("Generated tool definition for %s with %d inputs, %d required",
		jobName, len(jobConfig.Inputs), len(requiredFields))

	return tool
}

func populateDispatchWorkflowFiles(data *WorkflowData, markdownPath string) {
	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return
	}

	if len(data.SafeOutputs.DispatchWorkflow.Workflows) == 0 {
		return
	}

	safeOutputsConfigLog.Printf("Populating workflow files for %d dispatch workflows", len(data.SafeOutputs.DispatchWorkflow.Workflows))

	// Initialize WorkflowFiles map if not already initialized
	if data.SafeOutputs.DispatchWorkflow.WorkflowFiles == nil {
		data.SafeOutputs.DispatchWorkflow.WorkflowFiles = make(map[string]string)
	}

	for _, workflowName := range data.SafeOutputs.DispatchWorkflow.Workflows {
		// Find the workflow file
		fileResult, err := findWorkflowFile(workflowName, markdownPath)
		if err != nil {
			safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
			continue
		}

		// Determine which file to use - priority: .lock.yml > .yml
		var extension string
		if fileResult.lockExists {
			extension = ".lock.yml"
		} else if fileResult.ymlExists {
			extension = ".yml"
		} else {
			safeOutputsConfigLog.Printf("Warning: workflow file not found for %s (only .md exists, needs compilation)", workflowName)
			continue
		}

		// Store the file extension for runtime use
		data.SafeOutputs.DispatchWorkflow.WorkflowFiles[workflowName] = extension
		safeOutputsConfigLog.Printf("Mapped workflow %s to extension %s", workflowName, extension)
	}
}

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
			safeOutputsConfig["create_issue"] = config
		}
		if data.SafeOutputs.CreateAgentSessions != nil {
			safeOutputsConfig["create_agent_session"] = generateMaxConfig(
				data.SafeOutputs.CreateAgentSessions.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.AddComments != nil {
			additionalFields := make(map[string]any)
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
			safeOutputsConfig["create_discussion"] = config
		}
		if data.SafeOutputs.CloseDiscussions != nil {
			safeOutputsConfig["close_discussion"] = generateMaxWithDiscussionFieldsConfig(
				data.SafeOutputs.CloseDiscussions.Max,
				1, // default max
				data.SafeOutputs.CloseDiscussions.RequiredCategory,
				data.SafeOutputs.CloseDiscussions.RequiredLabels,
				data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix,
			)
		}
		if data.SafeOutputs.CloseIssues != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.CloseIssues.RequiredLabels) > 0 {
				additionalFields["required_labels"] = data.SafeOutputs.CloseIssues.RequiredLabels
			}
			if data.SafeOutputs.CloseIssues.RequiredTitlePrefix != "" {
				additionalFields["required_title_prefix"] = data.SafeOutputs.CloseIssues.RequiredTitlePrefix
			}
			safeOutputsConfig["close_issue"] = generateTargetConfigWithRepos(
				data.SafeOutputs.CloseIssues.SafeOutputTargetConfig,
				data.SafeOutputs.CloseIssues.Max,
				1, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			safeOutputsConfig["create_pull_request"] = generatePullRequestConfig(
				data.SafeOutputs.CreatePullRequests,
				1, // default max
			)
		}
		if data.SafeOutputs.CreatePullRequestReviewComments != nil {
			safeOutputsConfig["create_pull_request_review_comment"] = generateMaxConfig(
				data.SafeOutputs.CreatePullRequestReviewComments.Max,
				10, // default max
			)
		}
		if data.SafeOutputs.SubmitPullRequestReview != nil {
			safeOutputsConfig["submit_pull_request_review"] = generateMaxConfig(
				data.SafeOutputs.SubmitPullRequestReview.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.ResolvePullRequestReviewThread != nil {
			safeOutputsConfig["resolve_pull_request_review_thread"] = generateMaxConfig(
				data.SafeOutputs.ResolvePullRequestReviewThread.Max,
				10, // default max
			)
		}
		if data.SafeOutputs.CreateCodeScanningAlerts != nil {
			safeOutputsConfig["create_code_scanning_alert"] = generateMaxConfig(
				data.SafeOutputs.CreateCodeScanningAlerts.Max,
				0, // default: unlimited
			)
		}
		if data.SafeOutputs.AutofixCodeScanningAlert != nil {
			safeOutputsConfig["autofix_code_scanning_alert"] = generateMaxConfig(
				data.SafeOutputs.AutofixCodeScanningAlert.Max,
				10, // default max
			)
		}
		if data.SafeOutputs.AddLabels != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.AddLabels.Allowed) > 0 {
				additionalFields["allowed"] = data.SafeOutputs.AddLabels.Allowed
			}
			if len(data.SafeOutputs.AddLabels.Blocked) > 0 {
				additionalFields["blocked"] = data.SafeOutputs.AddLabels.Blocked
			}
			safeOutputsConfig["add_labels"] = generateTargetConfigWithRepos(
				data.SafeOutputs.AddLabels.SafeOutputTargetConfig,
				data.SafeOutputs.AddLabels.Max,
				3, // default max
				additionalFields,
			)
		}
		if data.SafeOutputs.RemoveLabels != nil {
			safeOutputsConfig["remove_labels"] = generateMaxWithAllowedConfig(
				data.SafeOutputs.RemoveLabels.Max,
				3, // default max
				data.SafeOutputs.RemoveLabels.Allowed,
			)
		}
		if data.SafeOutputs.AddReviewer != nil {
			safeOutputsConfig["add_reviewer"] = generateMaxWithReviewersConfig(
				data.SafeOutputs.AddReviewer.Max,
				3, // default max
				data.SafeOutputs.AddReviewer.Reviewers,
			)
		}
		if data.SafeOutputs.AssignMilestone != nil {
			safeOutputsConfig["assign_milestone"] = generateMaxWithAllowedConfig(
				data.SafeOutputs.AssignMilestone.Max,
				1, // default max
				data.SafeOutputs.AssignMilestone.Allowed,
			)
		}
		if data.SafeOutputs.AssignToAgent != nil {
			safeOutputsConfig["assign_to_agent"] = generateAssignToAgentConfig(
				data.SafeOutputs.AssignToAgent.Max,
				1, // default max
				data.SafeOutputs.AssignToAgent.DefaultAgent,
				data.SafeOutputs.AssignToAgent.Target,
				data.SafeOutputs.AssignToAgent.Allowed,
			)
		}
		if data.SafeOutputs.AssignToUser != nil {
			safeOutputsConfig["assign_to_user"] = generateMaxWithAllowedAndBlockedConfig(
				data.SafeOutputs.AssignToUser.Max,
				1, // default max
				data.SafeOutputs.AssignToUser.Allowed,
				data.SafeOutputs.AssignToUser.Blocked,
			)
		}
		if data.SafeOutputs.UnassignFromUser != nil {
			safeOutputsConfig["unassign_from_user"] = generateMaxWithAllowedAndBlockedConfig(
				data.SafeOutputs.UnassignFromUser.Max,
				1, // default max
				data.SafeOutputs.UnassignFromUser.Allowed,
				data.SafeOutputs.UnassignFromUser.Blocked,
			)
		}
		if data.SafeOutputs.UpdateIssues != nil {
			safeOutputsConfig["update_issue"] = generateMaxConfig(
				data.SafeOutputs.UpdateIssues.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.UpdateDiscussions != nil {
			safeOutputsConfig["update_discussion"] = generateMaxWithAllowedLabelsConfig(
				data.SafeOutputs.UpdateDiscussions.Max,
				1, // default max
				data.SafeOutputs.UpdateDiscussions.AllowedLabels,
			)
		}
		if data.SafeOutputs.UpdatePullRequests != nil {
			safeOutputsConfig["update_pull_request"] = generateMaxConfig(
				data.SafeOutputs.UpdatePullRequests.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.MarkPullRequestAsReadyForReview != nil {
			safeOutputsConfig["mark_pull_request_as_ready_for_review"] = generateMaxConfig(
				data.SafeOutputs.MarkPullRequestAsReadyForReview.Max,
				10, // default max
			)
		}
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			safeOutputsConfig["push_to_pull_request_branch"] = generateMaxWithTargetConfig(
				data.SafeOutputs.PushToPullRequestBranch.Max,
				0, // default: unlimited
				data.SafeOutputs.PushToPullRequestBranch.Target,
			)
		}
		if data.SafeOutputs.UploadAssets != nil {
			safeOutputsConfig["upload_asset"] = generateMaxConfig(
				data.SafeOutputs.UploadAssets.Max,
				0, // default: unlimited
			)
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
			safeOutputsConfig["update_project"] = generateMaxConfig(
				data.SafeOutputs.UpdateProjects.Max,
				10, // default max
			)
		}
		if data.SafeOutputs.CreateProjectStatusUpdates != nil {
			safeOutputsConfig["create_project_status_update"] = generateMaxConfig(
				data.SafeOutputs.CreateProjectStatusUpdates.Max,
				10, // default max
			)
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
			safeOutputsConfig["create_project"] = config
		}
		if data.SafeOutputs.UpdateRelease != nil {
			safeOutputsConfig["update_release"] = generateMaxConfig(
				data.SafeOutputs.UpdateRelease.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.LinkSubIssue != nil {
			safeOutputsConfig["link_sub_issue"] = generateMaxConfig(
				data.SafeOutputs.LinkSubIssue.Max,
				5, // default max
			)
		}
		if data.SafeOutputs.NoOp != nil {
			safeOutputsConfig["noop"] = generateMaxConfig(
				data.SafeOutputs.NoOp.Max,
				1, // default max
			)
		}
		if data.SafeOutputs.HideComment != nil {
			safeOutputsConfig["hide_comment"] = generateHideCommentConfig(
				data.SafeOutputs.HideComment.Max,
				5, // default max
				data.SafeOutputs.HideComment.AllowedReasons,
			)
		}
		if data.SafeOutputs.SetIssueType != nil {
			additionalFields := make(map[string]any)
			if len(data.SafeOutputs.SetIssueType.Allowed) > 0 {
				additionalFields["allowed"] = data.SafeOutputs.SetIssueType.Allowed
			}
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

		// Only add if it has fields
		if len(dispatchWorkflowConfig) > 0 {
			safeOutputsConfig["dispatch_workflow"] = dispatchWorkflowConfig
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

var safeOutputsConfigGenLog = logger.New("workflow:safe_outputs_config_generation_helpers")

// ========================================
// Safe Output Configuration Generation Helpers
// ========================================
//
// This file contains helper functions to reduce duplication in safe output
// configuration generation. These helpers extract common patterns for:
// - Generating max value configs with defaults
// - Generating configs with allowed fields (labels, repos, etc.)
// - Generating configs with optional target fields
//
// The goal is to make generateSafeOutputsConfig more maintainable by
// extracting repetitive code patterns into reusable functions.

// resolveMaxForConfig resolves a templatable max *string to a config value.
// For expression strings (e.g. "${{ inputs.max }}"), the expression is stored
// as-is so GitHub Actions can resolve it at runtime.
// For literal numeric strings, the parsed integer is used.
// Falls back to defaultMax if max is nil or zero.
func resolveMaxForConfig(max *string, defaultMax int) any {
	if max != nil {
		v := *max
		if strings.HasPrefix(v, "${{") {
			return v // expression: evaluated at runtime by GitHub Actions
		}
		if n := templatableIntValue(max); n > 0 {
			return n
		}
	}
	return defaultMax
}

// generateMaxConfig creates a simple config map with just a max value
func generateMaxConfig(max *string, defaultMax int) map[string]any {
	config := make(map[string]any)
	config["max"] = resolveMaxForConfig(max, defaultMax)
	return config
}

// generateMaxWithAllowedLabelsConfig creates a config with max and optional allowed_labels
func generateMaxWithAllowedLabelsConfig(max *string, defaultMax int, allowedLabels []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowedLabels) > 0 {
		config["allowed_labels"] = allowedLabels
	}
	return config
}

// generateMaxWithTargetConfig creates a config with max and optional target field
func generateMaxWithTargetConfig(max *string, defaultMax int, target string) map[string]any {
	config := make(map[string]any)
	if target != "" {
		config["target"] = target
	}
	config["max"] = resolveMaxForConfig(max, defaultMax)
	return config
}

// generateMaxWithAllowedConfig creates a config with max and optional allowed list
func generateMaxWithAllowedConfig(max *string, defaultMax int, allowed []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowed) > 0 {
		config["allowed"] = allowed
	}
	return config
}

// generateMaxWithAllowedAndBlockedConfig creates a config with max, optional allowed list, and optional blocked list
func generateMaxWithAllowedAndBlockedConfig(max *string, defaultMax int, allowed []string, blocked []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowed) > 0 {
		config["allowed"] = allowed
	}
	if len(blocked) > 0 {
		config["blocked"] = blocked
	}
	return config
}

// generateMaxWithDiscussionFieldsConfig creates a config with discussion-specific filter fields
func generateMaxWithDiscussionFieldsConfig(max *string, defaultMax int, requiredCategory string, requiredLabels []string, requiredTitlePrefix string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if requiredCategory != "" {
		config["required_category"] = requiredCategory
	}
	if len(requiredLabels) > 0 {
		config["required_labels"] = requiredLabels
	}
	if requiredTitlePrefix != "" {
		config["required_title_prefix"] = requiredTitlePrefix
	}
	return config
}

// generateMaxWithReviewersConfig creates a config with max and optional reviewers list
func generateMaxWithReviewersConfig(max *string, defaultMax int, reviewers []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(reviewers) > 0 {
		config["reviewers"] = reviewers
	}
	return config
}

// generateAssignToAgentConfig creates a config with optional max, default_agent, target, and allowed
func generateAssignToAgentConfig(max *string, defaultMax int, defaultAgent string, target string, allowed []string) map[string]any {
	if safeOutputsConfigGenLog.Enabled() {
		safeOutputsConfigGenLog.Printf("Generating assign-to-agent config: max=%v, defaultMax=%d, defaultAgent=%s, target=%s, allowed_count=%d",
			max, defaultMax, defaultAgent, target, len(allowed))
	}
	config := make(map[string]any)
	config["max"] = resolveMaxForConfig(max, defaultMax)
	if defaultAgent != "" {
		config["default_agent"] = defaultAgent
	}
	if target != "" {
		config["target"] = target
	}
	if len(allowed) > 0 {
		config["allowed"] = allowed
	}
	return config
}

// generatePullRequestConfig creates a config with all pull request fields including target-repo,
// allowed_repos, base_branch, draft, reviewers, title_prefix, fallback_as_issue, and more.
func generatePullRequestConfig(prConfig *CreatePullRequestsConfig, defaultMax int) map[string]any {
	safeOutputsConfigGenLog.Printf("Generating pull request config: max=%v, allowEmpty=%v, autoMerge=%v, expires=%d, labels_count=%d, targetRepo=%s",
		prConfig.Max, prConfig.AllowEmpty, prConfig.AutoMerge, prConfig.Expires, len(prConfig.AllowedLabels), prConfig.TargetRepoSlug)

	additionalFields := make(map[string]any)
	if len(prConfig.AllowedLabels) > 0 {
		additionalFields["allowed_labels"] = prConfig.AllowedLabels
	}
	// Pass allow_empty flag to MCP server so it can skip patch generation
	if prConfig.AllowEmpty != nil && *prConfig.AllowEmpty == "true" {
		additionalFields["allow_empty"] = true
	}
	// Pass auto_merge flag to enable auto-merge for the pull request
	if prConfig.AutoMerge != nil && *prConfig.AutoMerge == "true" {
		additionalFields["auto_merge"] = true
	}
	// Pass expires to configure pull request expiration
	if prConfig.Expires > 0 {
		additionalFields["expires"] = prConfig.Expires
	}
	// Pass base_branch to configure the base branch for the pull request
	if prConfig.BaseBranch != "" {
		additionalFields["base_branch"] = prConfig.BaseBranch
	}
	// Pass draft flag to create the pull request as a draft
	if prConfig.Draft != nil && *prConfig.Draft == "true" {
		additionalFields["draft"] = true
	}
	// Pass reviewers to assign reviewers to the pull request
	if len(prConfig.Reviewers) > 0 {
		additionalFields["reviewers"] = prConfig.Reviewers
	}
	// Pass title_prefix to prepend to pull request titles
	if prConfig.TitlePrefix != "" {
		additionalFields["title_prefix"] = prConfig.TitlePrefix
	}
	// Pass fallback_as_issue if explicitly configured
	if prConfig.FallbackAsIssue != nil {
		additionalFields["fallback_as_issue"] = *prConfig.FallbackAsIssue
	}

	// Use generateTargetConfigWithRepos to include target-repo and allowed_repos
	targetConfig := SafeOutputTargetConfig{
		TargetRepoSlug: prConfig.TargetRepoSlug,
		AllowedRepos:   prConfig.AllowedRepos,
	}
	return generateTargetConfigWithRepos(targetConfig, prConfig.Max, defaultMax, additionalFields)
}

// generateHideCommentConfig creates a config with max and optional allowed_reasons
func generateHideCommentConfig(max *string, defaultMax int, allowedReasons []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowedReasons) > 0 {
		config["allowed_reasons"] = allowedReasons
	}
	return config
}

// generateTargetConfigWithRepos creates a config with target, target-repo, allowed_repos, and optional fields.
// Note on naming conventions:
// - "target-repo" uses hyphen to match frontmatter YAML format (key in config.json)
// - "allowed_repos" uses underscore to match JavaScript handler expectations (see repo_helpers.cjs)
// This inconsistency is intentional to maintain compatibility with existing handler code.
func generateTargetConfigWithRepos(targetConfig SafeOutputTargetConfig, max *string, defaultMax int, additionalFields map[string]any) map[string]any {
	config := generateMaxConfig(max, defaultMax)

	// Add target if specified
	if targetConfig.Target != "" {
		config["target"] = targetConfig.Target
	}

	// Add target-repo if specified (use hyphenated key for consistency with frontmatter)
	if targetConfig.TargetRepoSlug != "" {
		config["target-repo"] = targetConfig.TargetRepoSlug
	}

	// Add allowed_repos if specified (use underscore for consistency with handler code)
	if len(targetConfig.AllowedRepos) > 0 {
		config["allowed_repos"] = targetConfig.AllowedRepos
	}

	// Add any additional fields
	maps.Copy(config, additionalFields)

	return config
}

// ========================================
// Safe Output Configuration Helpers
// ========================================

var safeOutputReflectionLog = logger.New("workflow:safe_outputs_config_helpers_reflection")

// safeOutputFieldMapping maps struct field names to their tool names
var safeOutputFieldMapping = map[string]string{
	"CreateIssues":                    "create_issue",
	"CreateAgentSessions":             "create_agent_session",
	"CreateDiscussions":               "create_discussion",
	"UpdateDiscussions":               "update_discussion",
	"CloseDiscussions":                "close_discussion",
	"CloseIssues":                     "close_issue",
	"ClosePullRequests":               "close_pull_request",
	"AddComments":                     "add_comment",
	"CreatePullRequests":              "create_pull_request",
	"CreatePullRequestReviewComments": "create_pull_request_review_comment",
	"SubmitPullRequestReview":         "submit_pull_request_review",
	"ReplyToPullRequestReviewComment": "reply_to_pull_request_review_comment",
	"ResolvePullRequestReviewThread":  "resolve_pull_request_review_thread",
	"CreateCodeScanningAlerts":        "create_code_scanning_alert",
	"AddLabels":                       "add_labels",
	"RemoveLabels":                    "remove_labels",
	"AddReviewer":                     "add_reviewer",
	"AssignMilestone":                 "assign_milestone",
	"AssignToAgent":                   "assign_to_agent",
	"AssignToUser":                    "assign_to_user",
	"UpdateIssues":                    "update_issue",
	"UpdatePullRequests":              "update_pull_request",
	"PushToPullRequestBranch":         "push_to_pull_request_branch",
	"UploadAssets":                    "upload_asset",
	"UpdateRelease":                   "update_release",
	"UpdateProjects":                  "update_project",
	"CreateProjects":                  "create_project",
	"CreateProjectStatusUpdates":      "create_project_status_update",
	"LinkSubIssue":                    "link_sub_issue",
	"HideComment":                     "hide_comment",
	"DispatchWorkflow":                "dispatch_workflow",
	"MissingTool":                     "missing_tool",
	"NoOp":                            "noop",
	"MarkPullRequestAsReadyForReview": "mark_pull_request_as_ready_for_review",
}

// hasAnySafeOutputEnabled uses reflection to check if any safe output field is non-nil
func hasAnySafeOutputEnabled(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}

	safeOutputReflectionLog.Print("Checking if any safe outputs are enabled using reflection")

	// Check Jobs separately as it's a map
	if len(safeOutputs.Jobs) > 0 {
		safeOutputReflectionLog.Printf("Found %d custom jobs enabled", len(safeOutputs.Jobs))
		return true
	}

	// Use reflection to check all pointer fields
	val := reflect.ValueOf(safeOutputs).Elem()
	for fieldName := range safeOutputFieldMapping {
		field := val.FieldByName(fieldName)
		if field.IsValid() && !field.IsNil() {
			safeOutputReflectionLog.Printf("Found enabled safe output field: %s", fieldName)
			return true
		}
	}

	safeOutputReflectionLog.Print("No safe outputs enabled")
	return false
}

// getEnabledSafeOutputToolNamesReflection uses reflection to get enabled tool names
func getEnabledSafeOutputToolNamesReflection(safeOutputs *SafeOutputsConfig) []string {
	if safeOutputs == nil {
		return nil
	}

	safeOutputReflectionLog.Print("Getting enabled safe output tool names using reflection")
	var tools []string

	// Use reflection to check all pointer fields
	val := reflect.ValueOf(safeOutputs).Elem()
	for fieldName, toolName := range safeOutputFieldMapping {
		field := val.FieldByName(fieldName)
		if field.IsValid() && !field.IsNil() {
			tools = append(tools, toolName)
		}
	}

	// Add custom job tools
	for jobName := range safeOutputs.Jobs {
		tools = append(tools, jobName)
		safeOutputReflectionLog.Printf("Added custom job tool: %s", jobName)
	}

	// Sort tools to ensure deterministic compilation
	sort.Strings(tools)

	safeOutputReflectionLog.Printf("Found %d enabled safe output tools", len(tools))
	return tools
}

// formatSafeOutputsRunsOn formats the runs-on value from SafeOutputsConfig for job output
func (c *Compiler) formatSafeOutputsRunsOn(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || safeOutputs.RunsOn == "" {
		return "runs-on: " + constants.DefaultActivationJobRunnerImage
	}

	return "runs-on: " + safeOutputs.RunsOn
}

// formatDetectionRunsOn resolves the runner for the detection job using the following priority:
// 1. safe-outputs.detection.runs-on (detection-specific override)
// 2. agentRunsOn (the agent job's runner, passed by the caller)
func (c *Compiler) formatDetectionRunsOn(safeOutputs *SafeOutputsConfig, agentRunsOn string) string {
	if safeOutputs != nil && safeOutputs.ThreatDetection != nil && safeOutputs.ThreatDetection.RunsOn != "" {
		return "runs-on: " + safeOutputs.ThreatDetection.RunsOn
	}
	return agentRunsOn
}

// builtinSafeOutputFields contains the struct field names for the built-in safe output types
// that are excluded from the "non-builtin" check. These are: noop, missing-data, missing-tool.
var builtinSafeOutputFields = map[string]bool{
	"NoOp":        true,
	"MissingData": true,
	"MissingTool": true,
}

// nonBuiltinSafeOutputFieldNames is a pre-computed list of field names from safeOutputFieldMapping
// that are not builtins, used by hasNonBuiltinSafeOutputsEnabled to avoid repeated map iterations.
var nonBuiltinSafeOutputFieldNames = func() []string {
	var fields []string
	for fieldName := range safeOutputFieldMapping {
		if !builtinSafeOutputFields[fieldName] {
			fields = append(fields, fieldName)
		}
	}
	return fields
}()

// hasNonBuiltinSafeOutputsEnabled checks if any non-builtin safe outputs are configured.
// The builtin types (noop, missing-data, missing-tool) are excluded from this check
// because they are always auto-enabled and do not represent a meaningful output action.
func hasNonBuiltinSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}

	// Custom safe-jobs are always non-builtin
	if len(safeOutputs.Jobs) > 0 {
		return true
	}

	// Check non-builtin pointer fields using the pre-computed list
	val := reflect.ValueOf(safeOutputs).Elem()
	for _, fieldName := range nonBuiltinSafeOutputFieldNames {
		field := val.FieldByName(fieldName)
		if field.IsValid() && !field.IsNil() {
			return true
		}
	}

	return false
}

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	enabled := hasAnySafeOutputEnabled(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// applyDefaultCreateIssue injects a default create-issues safe output when safe-outputs is configured
// but has no non-builtin output types. The injected config uses the workflow ID as the label
// and [workflowID] as the title prefix. The AutoInjectedCreateIssue flag is set so the prompt
// generator can add a specific instruction for the agent.
func applyDefaultCreateIssue(workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}
	if hasNonBuiltinSafeOutputsEnabled(workflowData.SafeOutputs) {
		return
	}

	workflowID := workflowData.WorkflowID
	safeOutputsConfigLog.Printf("Auto-injecting create-issues for workflow %q (no non-builtin safe outputs configured)", workflowID)
	workflowData.SafeOutputs.CreateIssues = &CreateIssuesConfig{
		BaseSafeOutputConfig: BaseSafeOutputConfig{Max: defaultIntStr(1)},
		Labels:               []string{workflowID},
		TitlePrefix:          fmt.Sprintf("[%s]", workflowID),
	}
	workflowData.SafeOutputs.AutoInjectedCreateIssue = true
}

// GetEnabledSafeOutputToolNames returns a list of enabled safe output tool names.
// NOTE: Tool names should NOT be included in agent prompts. The agent should query
// the MCP server to discover available tools. This function is used for generating
// the tools.json file that the MCP server provides, and for diagnostic logging.
func GetEnabledSafeOutputToolNames(safeOutputs *SafeOutputsConfig) []string {
	tools := getEnabledSafeOutputToolNamesReflection(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Enabled safe output tools: %v", tools)
	}

	return tools
}

// usesPatchesAndCheckouts checks if the workflow uses safe outputs that require
// git patches and checkouts (create-pull-request or push-to-pull-request-branch)
func usesPatchesAndCheckouts(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil
}

// ========================================
// Safe Output Tools Generation
// ========================================

// checkAllEnabledToolsPresent verifies that every tool in enabledTools has a matching entry
// in filteredTools. This is a compiler error check: if a safe-output type is registered in
// Go code but its definition is missing from safe-output-tools.json, it will not appear in
// filteredTools and this function returns an error.
//
// Dispatch-workflow and custom-job tools are intentionally excluded from this check because
// they are generated dynamically and are never part of the static tools JSON.
func checkAllEnabledToolsPresent(enabledTools map[string]bool, filteredTools []map[string]any) error {
	presentTools := make(map[string]bool, len(filteredTools))
	for _, tool := range filteredTools {
		if name, ok := tool["name"].(string); ok {
			presentTools[name] = true
		}
	}

	var missingTools []string
	for toolName := range enabledTools {
		if !presentTools[toolName] {
			missingTools = append(missingTools, toolName)
		}
	}

	if len(missingTools) == 0 {
		return nil
	}

	sort.Strings(missingTools)
	return fmt.Errorf("compiler error: safe-output tool(s) %v are registered but missing from safe-output-tools.json; please report this issue to the developer", missingTools)
}

// generateFilteredToolsJSON filters the ALL_TOOLS array based on enabled safe outputs
// Returns a JSON string containing only the tools that are enabled in the workflow
func generateFilteredToolsJSON(data *WorkflowData, markdownPath string) (string, error) {
	if data.SafeOutputs == nil {
		return "[]", nil
	}

	safeOutputsConfigLog.Print("Generating filtered tools JSON for workflow")

	// Load the full tools JSON
	allToolsJSON := GetSafeOutputsToolsJSON()

	// Parse the JSON to get all tools
	var allTools []map[string]any
	if err := json.Unmarshal([]byte(allToolsJSON), &allTools); err != nil {
		safeOutputsConfigLog.Printf("Failed to parse safe outputs tools JSON: %v", err)
		return "", fmt.Errorf("failed to parse safe outputs tools JSON: %w", err)
	}

	// Create a set of enabled tool names
	enabledTools := make(map[string]bool)

	// Check which safe outputs are enabled and add their corresponding tool names
	if data.SafeOutputs.CreateIssues != nil {
		enabledTools["create_issue"] = true
	}
	if data.SafeOutputs.CreateAgentSessions != nil {
		enabledTools["create_agent_session"] = true
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		enabledTools["create_discussion"] = true
	}
	if data.SafeOutputs.UpdateDiscussions != nil {
		enabledTools["update_discussion"] = true
	}
	if data.SafeOutputs.CloseDiscussions != nil {
		enabledTools["close_discussion"] = true
	}
	if data.SafeOutputs.CloseIssues != nil {
		enabledTools["close_issue"] = true
	}
	if data.SafeOutputs.ClosePullRequests != nil {
		enabledTools["close_pull_request"] = true
	}
	if data.SafeOutputs.MarkPullRequestAsReadyForReview != nil {
		enabledTools["mark_pull_request_as_ready_for_review"] = true
	}
	if data.SafeOutputs.AddComments != nil {
		enabledTools["add_comment"] = true
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		enabledTools["create_pull_request"] = true
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		enabledTools["create_pull_request_review_comment"] = true
	}
	if data.SafeOutputs.SubmitPullRequestReview != nil {
		enabledTools["submit_pull_request_review"] = true
	}
	if data.SafeOutputs.ReplyToPullRequestReviewComment != nil {
		enabledTools["reply_to_pull_request_review_comment"] = true
	}
	if data.SafeOutputs.ResolvePullRequestReviewThread != nil {
		enabledTools["resolve_pull_request_review_thread"] = true
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		enabledTools["create_code_scanning_alert"] = true
	}
	if data.SafeOutputs.AutofixCodeScanningAlert != nil {
		enabledTools["autofix_code_scanning_alert"] = true
	}
	if data.SafeOutputs.AddLabels != nil {
		enabledTools["add_labels"] = true
	}
	if data.SafeOutputs.RemoveLabels != nil {
		enabledTools["remove_labels"] = true
	}
	if data.SafeOutputs.AddReviewer != nil {
		enabledTools["add_reviewer"] = true
	}
	if data.SafeOutputs.AssignMilestone != nil {
		enabledTools["assign_milestone"] = true
	}
	if data.SafeOutputs.AssignToAgent != nil {
		enabledTools["assign_to_agent"] = true
	}
	if data.SafeOutputs.AssignToUser != nil {
		enabledTools["assign_to_user"] = true
	}
	if data.SafeOutputs.UnassignFromUser != nil {
		enabledTools["unassign_from_user"] = true
	}
	if data.SafeOutputs.UpdateIssues != nil {
		enabledTools["update_issue"] = true
	}
	if data.SafeOutputs.UpdatePullRequests != nil {
		enabledTools["update_pull_request"] = true
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		enabledTools["push_to_pull_request_branch"] = true
	}
	if data.SafeOutputs.UploadAssets != nil {
		enabledTools["upload_asset"] = true
	}
	if data.SafeOutputs.MissingTool != nil {
		enabledTools["missing_tool"] = true
	}
	if data.SafeOutputs.MissingData != nil {
		enabledTools["missing_data"] = true
	}
	if data.SafeOutputs.UpdateRelease != nil {
		enabledTools["update_release"] = true
	}
	if data.SafeOutputs.NoOp != nil {
		enabledTools["noop"] = true
	}
	if data.SafeOutputs.LinkSubIssue != nil {
		enabledTools["link_sub_issue"] = true
	}
	if data.SafeOutputs.HideComment != nil {
		enabledTools["hide_comment"] = true
	}
	if data.SafeOutputs.SetIssueType != nil {
		enabledTools["set_issue_type"] = true
	}
	if data.SafeOutputs.UpdateProjects != nil {
		enabledTools["update_project"] = true
	}
	if data.SafeOutputs.CreateProjectStatusUpdates != nil {
		enabledTools["create_project_status_update"] = true
	}
	if data.SafeOutputs.CreateProjects != nil {
		enabledTools["create_project"] = true
	}
	// Note: dispatch_workflow tools are generated dynamically below, not from the static tools list

	// Add push_repo_memory tool if repo-memory is configured
	// This tool enables early size validation during the agent session
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		enabledTools["push_repo_memory"] = true
	}

	// Filter tools to only include enabled ones and enhance descriptions
	var filteredTools []map[string]any
	for _, tool := range allTools {
		toolName, ok := tool["name"].(string)
		if !ok {
			continue
		}
		if enabledTools[toolName] {
			// Create a copy of the tool to avoid modifying the original
			enhancedTool := make(map[string]any)
			maps.Copy(enhancedTool, tool)

			// Enhance the description with configuration details
			if description, ok := enhancedTool["description"].(string); ok {
				enhancedDescription := enhanceToolDescription(toolName, description, data.SafeOutputs)
				enhancedTool["description"] = enhancedDescription
			}

			// Add repo parameter to inputSchema if allowed-repos has entries
			addRepoParameterIfNeeded(enhancedTool, toolName, data.SafeOutputs)

			filteredTools = append(filteredTools, enhancedTool)
		}
	}

	// Verify all registered safe-outputs are present in the static tools JSON.
	// Dispatch-workflow and custom-job tools are excluded because they are generated dynamically.
	if err := checkAllEnabledToolsPresent(enabledTools, filteredTools); err != nil {
		return "", err
	}

	// Add custom job tools from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom job tools", len(data.SafeOutputs.Jobs))

		// Sort job names for deterministic output
		// This ensures compiled workflows have consistent tool ordering
		jobNames := make([]string, 0, len(data.SafeOutputs.Jobs))
		for jobName := range data.SafeOutputs.Jobs {
			jobNames = append(jobNames, jobName)
		}
		sort.Strings(jobNames)

		// Iterate over jobs in sorted order
		for _, jobName := range jobNames {
			jobConfig := data.SafeOutputs.Jobs[jobName]
			// Normalize job name to use underscores for consistency
			normalizedJobName := stringutil.NormalizeSafeOutputIdentifier(jobName)

			// Create the tool definition for this custom job
			customTool := generateCustomJobToolDefinition(normalizedJobName, jobConfig)
			filteredTools = append(filteredTools, customTool)
		}
	}

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Filtered %d tools from %d total tools (including %d custom jobs)", len(filteredTools), len(allTools), len(data.SafeOutputs.Jobs))
	}

	// Add dynamic dispatch_workflow tools
	if data.SafeOutputs.DispatchWorkflow != nil && len(data.SafeOutputs.DispatchWorkflow.Workflows) > 0 {
		safeOutputsConfigLog.Printf("Adding %d dispatch_workflow tools", len(data.SafeOutputs.DispatchWorkflow.Workflows))

		// Initialize WorkflowFiles map if not already initialized
		if data.SafeOutputs.DispatchWorkflow.WorkflowFiles == nil {
			data.SafeOutputs.DispatchWorkflow.WorkflowFiles = make(map[string]string)
		}

		for _, workflowName := range data.SafeOutputs.DispatchWorkflow.Workflows {
			// Find the workflow file in multiple locations
			fileResult, err := findWorkflowFile(workflowName, markdownPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
				// Continue with empty inputs
				tool := generateDispatchWorkflowTool(workflowName, make(map[string]any))
				filteredTools = append(filteredTools, tool)
				continue
			}

			// Determine which file to use - priority: .lock.yml > .yml
			var workflowPath string
			var extension string
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else {
				safeOutputsConfigLog.Printf("Warning: workflow file not found for %s (only .md exists, needs compilation)", workflowName)
				// Continue with empty inputs
				tool := generateDispatchWorkflowTool(workflowName, make(map[string]any))
				filteredTools = append(filteredTools, tool)
				continue
			}

			// Store the file extension for runtime use
			data.SafeOutputs.DispatchWorkflow.WorkflowFiles[workflowName] = extension

			// Extract workflow_dispatch inputs
			workflowInputs, err := extractWorkflowDispatchInputs(workflowPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, err)
				// Continue with empty inputs
				workflowInputs = make(map[string]any)
			}

			// Generate tool schema
			tool := generateDispatchWorkflowTool(workflowName, workflowInputs)
			filteredTools = append(filteredTools, tool)
		}
	}

	// Marshal the filtered tools back to JSON with indentation for better readability
	// and to reduce merge conflicts in generated lockfiles
	filteredJSON, err := json.MarshalIndent(filteredTools, "", "  ")
	if err != nil {
		safeOutputsConfigLog.Printf("Failed to marshal filtered tools: %v", err)
		return "", fmt.Errorf("failed to marshal filtered tools: %w", err)
	}

	safeOutputsConfigLog.Printf("Successfully generated filtered tools JSON with %d tools", len(filteredTools))
	return string(filteredJSON), nil
}

// addRepoParameterIfNeeded adds a "repo" parameter to the tool's inputSchema
// if the safe output configuration has allowed-repos entries
func addRepoParameterIfNeeded(tool map[string]any, toolName string, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	// Determine if this tool should have a repo parameter based on allowed-repos configuration
	var hasAllowedRepos bool
	var targetRepoSlug string

	switch toolName {
	case "create_issue":
		if config := safeOutputs.CreateIssues; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_discussion":
		if config := safeOutputs.CreateDiscussions; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "add_comment":
		if config := safeOutputs.AddComments; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_pull_request":
		if config := safeOutputs.CreatePullRequests; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_pull_request_review_comment":
		if config := safeOutputs.CreatePullRequestReviewComments; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "reply_to_pull_request_review_comment":
		if config := safeOutputs.ReplyToPullRequestReviewComment; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_agent_session":
		if config := safeOutputs.CreateAgentSessions; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "close_issue", "update_issue":
		if config := safeOutputs.CloseIssues; config != nil && toolName == "close_issue" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		} else if config := safeOutputs.UpdateIssues; config != nil && toolName == "update_issue" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "close_discussion", "update_discussion":
		if config := safeOutputs.CloseDiscussions; config != nil && toolName == "close_discussion" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		} else if config := safeOutputs.UpdateDiscussions; config != nil && toolName == "update_discussion" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "close_pull_request", "update_pull_request":
		if config := safeOutputs.ClosePullRequests; config != nil && toolName == "close_pull_request" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		} else if config := safeOutputs.UpdatePullRequests; config != nil && toolName == "update_pull_request" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "add_labels", "remove_labels", "hide_comment", "link_sub_issue", "mark_pull_request_as_ready_for_review",
		"add_reviewer", "assign_milestone", "assign_to_agent", "assign_to_user", "unassign_from_user",
		"set_issue_type":
		// These use SafeOutputTargetConfig - check the appropriate config
		switch toolName {
		case "add_labels":
			if config := safeOutputs.AddLabels; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "remove_labels":
			if config := safeOutputs.RemoveLabels; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "hide_comment":
			if config := safeOutputs.HideComment; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "link_sub_issue":
			if config := safeOutputs.LinkSubIssue; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "mark_pull_request_as_ready_for_review":
			if config := safeOutputs.MarkPullRequestAsReadyForReview; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "add_reviewer":
			if config := safeOutputs.AddReviewer; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "assign_milestone":
			if config := safeOutputs.AssignMilestone; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "assign_to_agent":
			if config := safeOutputs.AssignToAgent; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "assign_to_user":
			if config := safeOutputs.AssignToUser; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "unassign_from_user":
			if config := safeOutputs.UnassignFromUser; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "set_issue_type":
			if config := safeOutputs.SetIssueType; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		}
	}

	// Only add repo parameter if allowed-repos has entries or target-repo is wildcard ("*")
	if !hasAllowedRepos && targetRepoSlug != "*" {
		return
	}

	// Get the inputSchema
	inputSchema, ok := tool["inputSchema"].(map[string]any)
	if !ok {
		return
	}

	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		return
	}

	// Build repo parameter description
	var repoDescription string
	if targetRepoSlug == "*" {
		repoDescription = "Target repository for this operation in 'owner/repo' format. Any repository can be targeted."
	} else if targetRepoSlug != "" {
		repoDescription = fmt.Sprintf("Target repository for this operation in 'owner/repo' format. Default is %q. Must be the target-repo or in the allowed-repos list.", targetRepoSlug)
	} else {
		repoDescription = "Target repository for this operation in 'owner/repo' format. Must be the target-repo or in the allowed-repos list."
	}

	// Add repo parameter to properties
	properties["repo"] = map[string]any{
		"type":        "string",
		"description": repoDescription,
	}

	safeOutputsConfigLog.Printf("Added repo parameter to tool: %s (has allowed-repos or wildcard target-repo)", toolName)
}

// generateDispatchWorkflowTool generates an MCP tool definition for a specific workflow
// The tool will be named after the workflow and accept the workflow's defined inputs
func generateDispatchWorkflowTool(workflowName string, workflowInputs map[string]any) map[string]any {
	// Normalize workflow name to use underscores for tool name
	toolName := stringutil.NormalizeSafeOutputIdentifier(workflowName)

	// Build the description
	description := fmt.Sprintf("Dispatch the '%s' workflow with workflow_dispatch trigger. This workflow must support workflow_dispatch and be in .github/workflows/ directory in the same repository.", workflowName)

	// Build input schema properties
	properties := make(map[string]any)
	required := []string{} // No required fields by default

	// Convert GitHub Actions workflow_dispatch inputs to MCP tool schema
	for inputName, inputDef := range workflowInputs {
		inputDefMap, ok := inputDef.(map[string]any)
		if !ok {
			continue
		}

		// Extract input properties
		inputType := "string" // Default type
		inputDescription := fmt.Sprintf("Input parameter '%s' for workflow %s", inputName, workflowName)
		inputRequired := false

		if desc, ok := inputDefMap["description"].(string); ok && desc != "" {
			inputDescription = desc
		}

		if req, ok := inputDefMap["required"].(bool); ok {
			inputRequired = req
		}

		// GitHub Actions workflow_dispatch supports: string, number, boolean, choice, environment
		// Map these to JSON schema types
		if typeStr, ok := inputDefMap["type"].(string); ok {
			switch typeStr {
			case "number":
				inputType = "number"
			case "boolean":
				inputType = "boolean"
			case "choice":
				inputType = "string"
				// Add enum if options are provided
				if options, ok := inputDefMap["options"].([]any); ok && len(options) > 0 {
					properties[inputName] = map[string]any{
						"type":        inputType,
						"description": inputDescription,
						"enum":        options,
					}
					if inputRequired {
						required = append(required, inputName)
					}
					continue
				}
			case "environment":
				inputType = "string"
			}
		}

		properties[inputName] = map[string]any{
			"type":        inputType,
			"description": inputDescription,
		}

		// Add default value if provided
		if defaultVal, ok := inputDefMap["default"]; ok {
			properties[inputName].(map[string]any)["default"] = defaultVal
		}

		if inputRequired {
			required = append(required, inputName)
		}
	}

	// Add internal workflow_name parameter (hidden from description but used internally)
	// This will be injected by the safe output handler

	// Build the complete tool definition
	tool := map[string]any{
		"name":           toolName,
		"description":    description,
		"_workflow_name": workflowName, // Internal metadata for handler routing
		"inputSchema": map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		},
	}

	if len(required) > 0 {
		sort.Strings(required)
		tool["inputSchema"].(map[string]any)["required"] = required
	}

	return tool
}
