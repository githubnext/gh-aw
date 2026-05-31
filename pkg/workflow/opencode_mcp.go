package workflow

import (
	"strings"
)

// RenderMCPConfig renders MCP server configuration for OpenCode CLI
func (e *OpenCodeEngine) RenderMCPConfig(sb *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	return renderJSONMCPConfigForEngine("OpenCode", sb, tools, mcpTools, workflowData, "/tmp/gh-aw/mcp-config/mcp-servers.json")
}
