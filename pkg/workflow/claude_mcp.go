package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var claudeMCPLog = logger.New("workflow:claude_mcp")

// RenderMCPConfig renders the MCP configuration for Claude engine
func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	claudeMCPLog.Printf("Rendering MCP config for Claude: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Claude uses JSON format without Copilot-specific fields and multi-line args
	return renderStandardJSONMCPConfig(yaml, tools, mcpTools, workflowData,
		"/tmp/gh-aw/mcp-config/mcp-servers.json", false, false, false,
		func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return e.renderClaudeMCPConfigWithContext(yaml, toolName, toolConfig, isLast, workflowData)
		}, nil)
}

// renderClaudeMCPConfigWithContext generates custom MCP server configuration for a single tool in Claude workflow mcp-servers.json
// This version includes workflowData to determine if localhost URLs should be rewritten
func (e *ClaudeEngine) renderClaudeMCPConfigWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool, workflowData *WorkflowData) error {
	return renderCustomMCPConfigWrapperWithContext(yaml, toolName, toolConfig, isLast, workflowData)
}
