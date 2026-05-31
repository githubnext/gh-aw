package workflow

import (
	"strings"
)

// RenderMCPConfig renders the MCP configuration for Claude engine
func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	return renderJSONMCPConfigForEngine("Claude", yaml, tools, mcpTools, workflowData, "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json")
}
