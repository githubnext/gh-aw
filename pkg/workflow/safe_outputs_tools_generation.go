package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Tools Generation
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
		"add_reviewer", "assign_milestone", "assign_to_agent", "assign_to_user", "unassign_from_user":
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
		tool["inputSchema"].(map[string]any)["required"] = required
	}

	return tool
}
