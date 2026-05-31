package workflow

import (
	"strings"
)

// RenderMCPConfig renders MCP server configuration for Gemini CLI
func (e *GeminiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	return renderJSONMCPConfigForEngine("Gemini", yaml, tools, mcpTools, workflowData, "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json")
}
