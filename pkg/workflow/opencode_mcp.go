package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var openCodeMCPLog = logger.New("workflow:opencode_mcp")

// RenderMCPConfig renders MCP server configuration for OpenCode CLI
func (e *OpenCodeEngine) RenderMCPConfig(sb *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	openCodeMCPLog.Printf("Rendering MCP config for OpenCode: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	return renderDefaultJSONMCPConfig(sb, tools, mcpTools, workflowData, "/tmp/gh-aw/mcp-config/mcp-servers.json")
}
