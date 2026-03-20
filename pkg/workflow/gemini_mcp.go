package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var geminiMCPLog = logger.New("workflow:gemini_mcp")

// RenderMCPConfig renders MCP server configuration for Gemini CLI
func (e *GeminiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	geminiMCPLog.Printf("Rendering MCP config for Gemini: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Gemini uses JSON format without Copilot-specific fields and multi-line args
	return renderStandardJSONMCPConfig(yaml, tools, mcpTools, workflowData,
		"/tmp/gh-aw/mcp-config/mcp-servers.json", false, false, false,
		func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return renderCustomMCPConfigWrapperWithContext(yaml, toolName, toolConfig, isLast, workflowData)
		}, nil)
}
