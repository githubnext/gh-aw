package workflow

import (
	"encoding/json"
	"fmt"
)

// ========================================
// Safe Output Configuration Generation
// ========================================

func generateSafeOutputsConfig(data *WorkflowData) string {
	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs == nil {
		return ""
	}
	safeOutputsConfigLog.Print("Generating safe outputs configuration for workflow")
	// Create a simplified config object for validation
	safeOutputsConfig := make(map[string]any)

	// Handle safe-outputs configuration if present
	if data.SafeOutputs != nil {
		if data.SafeOutputs.CreateIssues != nil {
			issueConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CreateIssues.Max > 0 {
				maxValue = data.SafeOutputs.CreateIssues.Max
			}
			issueConfig["max"] = maxValue
			if len(data.SafeOutputs.CreateIssues.AllowedLabels) > 0 {
				issueConfig["allowed_labels"] = data.SafeOutputs.CreateIssues.AllowedLabels
			}
			safeOutputsConfig["create_issue"] = issueConfig
		}
		if data.SafeOutputs.CreateAgentTasks != nil {
			agentTaskConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CreateAgentTasks.Max > 0 {
				maxValue = data.SafeOutputs.CreateAgentTasks.Max
			}
			agentTaskConfig["max"] = maxValue
			safeOutputsConfig["create_agent_task"] = agentTaskConfig
		}
		if data.SafeOutputs.AddComments != nil {
			commentConfig := map[string]any{}
			if data.SafeOutputs.AddComments.Target != "" {
				commentConfig["target"] = data.SafeOutputs.AddComments.Target
			}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.AddComments.Max > 0 {
				maxValue = data.SafeOutputs.AddComments.Max
			}
			commentConfig["max"] = maxValue
			safeOutputsConfig["add_comment"] = commentConfig
		}
		if data.SafeOutputs.CreateDiscussions != nil {
			discussionConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CreateDiscussions.Max > 0 {
				maxValue = data.SafeOutputs.CreateDiscussions.Max
			}
			discussionConfig["max"] = maxValue
			if len(data.SafeOutputs.CreateDiscussions.AllowedLabels) > 0 {
				discussionConfig["allowed_labels"] = data.SafeOutputs.CreateDiscussions.AllowedLabels
			}
			safeOutputsConfig["create_discussion"] = discussionConfig
		}
		if data.SafeOutputs.CloseDiscussions != nil {
			closeDiscussionConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CloseDiscussions.Max > 0 {
				maxValue = data.SafeOutputs.CloseDiscussions.Max
			}
			closeDiscussionConfig["max"] = maxValue
			if data.SafeOutputs.CloseDiscussions.RequiredCategory != "" {
				closeDiscussionConfig["required_category"] = data.SafeOutputs.CloseDiscussions.RequiredCategory
			}
			if len(data.SafeOutputs.CloseDiscussions.RequiredLabels) > 0 {
				closeDiscussionConfig["required_labels"] = data.SafeOutputs.CloseDiscussions.RequiredLabels
			}
			if data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix != "" {
				closeDiscussionConfig["required_title_prefix"] = data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix
			}
			safeOutputsConfig["close_discussion"] = closeDiscussionConfig
		}
		if data.SafeOutputs.CloseIssues != nil {
			closeIssueConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CloseIssues.Max > 0 {
				maxValue = data.SafeOutputs.CloseIssues.Max
			}
			closeIssueConfig["max"] = maxValue
			if len(data.SafeOutputs.CloseIssues.RequiredLabels) > 0 {
				closeIssueConfig["required_labels"] = data.SafeOutputs.CloseIssues.RequiredLabels
			}
			if data.SafeOutputs.CloseIssues.RequiredTitlePrefix != "" {
				closeIssueConfig["required_title_prefix"] = data.SafeOutputs.CloseIssues.RequiredTitlePrefix
			}
			safeOutputsConfig["close_issue"] = closeIssueConfig
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			prConfig := map[string]any{}
			// Note: max is always 1 for pull requests, not configurable
			if len(data.SafeOutputs.CreatePullRequests.AllowedLabels) > 0 {
				prConfig["allowed_labels"] = data.SafeOutputs.CreatePullRequests.AllowedLabels
			}
			// Pass allow_empty flag to MCP server so it can skip patch generation
			if data.SafeOutputs.CreatePullRequests.AllowEmpty {
				prConfig["allow_empty"] = true
			}
			safeOutputsConfig["create_pull_request"] = prConfig
		}
		if data.SafeOutputs.CreatePullRequestReviewComments != nil {
			prReviewCommentConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 10 // default
			if data.SafeOutputs.CreatePullRequestReviewComments.Max > 0 {
				maxValue = data.SafeOutputs.CreatePullRequestReviewComments.Max
			}
			prReviewCommentConfig["max"] = maxValue
			safeOutputsConfig["create_pull_request_review_comment"] = prReviewCommentConfig
		}
		if data.SafeOutputs.CreateCodeScanningAlerts != nil {
			// Security reports typically have unlimited max, but check if configured
			securityReportConfig := map[string]any{}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
				maxValue = data.SafeOutputs.CreateCodeScanningAlerts.Max
			}
			securityReportConfig["max"] = maxValue
			safeOutputsConfig["create_code_scanning_alert"] = securityReportConfig
		}
		if data.SafeOutputs.AddLabels != nil {
			labelConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 3 // default
			if data.SafeOutputs.AddLabels.Max > 0 {
				maxValue = data.SafeOutputs.AddLabels.Max
			}
			labelConfig["max"] = maxValue
			if len(data.SafeOutputs.AddLabels.Allowed) > 0 {
				labelConfig["allowed"] = data.SafeOutputs.AddLabels.Allowed
			}
			safeOutputsConfig["add_labels"] = labelConfig
		}
		if data.SafeOutputs.AddReviewer != nil {
			reviewerConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 3 // default
			if data.SafeOutputs.AddReviewer.Max > 0 {
				maxValue = data.SafeOutputs.AddReviewer.Max
			}
			reviewerConfig["max"] = maxValue
			if len(data.SafeOutputs.AddReviewer.Reviewers) > 0 {
				reviewerConfig["reviewers"] = data.SafeOutputs.AddReviewer.Reviewers
			}
			safeOutputsConfig["add_reviewer"] = reviewerConfig
		}
		if data.SafeOutputs.AssignMilestone != nil {
			assignMilestoneConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.AssignMilestone.Max > 0 {
				maxValue = data.SafeOutputs.AssignMilestone.Max
			}
			assignMilestoneConfig["max"] = maxValue
			if len(data.SafeOutputs.AssignMilestone.Allowed) > 0 {
				assignMilestoneConfig["allowed"] = data.SafeOutputs.AssignMilestone.Allowed
			}
			safeOutputsConfig["assign_milestone"] = assignMilestoneConfig
		}
		if data.SafeOutputs.AssignToAgent != nil {
			assignToAgentConfig := map[string]any{}
			if data.SafeOutputs.AssignToAgent.Max > 0 {
				assignToAgentConfig["max"] = data.SafeOutputs.AssignToAgent.Max
			}
			if data.SafeOutputs.AssignToAgent.DefaultAgent != "" {
				assignToAgentConfig["default_agent"] = data.SafeOutputs.AssignToAgent.DefaultAgent
			}
			safeOutputsConfig["assign_to_agent"] = assignToAgentConfig
		}
		if data.SafeOutputs.AssignToUser != nil {
			assignToUserConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.AssignToUser.Max > 0 {
				maxValue = data.SafeOutputs.AssignToUser.Max
			}
			assignToUserConfig["max"] = maxValue
			if len(data.SafeOutputs.AssignToUser.Allowed) > 0 {
				assignToUserConfig["allowed"] = data.SafeOutputs.AssignToUser.Allowed
			}
			safeOutputsConfig["assign_to_user"] = assignToUserConfig
		}
		if data.SafeOutputs.UpdateIssues != nil {
			updateConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdateIssues.Max > 0 {
				maxValue = data.SafeOutputs.UpdateIssues.Max
			}
			updateConfig["max"] = maxValue
			safeOutputsConfig["update_issue"] = updateConfig
		}
		if data.SafeOutputs.UpdateDiscussions != nil {
			updateDiscussionConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdateDiscussions.Max > 0 {
				maxValue = data.SafeOutputs.UpdateDiscussions.Max
			}
			updateDiscussionConfig["max"] = maxValue
			if len(data.SafeOutputs.UpdateDiscussions.AllowedLabels) > 0 {
				updateDiscussionConfig["allowed_labels"] = data.SafeOutputs.UpdateDiscussions.AllowedLabels
			}
			safeOutputsConfig["update_discussion"] = updateDiscussionConfig
		}
		if data.SafeOutputs.UpdatePullRequests != nil {
			updatePRConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdatePullRequests.Max > 0 {
				maxValue = data.SafeOutputs.UpdatePullRequests.Max
			}
			updatePRConfig["max"] = maxValue
			safeOutputsConfig["update_pull_request"] = updatePRConfig
		}
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			pushToBranchConfig := map[string]any{}
			if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
				pushToBranchConfig["target"] = data.SafeOutputs.PushToPullRequestBranch.Target
			}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.PushToPullRequestBranch.Max > 0 {
				maxValue = data.SafeOutputs.PushToPullRequestBranch.Max
			}
			pushToBranchConfig["max"] = maxValue
			safeOutputsConfig["push_to_pull_request_branch"] = pushToBranchConfig
		}
		if data.SafeOutputs.UploadAssets != nil {
			uploadConfig := map[string]any{}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.UploadAssets.Max > 0 {
				maxValue = data.SafeOutputs.UploadAssets.Max
			}
			uploadConfig["max"] = maxValue
			safeOutputsConfig["upload_asset"] = uploadConfig
		}
		if data.SafeOutputs.MissingTool != nil {
			missingToolConfig := map[string]any{}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.MissingTool.Max > 0 {
				maxValue = data.SafeOutputs.MissingTool.Max
			}
			missingToolConfig["max"] = maxValue
			safeOutputsConfig["missing_tool"] = missingToolConfig
		}
		if data.SafeOutputs.UpdateProjects != nil {
			updateProjectConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 10 // default
			if data.SafeOutputs.UpdateProjects.Max > 0 {
				maxValue = data.SafeOutputs.UpdateProjects.Max
			}
			updateProjectConfig["max"] = maxValue
			safeOutputsConfig["update_project"] = updateProjectConfig
		}
		if data.SafeOutputs.UpdateRelease != nil {
			updateReleaseConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdateRelease.Max > 0 {
				maxValue = data.SafeOutputs.UpdateRelease.Max
			}
			updateReleaseConfig["max"] = maxValue
			safeOutputsConfig["update_release"] = updateReleaseConfig
		}
		if data.SafeOutputs.LinkSubIssue != nil {
			linkSubIssueConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 5 // default
			if data.SafeOutputs.LinkSubIssue.Max > 0 {
				maxValue = data.SafeOutputs.LinkSubIssue.Max
			}
			linkSubIssueConfig["max"] = maxValue
			safeOutputsConfig["link_sub_issue"] = linkSubIssueConfig
		}
		if data.SafeOutputs.NoOp != nil {
			noopConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.NoOp.Max > 0 {
				maxValue = data.SafeOutputs.NoOp.Max
			}
			noopConfig["max"] = maxValue
			safeOutputsConfig["noop"] = noopConfig
		}
		if data.SafeOutputs.HideComment != nil {
			hideCommentConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 5 // default
			if data.SafeOutputs.HideComment.Max > 0 {
				maxValue = data.SafeOutputs.HideComment.Max
			}
			hideCommentConfig["max"] = maxValue
			if len(data.SafeOutputs.HideComment.AllowedReasons) > 0 {
				hideCommentConfig["allowed_reasons"] = data.SafeOutputs.HideComment.AllowedReasons
			}
			safeOutputsConfig["hide_comment"] = hideCommentConfig
		}
	}

	// Add safe-jobs configuration from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		for jobName, jobConfig := range data.SafeOutputs.Jobs {
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

	configJSON, _ := json.Marshal(safeOutputsConfig)
	return string(configJSON)
}

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
		inputSchema["required"] = requiredFields
	}

	// Prevent additional properties to maintain schema strictness
	inputSchema["additionalProperties"] = false

	tool["inputSchema"] = inputSchema

	safeOutputsConfigLog.Printf("Generated tool definition for %s with %d inputs, %d required",
		jobName, len(jobConfig.Inputs), len(requiredFields))

	return tool
}

// generateFilteredToolsJSON filters the ALL_TOOLS array based on enabled safe outputs
// Returns a JSON string containing only the tools that are enabled in the workflow
func generateFilteredToolsJSON(data *WorkflowData) (string, error) {
	if data.SafeOutputs == nil {
		return "[]", nil
	}

	safeOutputsConfigLog.Print("Generating filtered tools JSON for workflow")

	// Load the full tools JSON
	allToolsJSON := GetSafeOutputsToolsJSON()

	// Parse the JSON to get all tools
	var allTools []map[string]any
	if err := json.Unmarshal([]byte(allToolsJSON), &allTools); err != nil {
		return "", fmt.Errorf("failed to parse safe outputs tools JSON: %w", err)
	}

	// Create a set of enabled tool names
	enabledTools := make(map[string]bool)

	// Check which safe outputs are enabled and add their corresponding tool names
	if data.SafeOutputs.CreateIssues != nil {
		enabledTools["create_issue"] = true
	}
	if data.SafeOutputs.CreateAgentTasks != nil {
		enabledTools["create_agent_task"] = true
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
	if data.SafeOutputs.AddComments != nil {
		enabledTools["add_comment"] = true
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		enabledTools["create_pull_request"] = true
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		enabledTools["create_pull_request_review_comment"] = true
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		enabledTools["create_code_scanning_alert"] = true
	}
	if data.SafeOutputs.AddLabels != nil {
		enabledTools["add_labels"] = true
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
	if data.SafeOutputs.UpdateProjects != nil {
		enabledTools["update_project"] = true
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
			for k, v := range tool {
				enhancedTool[k] = v
			}

			// Enhance the description with configuration details
			if description, ok := enhancedTool["description"].(string); ok {
				enhancedDescription := enhanceToolDescription(toolName, description, data.SafeOutputs)
				enhancedTool["description"] = enhancedDescription
			}

			filteredTools = append(filteredTools, enhancedTool)
		}
	}

	// Add custom job tools from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom job tools", len(data.SafeOutputs.Jobs))
		for jobName, jobConfig := range data.SafeOutputs.Jobs {
			// Normalize job name to use underscores for consistency
			normalizedJobName := normalizeSafeOutputIdentifier(jobName)

			// Create the tool definition for this custom job
			customTool := generateCustomJobToolDefinition(normalizedJobName, jobConfig)
			filteredTools = append(filteredTools, customTool)
		}
	}

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Filtered %d tools from %d total tools (including %d custom jobs)", len(filteredTools), len(allTools), len(data.SafeOutputs.Jobs))
	}

	// Marshal the filtered tools back to JSON with indentation for better readability
	// and to reduce merge conflicts in generated lockfiles
	filteredJSON, err := json.MarshalIndent(filteredTools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered tools: %w", err)
	}

	return string(filteredJSON), nil
}
