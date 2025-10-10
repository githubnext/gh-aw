package workflow

import (
	_ "embed"
	"encoding/json"
)

//go:embed data/safe_outputs_tools.json
var safeOutputsToolsJSON string

// SafeOutputToolDefinition represents a tool definition for the safe-outputs MCP server
type SafeOutputToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	HasHandler  bool                   `json:"hasHandler,omitempty"`
}

// GetSafeOutputsToolsDefinitions returns the parsed tool definitions
func GetSafeOutputsToolsDefinitions() ([]SafeOutputToolDefinition, error) {
	var tools []SafeOutputToolDefinition
	if err := json.Unmarshal([]byte(safeOutputsToolsJSON), &tools); err != nil {
		return nil, err
	}
	return tools, nil
}

// GenerateFilteredToolsJSON generates a JSON string of tools filtered by the safe-outputs configuration
// The output is a JSON object mapping normalized tool names to tool definitions
func (c *Compiler) GenerateFilteredToolsJSON(data *WorkflowData) (string, error) {
	if data == nil || data.SafeOutputs == nil {
		return "{}", nil
	}

	// Get all tool definitions
	allTools, err := GetSafeOutputsToolsDefinitions()
	if err != nil {
		return "", err
	}

	// Build a map of enabled tools based on safe-outputs configuration
	enabledTools := make(map[string]SafeOutputToolDefinition)

	// Helper function to normalize tool names (convert dashes to underscores, lowercase)
	normalizeTool := func(name string) string {
		// This matches the normTool function in the JavaScript
		result := ""
		for _, ch := range name {
			if ch == '-' {
				result += "_"
			} else {
				result += string(ch)
			}
		}
		// JavaScript toLowerCase is locale-independent for ASCII
		return result
	}

	// Check which tools are enabled in the configuration
	safeOutputsConfig := c.generateSafeOutputsConfigMap(data)

	for _, tool := range allTools {
		// Check if this tool is enabled in the configuration
		for configKey := range safeOutputsConfig {
			if normalizeTool(configKey) == tool.Name {
				enabledTools[tool.Name] = tool
				break
			}
		}
	}

	// Convert to JSON
	toolsJSON, err := json.Marshal(enabledTools)
	if err != nil {
		return "", err
	}

	return string(toolsJSON), nil
}

// generateSafeOutputsConfigMap generates a map of safe-outputs configuration
// This is similar to generateSafeOutputsConfig but returns a map instead of JSON string
func (c *Compiler) generateSafeOutputsConfigMap(data *WorkflowData) map[string]interface{} {
	safeOutputsConfig := make(map[string]interface{})

	if data.SafeOutputs == nil {
		return safeOutputsConfig
	}

	// Handle safe-outputs configuration if present
	if data.SafeOutputs.CreateIssues != nil {
		issueConfig := map[string]interface{}{}
		if data.SafeOutputs.CreateIssues.Max > 0 {
			issueConfig["max"] = data.SafeOutputs.CreateIssues.Max
		}
		if data.SafeOutputs.CreateIssues.Min > 0 {
			issueConfig["min"] = data.SafeOutputs.CreateIssues.Min
		}
		safeOutputsConfig["create-issue"] = issueConfig
	}
	if data.SafeOutputs.AddComments != nil {
		commentConfig := map[string]interface{}{}
		if data.SafeOutputs.AddComments.Target != "" {
			commentConfig["target"] = data.SafeOutputs.AddComments.Target
		}
		if data.SafeOutputs.AddComments.Max > 0 {
			commentConfig["max"] = data.SafeOutputs.AddComments.Max
		}
		if data.SafeOutputs.AddComments.Min > 0 {
			commentConfig["min"] = data.SafeOutputs.AddComments.Min
		}
		safeOutputsConfig["add-comment"] = commentConfig
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		discussionConfig := map[string]interface{}{}
		if data.SafeOutputs.CreateDiscussions.Max > 0 {
			discussionConfig["max"] = data.SafeOutputs.CreateDiscussions.Max
		}
		if data.SafeOutputs.CreateDiscussions.Min > 0 {
			discussionConfig["min"] = data.SafeOutputs.CreateDiscussions.Min
		}
		safeOutputsConfig["create-discussion"] = discussionConfig
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		prConfig := map[string]interface{}{}
		if data.SafeOutputs.CreatePullRequests.Min > 0 {
			prConfig["min"] = data.SafeOutputs.CreatePullRequests.Min
		}
		safeOutputsConfig["create-pull-request"] = prConfig
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		prReviewCommentConfig := map[string]interface{}{}
		if data.SafeOutputs.CreatePullRequestReviewComments.Max > 0 {
			prReviewCommentConfig["max"] = data.SafeOutputs.CreatePullRequestReviewComments.Max
		}
		if data.SafeOutputs.CreatePullRequestReviewComments.Min > 0 {
			prReviewCommentConfig["min"] = data.SafeOutputs.CreatePullRequestReviewComments.Min
		}
		safeOutputsConfig["create-pull-request-review-comment"] = prReviewCommentConfig
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		securityReportConfig := map[string]interface{}{}
		if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
			securityReportConfig["max"] = data.SafeOutputs.CreateCodeScanningAlerts.Max
		}
		if data.SafeOutputs.CreateCodeScanningAlerts.Min > 0 {
			securityReportConfig["min"] = data.SafeOutputs.CreateCodeScanningAlerts.Min
		}
		safeOutputsConfig["create-code-scanning-alert"] = securityReportConfig
	}
	if data.SafeOutputs.AddLabels != nil {
		labelConfig := map[string]interface{}{}
		if data.SafeOutputs.AddLabels.Max > 0 {
			labelConfig["max"] = data.SafeOutputs.AddLabels.Max
		}
		if data.SafeOutputs.AddLabels.Min > 0 {
			labelConfig["min"] = data.SafeOutputs.AddLabels.Min
		}
		if len(data.SafeOutputs.AddLabels.Allowed) > 0 {
			labelConfig["allowed"] = data.SafeOutputs.AddLabels.Allowed
		}
		safeOutputsConfig["add-labels"] = labelConfig
	}
	if data.SafeOutputs.UpdateIssues != nil {
		updateConfig := map[string]interface{}{}
		if data.SafeOutputs.UpdateIssues.Max > 0 {
			updateConfig["max"] = data.SafeOutputs.UpdateIssues.Max
		}
		if data.SafeOutputs.UpdateIssues.Min > 0 {
			updateConfig["min"] = data.SafeOutputs.UpdateIssues.Min
		}
		safeOutputsConfig["update-issue"] = updateConfig
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		pushToBranchConfig := map[string]interface{}{}
		if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
			pushToBranchConfig["target"] = data.SafeOutputs.PushToPullRequestBranch.Target
		}
		if data.SafeOutputs.PushToPullRequestBranch.Max > 0 {
			pushToBranchConfig["max"] = data.SafeOutputs.PushToPullRequestBranch.Max
		}
		if data.SafeOutputs.PushToPullRequestBranch.Min > 0 {
			pushToBranchConfig["min"] = data.SafeOutputs.PushToPullRequestBranch.Min
		}
		safeOutputsConfig["push-to-pull-request-branch"] = pushToBranchConfig
	}
	if data.SafeOutputs.UploadAssets != nil {
		uploadConfig := map[string]interface{}{}
		if data.SafeOutputs.UploadAssets.Max > 0 {
			uploadConfig["max"] = data.SafeOutputs.UploadAssets.Max
		}
		if data.SafeOutputs.UploadAssets.Min > 0 {
			uploadConfig["min"] = data.SafeOutputs.UploadAssets.Min
		}
		safeOutputsConfig["upload-asset"] = uploadConfig
	}
	if data.SafeOutputs.MissingTool != nil {
		missingToolConfig := map[string]interface{}{}
		safeOutputsConfig["missing-tool"] = missingToolConfig
	}

	// Add safe-jobs as well
	for jobName := range data.SafeOutputs.Jobs {
		safeOutputsConfig[jobName] = map[string]interface{}{}
	}

	return safeOutputsConfig
}
