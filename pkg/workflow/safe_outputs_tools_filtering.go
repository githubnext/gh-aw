package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Tools Filtering
// ========================================
//
// This file handles tool enumeration and filtering: it takes the full set of
// safe-output tool definitions (from safe-output-tools.json) and produces a
// filtered subset containing only those tools enabled by the workflow's
// SafeOutputsConfig. Dynamic tools (dispatch-workflow, custom jobs) are also
// generated here.
//
// There are two generation strategies:
//
//  1. generateFilteredToolsJSON: Legacy approach that loads the embedded
//     safe_outputs_tools.json, filters and enhances it in Go, then returns the
//     full JSON to be inlined directly into the compiled workflow YAML as a
//     heredoc. Used by tests and kept for backward compatibility.
//
//  2. generateToolsMetaJSON: New approach that generates a small "meta" JSON
//     (description suffixes, repo params, dynamic tools) to be written at
//     compile time. At runtime, generate_safe_outputs_tools.cjs reads the
//     source safe_outputs_tools.json from the actions folder, applies the meta
//     overrides, and writes the final tools.json—avoiding inlining the entire
//     file into the compiled workflow YAML.

// generateFilteredToolsJSON filters the ALL_TOOLS array based on enabled safe outputs.
// Returns a JSON string containing only the tools that are enabled in the workflow.
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

	// Add custom script tools from SafeOutputs.Scripts
	if len(data.SafeOutputs.Scripts) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom script tools", len(data.SafeOutputs.Scripts))

		// Sort script names for deterministic output
		scriptNames := make([]string, 0, len(data.SafeOutputs.Scripts))
		for scriptName := range data.SafeOutputs.Scripts {
			scriptNames = append(scriptNames, scriptName)
		}
		sort.Strings(scriptNames)

		for _, scriptName := range scriptNames {
			scriptConfig := data.SafeOutputs.Scripts[scriptName]
			normalizedScriptName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
			customTool := generateCustomScriptToolDefinition(normalizedScriptName, scriptConfig)
			filteredTools = append(filteredTools, customTool)
		}
	}

	// Add custom action tools from SafeOutputs.Actions.
	// Action resolution (action.yml fetch + pinning) is done once in buildJobs before
	// generateToolsMetaJSON/generateFilteredToolsJSON are called, so Inputs and
	// ActionDescription are already populated here.
	if len(data.SafeOutputs.Actions) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom action tools", len(data.SafeOutputs.Actions))

		actionNames := make([]string, 0, len(data.SafeOutputs.Actions))
		for actionName := range data.SafeOutputs.Actions {
			actionNames = append(actionNames, actionName)
		}
		sort.Strings(actionNames)

		for _, actionName := range actionNames {
			actionConfig := data.SafeOutputs.Actions[actionName]
			customTool := generateActionToolDefinition(actionName, actionConfig)
			filteredTools = append(filteredTools, customTool)
		}
	}

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Filtered %d tools from %d total tools (including %d custom jobs, %d custom scripts, %d custom actions)", len(filteredTools), len(allTools), len(data.SafeOutputs.Jobs), len(data.SafeOutputs.Scripts), len(data.SafeOutputs.Actions))
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

			// Determine which file to use - priority: .lock.yml > .yml > .md (batch target)
			var workflowPath string
			var extension string
			var useMD bool
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else if fileResult.mdExists {
				// .md-only: the workflow is a same-batch compilation target that will produce a .lock.yml
				workflowPath = fileResult.mdPath
				extension = ".lock.yml"
				useMD = true
			} else {
				safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
				// Continue with empty inputs
				tool := generateDispatchWorkflowTool(workflowName, make(map[string]any))
				filteredTools = append(filteredTools, tool)
				continue
			}

			// Store the file extension for runtime use
			data.SafeOutputs.DispatchWorkflow.WorkflowFiles[workflowName] = extension

			// Extract workflow_dispatch inputs
			var workflowInputs map[string]any
			var inputsErr error
			if useMD {
				workflowInputs, inputsErr = extractMDWorkflowDispatchInputs(workflowPath)
			} else {
				workflowInputs, inputsErr = extractWorkflowDispatchInputs(workflowPath)
			}
			if inputsErr != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
				// Continue with empty inputs
				workflowInputs = make(map[string]any)
			}

			// Generate tool schema
			tool := generateDispatchWorkflowTool(workflowName, workflowInputs)
			filteredTools = append(filteredTools, tool)
		}
	}

	// Add dynamic dispatch_repository tools
	if data.SafeOutputs.DispatchRepository != nil && len(data.SafeOutputs.DispatchRepository.Tools) > 0 {
		safeOutputsConfigLog.Printf("Adding %d dispatch_repository tools", len(data.SafeOutputs.DispatchRepository.Tools))

		// Sort tool keys for deterministic output
		toolKeys := make([]string, 0, len(data.SafeOutputs.DispatchRepository.Tools))
		for toolKey := range data.SafeOutputs.DispatchRepository.Tools {
			toolKeys = append(toolKeys, toolKey)
		}
		sort.Strings(toolKeys)

		for _, toolKey := range toolKeys {
			toolConfig := data.SafeOutputs.DispatchRepository.Tools[toolKey]
			tool := generateDispatchRepositoryTool(toolKey, toolConfig)
			filteredTools = append(filteredTools, tool)
		}
	}

	// Add dynamic call_workflow tools
	if data.SafeOutputs.CallWorkflow != nil && len(data.SafeOutputs.CallWorkflow.Workflows) > 0 {
		safeOutputsConfigLog.Printf("Adding %d call_workflow tools", len(data.SafeOutputs.CallWorkflow.Workflows))

		// Initialize WorkflowFiles map if not already initialized
		if data.SafeOutputs.CallWorkflow.WorkflowFiles == nil {
			data.SafeOutputs.CallWorkflow.WorkflowFiles = make(map[string]string)
		}

		for _, workflowName := range data.SafeOutputs.CallWorkflow.Workflows {
			// Find the workflow file in multiple locations
			fileResult, err := findWorkflowFile(workflowName, markdownPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
				tool := generateCallWorkflowTool(workflowName, make(map[string]any))
				filteredTools = append(filteredTools, tool)
				continue
			}

			// Determine which file to use - priority: .lock.yml > .yml > .md (batch target)
			var workflowPath string
			var extension string
			var useMD bool
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else if fileResult.mdExists {
				workflowPath = fileResult.mdPath
				extension = ".lock.yml"
				useMD = true
			} else {
				safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
				tool := generateCallWorkflowTool(workflowName, make(map[string]any))
				filteredTools = append(filteredTools, tool)
				continue
			}

			// Store the relative path for compile-time use
			relativePath := fmt.Sprintf("./.github/workflows/%s%s", workflowName, extension)
			data.SafeOutputs.CallWorkflow.WorkflowFiles[workflowName] = relativePath

			// Extract workflow_call inputs
			var workflowInputs map[string]any
			var inputsErr error
			if useMD {
				workflowInputs, inputsErr = extractMDWorkflowCallInputs(workflowPath)
			} else {
				workflowInputs, inputsErr = extractWorkflowCallInputs(workflowPath)
			}
			if inputsErr != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
				workflowInputs = make(map[string]any)
			}

			// Generate tool schema
			tool := generateCallWorkflowTool(workflowName, workflowInputs)
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

// computeEnabledToolNames returns the set of predefined tool names that are enabled
// by the workflow's SafeOutputsConfig. Dynamic tools (dispatch-workflow, custom jobs,
// call-workflow) are excluded because they are generated separately.
func computeEnabledToolNames(data *WorkflowData) map[string]bool {
	enabledTools := make(map[string]bool)
	if data.SafeOutputs == nil {
		return enabledTools
	}

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

	// Add push_repo_memory tool if repo-memory is configured
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		enabledTools["push_repo_memory"] = true
	}

	return enabledTools
}

// computeRepoParamForTool returns the "repo" input parameter definition that should
// be added to a tool's inputSchema, or nil if no repo parameter is needed.
// This mirrors the logic in addRepoParameterIfNeeded but returns the param instead
// of modifying a tool in place, making it usable for generateToolsMetaJSON.
func computeRepoParamForTool(toolName string, safeOutputs *SafeOutputsConfig) map[string]any {
	// Reuse addRepoParameterIfNeeded by passing a scratch tool with an empty inputSchema.
	scratch := map[string]any{
		"name":        toolName,
		"inputSchema": map[string]any{"properties": map[string]any{}},
	}
	addRepoParameterIfNeeded(scratch, toolName, safeOutputs)

	inputSchema, ok := scratch["inputSchema"].(map[string]any)
	if !ok {
		return nil
	}
	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	repoProp, ok := properties["repo"].(map[string]any)
	if !ok {
		return nil
	}
	return repoProp
}

// generateDynamicTools generates MCP tool definitions for dynamic tools:
// custom safe-jobs, dispatch_workflow targets, and call_workflow targets.
// These tools are not in safe_outputs_tools.json and must be generated from
// the workflow configuration at compile time.
func generateDynamicTools(data *WorkflowData, markdownPath string) ([]map[string]any, error) {
	var dynamicTools []map[string]any

	// Add custom job tools from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom job tools", len(data.SafeOutputs.Jobs))

		// Sort job names for deterministic output
		jobNames := make([]string, 0, len(data.SafeOutputs.Jobs))
		for jobName := range data.SafeOutputs.Jobs {
			jobNames = append(jobNames, jobName)
		}
		sort.Strings(jobNames)

		for _, jobName := range jobNames {
			jobConfig := data.SafeOutputs.Jobs[jobName]
			normalizedJobName := stringutil.NormalizeSafeOutputIdentifier(jobName)
			customTool := generateCustomJobToolDefinition(normalizedJobName, jobConfig)
			dynamicTools = append(dynamicTools, customTool)
		}
	}

	// Add custom script tools from SafeOutputs.Scripts
	if len(data.SafeOutputs.Scripts) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom script tools to dynamic tools", len(data.SafeOutputs.Scripts))

		scriptNames := make([]string, 0, len(data.SafeOutputs.Scripts))
		for scriptName := range data.SafeOutputs.Scripts {
			scriptNames = append(scriptNames, scriptName)
		}
		sort.Strings(scriptNames)

		for _, scriptName := range scriptNames {
			scriptConfig := data.SafeOutputs.Scripts[scriptName]
			normalizedScriptName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
			customTool := generateCustomScriptToolDefinition(normalizedScriptName, scriptConfig)
			dynamicTools = append(dynamicTools, customTool)
		}
	}

	// Add custom action tools from SafeOutputs.Actions
	// Each configured action is exposed as an MCP tool with schema derived from action.yml.
	// The compiler resolves action.yml at compile time; if resolution fails the tool is still
	// added with an empty inputSchema so the agent can still attempt to call it.
	if len(data.SafeOutputs.Actions) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom action tools to dynamic tools", len(data.SafeOutputs.Actions))

		actionNames := make([]string, 0, len(data.SafeOutputs.Actions))
		for actionName := range data.SafeOutputs.Actions {
			actionNames = append(actionNames, actionName)
		}
		sort.Strings(actionNames)

		for _, actionName := range actionNames {
			actionConfig := data.SafeOutputs.Actions[actionName]
			customTool := generateActionToolDefinition(actionName, actionConfig)
			dynamicTools = append(dynamicTools, customTool)
		}
	}

	// Add dynamic dispatch_workflow tools
	if data.SafeOutputs.DispatchWorkflow != nil && len(data.SafeOutputs.DispatchWorkflow.Workflows) > 0 {
		safeOutputsConfigLog.Printf("Adding %d dispatch_workflow tools", len(data.SafeOutputs.DispatchWorkflow.Workflows))

		if data.SafeOutputs.DispatchWorkflow.WorkflowFiles == nil {
			data.SafeOutputs.DispatchWorkflow.WorkflowFiles = make(map[string]string)
		}

		for _, workflowName := range data.SafeOutputs.DispatchWorkflow.Workflows {
			fileResult, err := findWorkflowFile(workflowName, markdownPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
				dynamicTools = append(dynamicTools, generateDispatchWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			var workflowPath string
			var extension string
			var useMD bool
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else if fileResult.mdExists {
				workflowPath = fileResult.mdPath
				extension = ".lock.yml"
				useMD = true
			} else {
				safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
				dynamicTools = append(dynamicTools, generateDispatchWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			data.SafeOutputs.DispatchWorkflow.WorkflowFiles[workflowName] = extension

			var workflowInputs map[string]any
			var inputsErr error
			if useMD {
				workflowInputs, inputsErr = extractMDWorkflowDispatchInputs(workflowPath)
			} else {
				workflowInputs, inputsErr = extractWorkflowDispatchInputs(workflowPath)
			}
			if inputsErr != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
				workflowInputs = make(map[string]any)
			}

			dynamicTools = append(dynamicTools, generateDispatchWorkflowTool(workflowName, workflowInputs))
		}
	}

	// Add dynamic dispatch_repository tools
	if data.SafeOutputs.DispatchRepository != nil && len(data.SafeOutputs.DispatchRepository.Tools) > 0 {
		safeOutputsConfigLog.Printf("Adding %d dispatch_repository tools to dynamic tools", len(data.SafeOutputs.DispatchRepository.Tools))

		// Sort tool keys for deterministic output
		toolKeys := make([]string, 0, len(data.SafeOutputs.DispatchRepository.Tools))
		for toolKey := range data.SafeOutputs.DispatchRepository.Tools {
			toolKeys = append(toolKeys, toolKey)
		}
		sort.Strings(toolKeys)

		for _, toolKey := range toolKeys {
			toolConfig := data.SafeOutputs.DispatchRepository.Tools[toolKey]
			dynamicTools = append(dynamicTools, generateDispatchRepositoryTool(toolKey, toolConfig))
		}
	}

	// Add dynamic call_workflow tools
	if data.SafeOutputs.CallWorkflow != nil && len(data.SafeOutputs.CallWorkflow.Workflows) > 0 {
		safeOutputsConfigLog.Printf("Adding %d call_workflow tools", len(data.SafeOutputs.CallWorkflow.Workflows))

		if data.SafeOutputs.CallWorkflow.WorkflowFiles == nil {
			data.SafeOutputs.CallWorkflow.WorkflowFiles = make(map[string]string)
		}

		for _, workflowName := range data.SafeOutputs.CallWorkflow.Workflows {
			fileResult, err := findWorkflowFile(workflowName, markdownPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
				dynamicTools = append(dynamicTools, generateCallWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			var workflowPath string
			var extension string
			var useMD bool
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else if fileResult.mdExists {
				workflowPath = fileResult.mdPath
				extension = ".lock.yml"
				useMD = true
			} else {
				safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
				dynamicTools = append(dynamicTools, generateCallWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			relativePath := fmt.Sprintf("./.github/workflows/%s%s", workflowName, extension)
			data.SafeOutputs.CallWorkflow.WorkflowFiles[workflowName] = relativePath

			var workflowInputs map[string]any
			var inputsErr error
			if useMD {
				workflowInputs, inputsErr = extractMDWorkflowCallInputs(workflowPath)
			} else {
				workflowInputs, inputsErr = extractWorkflowCallInputs(workflowPath)
			}
			if inputsErr != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
				workflowInputs = make(map[string]any)
			}

			dynamicTools = append(dynamicTools, generateCallWorkflowTool(workflowName, workflowInputs))
		}
	}

	return dynamicTools, nil
}

// ToolsMeta is the structure written to tools_meta.json at compile time and read
// by generate_safe_outputs_tools.cjs at runtime. It avoids inlining the entire
// safe_outputs_tools.json into the compiled workflow YAML.
type ToolsMeta struct {
	// DescriptionSuffixes maps tool name → constraint text to append to the base description.
	// Example: " CONSTRAINTS: Maximum 5 issue(s) can be created."
	DescriptionSuffixes map[string]string `json:"description_suffixes"`
	// RepoParams maps tool name → "repo" inputSchema property definition, only present
	// when allowed-repos or a wildcard target-repo is configured for that tool.
	RepoParams map[string]map[string]any `json:"repo_params"`
	// DynamicTools contains tool definitions for custom safe-jobs, dispatch_workflow
	// targets, and call_workflow targets. These are workflow-specific and cannot be
	// derived from the static safe_outputs_tools.json at runtime.
	DynamicTools []map[string]any `json:"dynamic_tools"`
}

// generateToolsMetaJSON generates the content for tools_meta.json: a compact file
// that captures the workflow-specific customisations (description constraints,
// repo parameters, dynamic tools) without inlining the entire
// safe_outputs_tools.json into the compiled workflow YAML.
//
// At runtime, generate_safe_outputs_tools.cjs reads safe_outputs_tools.json from
// the actions folder, applies the meta overrides from tools_meta.json, and writes
// the final ${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json.
func generateToolsMetaJSON(data *WorkflowData, markdownPath string) (string, error) {
	if data.SafeOutputs == nil {
		empty := ToolsMeta{
			DescriptionSuffixes: map[string]string{},
			RepoParams:          map[string]map[string]any{},
			DynamicTools:        []map[string]any{},
		}
		result, err := json.Marshal(empty)
		if err != nil {
			return "", fmt.Errorf("failed to marshal empty tools meta: %w", err)
		}
		return string(result), nil
	}

	safeOutputsConfigLog.Print("Generating tools meta JSON for workflow")

	enabledTools := computeEnabledToolNames(data)

	// Compute description suffix for each enabled predefined tool.
	// enhanceToolDescription with an empty base returns just the constraint text
	// (e.g. " CONSTRAINTS: Maximum 5 issue(s).") so JavaScript can append it.
	descriptionSuffixes := make(map[string]string)
	for toolName := range enabledTools {
		suffix := enhanceToolDescription(toolName, "", data.SafeOutputs)
		if suffix != "" {
			descriptionSuffixes[toolName] = suffix
		}
	}

	// Compute repo parameter definition for each tool that needs it.
	repoParams := make(map[string]map[string]any)
	for toolName := range enabledTools {
		if param := computeRepoParamForTool(toolName, data.SafeOutputs); param != nil {
			repoParams[toolName] = param
		}
	}

	// Generate dynamic tool definitions (custom jobs + dispatch/call workflow tools).
	dynamicTools, err := generateDynamicTools(data, markdownPath)
	if err != nil {
		safeOutputsConfigLog.Printf("Error generating dynamic tools: %v", err)
		dynamicTools = []map[string]any{}
	}
	if dynamicTools == nil {
		dynamicTools = []map[string]any{}
	}

	meta := ToolsMeta{
		DescriptionSuffixes: descriptionSuffixes,
		RepoParams:          repoParams,
		DynamicTools:        dynamicTools,
	}

	result, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		safeOutputsConfigLog.Printf("Failed to marshal tools meta: %v", err)
		return "", fmt.Errorf("failed to marshal tools meta: %w", err)
	}

	safeOutputsConfigLog.Printf("Successfully generated tools meta JSON: %d description suffixes, %d repo params, %d dynamic tools",
		len(descriptionSuffixes), len(repoParams), len(dynamicTools))
	return string(result), nil
}
