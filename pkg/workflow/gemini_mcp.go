package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var geminiMCPLog = logger.New("workflow:gemini_mcp")

// RenderMCPConfig renders MCP server configuration for Gemini CLI
func (e *GeminiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	geminiMCPLog.Printf("Rendering MCP config for Gemini: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Gemini uses JSON format without Copilot-specific fields and multi-line args
	return renderDefaultJSONMCPConfig(yaml, tools, mcpTools, workflowData, constants.ShellMcpServersJsonPath)
}
