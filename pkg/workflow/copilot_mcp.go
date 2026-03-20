package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var copilotMCPLog = logger.New("workflow:copilot_mcp")

// copilotMCPToolFilter returns true for MCP tools that should be included in the Copilot MCP config.
// Cache-memory is excluded because it is handled as a simple file share, not an MCP server.
func copilotMCPToolFilter(toolName string) bool {
	return toolName != "cache-memory"
}

// RenderMCPConfig generates MCP server configuration for Copilot CLI
func (e *CopilotEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	copilotMCPLog.Printf("Rendering MCP config for Copilot engine: mcpTools=%d", len(mcpTools))

	// Create the directory first
	yaml.WriteString("          mkdir -p /home/runner/.copilot\n")

	// Copilot uses JSON format with type and tools fields, and inline args
	// webFetchIncludeTools=true: Copilot requires a tools field in the web-fetch server config
	return renderStandardJSONMCPConfig(yaml, tools, mcpTools, workflowData,
		"/home/runner/.copilot/mcp-config.json", true, true, true,
		func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return e.renderCopilotMCPConfigWithContext(yaml, toolName, toolConfig, isLast, workflowData)
		},
		copilotMCPToolFilter,
	)
}

// renderCopilotMCPConfigWithContext generates custom MCP server configuration for Copilot CLI
// This version includes workflowData to determine if localhost URLs should be rewritten
func (e *CopilotEngine) renderCopilotMCPConfigWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool, workflowData *WorkflowData) error {
	copilotMCPLog.Printf("Rendering custom MCP config for tool: %s", toolName)

	// Determine if localhost URLs should be rewritten to host.docker.internal
	// This is needed when firewall is enabled (agent is not disabled)
	rewriteLocalhost := shouldRewriteLocalhostToDocker(workflowData)
	copilotMCPLog.Printf("Localhost URL rewriting for tool %s: enabled=%t", toolName, rewriteLocalhost)

	// Use the shared renderer with copilot-specific requirements
	renderer := MCPConfigRenderer{
		Format:                   "json",
		IndentLevel:              "                ",
		RequiresCopilotFields:    true,
		RewriteLocalhostToDocker: rewriteLocalhost,
		GuardPolicies:            deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
	}

	yaml.WriteString("              \"" + toolName + "\": {\n")

	// Use shared renderer for the server configuration
	if err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer); err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}
