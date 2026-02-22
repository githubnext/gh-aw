package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
				data.SafeOutputs.CreatePullRequests.Max,
				1, // default max
				data.SafeOutputs.CreatePullRequests.AllowedLabels,
				data.SafeOutputs.CreatePullRequests.AllowEmpty,
				data.SafeOutputs.CreatePullRequests.AutoMerge,
				data.SafeOutputs.CreatePullRequests.Expires,
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

	configJSON, _ := json.Marshal(safeOutputsConfig)
	safeOutputsConfigLog.Printf("Safe outputs config generation complete: %d tool types configured", len(safeOutputsConfig))
	return string(configJSON)
}
