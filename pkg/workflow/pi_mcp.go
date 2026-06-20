package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var piMCPLog = logger.New("workflow:pi_mcp")

// RenderMCPConfig renders the MCP configuration for Pi engine.
func (e *PiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	piMCPLog.Printf("Rendering MCP config for Pi: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Pi uses the same JSON MCP format as Claude and Gemini: no Copilot-specific
	// "type"/"tools" fields, no multi-line args. If Pi requires custom config
	// sections (e.g., shell-policy or provider blocks) in the future, add them here
	// similarly to how CodexEngine.RenderMCPConfig handles TOML-specific sections.
	//
	// Pi uses ShellMcpServersJsonPath (same as Claude/Gemini/Antigravity) because
	// the Pi CLI resolves its MCP config from the shell environment path. Crush and
	// OpenCode use TmpMcpServersJsonPath instead because their CLIs look for the
	// config in a different location.
	return renderDefaultJSONMCPConfig(yaml, tools, mcpTools, workflowData, constants.ShellMcpServersJsonPath)
}
