package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var mcpDetectionLog = logger.New("workflow:mcp_detection")

// HasMCPServers checks if the workflow has any MCP servers configured
func HasMCPServers(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}
	parsedTools := workflowData.ParsedTools
	if parsedTools == nil {
		// Check safe-outputs and mcp-scripts even without tools
		if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
			mcpDetectionLog.Print("MCP server detected via safe-outputs configuration")
			return true
		}
		if IsMCPScriptsEnabled(workflowData.MCPScripts) {
			mcpDetectionLog.Print("MCP server detected via mcp-scripts configuration")
			return true
		}
		return false
	}

	// Check for standard MCP tools
	if parsedTools.GitHub != nil {
		mcpDetectionLog.Print("MCP server detected via built-in tool: github")
		return true
	}
	if parsedTools.Playwright != nil {
		if isPlaywrightCLIMode(parsedTools) {
			mcpDetectionLog.Print("Skipping playwright MCP detection: tools.playwright.mode is cli")
		} else {
			mcpDetectionLog.Print("MCP server detected via built-in tool: playwright")
			return true
		}
	}
	if parsedTools.CacheMemory != nil {
		mcpDetectionLog.Print("MCP server detected via built-in tool: cache-memory")
		return true
	}
	if parsedTools.AgenticWorkflows != nil && parsedTools.AgenticWorkflows.Enabled {
		mcpDetectionLog.Print("MCP server detected via built-in tool: agentic-workflows")
		return true
	}
	// Check for custom MCP tools
	for toolName, toolConfig := range parsedTools.Custom {
		mcpConfigMap := mcpServerConfigToMap(toolConfig)
		if hasMcp, _ := hasMCPConfig(mcpConfigMap); hasMcp {
			if mcpDetectionLog.Enabled() {
				mcpDetectionLog.Printf("MCP server detected via custom tool config: %s", toolName)
			}
			return true
		}
	}

	// Check if safe-outputs is enabled (adds safe-outputs MCP server)
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		mcpDetectionLog.Print("MCP server detected via safe-outputs configuration")
		return true
	}

	// Check if mcp-scripts is configured and feature flag is enabled (adds mcp-scripts MCP server)
	if IsMCPScriptsEnabled(workflowData.MCPScripts) {
		mcpDetectionLog.Print("MCP server detected via mcp-scripts configuration")
		return true
	}

	mcpDetectionLog.Print("No MCP servers detected in workflow configuration")
	return false
}
