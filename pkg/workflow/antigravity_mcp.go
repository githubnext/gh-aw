package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var antigravityMCPLog = logger.New("workflow:antigravity_mcp")

// RenderMCPConfig renders MCP server configuration for Antigravity CLI
func (e *AntigravityEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	antigravityMCPLog.Printf("Rendering MCP config for Antigravity: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Antigravity uses JSON format without Copilot-specific fields and multi-line args
	return renderDefaultJSONMCPConfig(yaml, tools, mcpTools, workflowData, constants.ShellMcpServersJsonPath)
}
