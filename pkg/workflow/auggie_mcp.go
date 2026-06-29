package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// RenderMCPConfig renders the MCP configuration for the Auggie engine.
func (e *AuggieEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	return renderDefaultJSONMCPConfig(yaml, tools, mcpTools, workflowData, constants.ShellMcpServersJsonPath)
}
