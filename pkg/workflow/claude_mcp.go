package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var claudeMCPLog = logger.New("workflow:claude_mcp")

// RenderMCPConfig renders the MCP configuration for Claude engine
func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	claudeMCPLog.Printf("Rendering MCP config for Claude: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Claude uses JSON format without Copilot-specific fields and multi-line args
	return renderDefaultJSONMCPConfig(yaml, tools, mcpTools, workflowData, constants.ShellMcpServersJsonPath)
}
