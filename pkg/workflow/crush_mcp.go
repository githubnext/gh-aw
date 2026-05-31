package workflow

import (
	"strings"
)

// RenderMCPConfig renders MCP server configuration for Crush CLI
func (e *CrushEngine) RenderMCPConfig(sb *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	return renderJSONMCPConfigForEngine("Crush", sb, tools, mcpTools, workflowData, "/tmp/gh-aw/mcp-config/mcp-servers.json")
}
