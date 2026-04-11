package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var opencodeMCPLog = logger.New("workflow:opencode_mcp")

// RenderMCPConfig renders MCP server configuration for OpenCode CLI
func (e *OpenCodeEngine) RenderMCPConfig(sb *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	opencodeMCPLog.Printf("Rendering MCP config for OpenCode: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// OpenCode uses JSON format without Copilot-specific fields and multi-line args
	return renderStandardJSONMCPConfig(sb, tools, mcpTools, workflowData,
		"/tmp/gh-aw/mcp-config/mcp-servers.json", false, false,
		func(builder *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return renderCustomMCPConfigWrapperWithContext(builder, toolName, toolConfig, isLast, workflowData)
		}, nil)
}
