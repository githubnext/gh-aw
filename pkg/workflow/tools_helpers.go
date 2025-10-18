package workflow

// This file contains helper functions for working with tools using the parsed Tools struct

// getToolFromWorkflowData is a helper to get a tool configuration from WorkflowData
// It checks ParsedTools first, and falls back to the raw Tools map for backwards compatibility
func getToolFromWorkflowData(data *WorkflowData, toolName string) any {
	if data.ParsedTools != nil {
		return data.ParsedTools.GetTool(toolName)
	}
	return data.Tools[toolName]
}

// hasToolInWorkflowData is a helper to check if a tool is configured in WorkflowData
// It checks ParsedTools first, and falls back to the raw Tools map for backwards compatibility
func hasToolInWorkflowData(data *WorkflowData, toolName string) bool {
	if data.ParsedTools != nil {
		return data.ParsedTools.HasTool(toolName)
	}
	_, exists := data.Tools[toolName]
	return exists
}

// getToolNamesFromWorkflowData is a helper to get all tool names from WorkflowData
// It checks ParsedTools first, and falls back to the raw Tools map for backwards compatibility
func getToolNamesFromWorkflowData(data *WorkflowData) []string {
	if data.ParsedTools != nil {
		return data.ParsedTools.GetToolNames()
	}

	names := make([]string, 0, len(data.Tools))
	for name := range data.Tools {
		names = append(names, name)
	}
	return names
}

// getGitHubConfigFromWorkflowData is a helper to get GitHub tool configuration from WorkflowData
// Returns the configuration as a typed struct when ParsedTools is available, or the raw value otherwise
func getGitHubConfigFromWorkflowData(data *WorkflowData) any {
	if data.ParsedTools != nil {
		config := data.ParsedTools.GetGitHubConfig()
		if config != nil {
			return config
		}
	}
	return data.Tools["github"]
}

// getPlaywrightConfigFromWorkflowData is a helper to get Playwright tool configuration from WorkflowData
// Returns the configuration as a typed struct when ParsedTools is available, or the raw value otherwise
// nolint:unused // Public API for accessing parsed tool configuration
func getPlaywrightConfigFromWorkflowData(data *WorkflowData) any {
	if data.ParsedTools != nil {
		config := data.ParsedTools.GetPlaywrightConfig()
		if config != nil {
			return config
		}
	}
	return data.Tools["playwright"]
}

// getClaudeConfigFromWorkflowData is a helper to get Claude tool configuration from WorkflowData
// Returns the configuration as a typed struct when ParsedTools is available, or the raw value otherwise
// nolint:unused // Public API for accessing parsed tool configuration
func getClaudeConfigFromWorkflowData(data *WorkflowData) any {
	if data.ParsedTools != nil {
		config := data.ParsedTools.GetClaudeConfig()
		if config != nil {
			return config
		}
	}
	return data.Tools["claude"]
}
