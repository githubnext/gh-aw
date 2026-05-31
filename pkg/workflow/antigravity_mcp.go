package workflow

import (
	"strings"
)

// RenderMCPConfig renders MCP server configuration for Antigravity CLI
func (e *AntigravityEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	return renderJSONMCPConfigForEngine("Antigravity", yaml, tools, mcpTools, workflowData, "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json")
}
