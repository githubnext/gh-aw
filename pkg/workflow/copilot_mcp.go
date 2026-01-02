package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var copilotMCPLog = logger.New("workflow:copilot_mcp")

// RenderMCPConfig generates MCP server configuration for Copilot CLI
func (e *CopilotEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	copilotMCPLog.Printf("Rendering MCP config for Copilot engine: mcpTools=%d", len(mcpTools))

	// Create the directory first
	yaml.WriteString("          mkdir -p /home/runner/.copilot\n")

	// Create unified renderer with Copilot-specific options
	// Copilot uses JSON format with type and tools fields, and inline args
	createRenderer := func(isLast bool) *MCPConfigRendererUnified {
		return NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: true, // Copilot uses "type" and "tools" fields
			InlineArgs:           true, // Copilot uses inline args format
			Format:               "json",
			IsLast:               isLast,
		})
	}

	// Use shared JSON MCP config renderer with unified renderer methods
	options := JSONMCPConfigOptions{
		ConfigPath: "/home/runner/.copilot/mcp-config.json",
		Renderers: MCPToolRenderers{
			RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
				renderer := createRenderer(isLast)
				renderer.RenderGitHubMCP(yaml, githubTool, workflowData)
			},
			RenderPlaywright: func(yaml *strings.Builder, playwrightTool any, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderPlaywrightMCP(yaml, playwrightTool)
			},
			RenderSerena: func(yaml *strings.Builder, serenaTool any, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSerenaMCP(yaml, serenaTool)
			},
			RenderCacheMemory: func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
				// Cache-memory is not used for Copilot (filtered out)
			},
			RenderAgenticWorkflows: func(yaml *strings.Builder, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderAgenticWorkflowsMCP(yaml)
			},
			RenderSafeOutputs: func(yaml *strings.Builder, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSafeOutputsMCP(yaml)
			},
			RenderSafeInputs: func(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSafeInputsMCP(yaml, safeInputs, workflowData)
			},
			RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
				renderMCPFetchServerConfig(yaml, "json", "              ", isLast, true)
			},
			RenderCustomMCPConfig: e.renderCopilotMCPConfig,
		},
		FilterTool: func(toolName string) bool {
			// Filter out cache-memory for Copilot
			// Cache-memory is handled as a simple file share, not an MCP server
			return toolName != "cache-memory"
		},
		PostEOFCommands: func(yaml *strings.Builder) {
			// Add debug output
			yaml.WriteString("          echo \"-------START MCP CONFIG-----------\"\n")
			yaml.WriteString("          cat /home/runner/.copilot/mcp-config.json\n")
			yaml.WriteString("          echo \"-------END MCP CONFIG-----------\"\n")
			yaml.WriteString("          echo \"-------/home/runner/.copilot-----------\"\n")
			yaml.WriteString("          find /home/runner/.copilot\n")
		},
	}

	// Add gateway configuration if MCP gateway is enabled
	if workflowData != nil && workflowData.SandboxConfig != nil && workflowData.SandboxConfig.MCP != nil {
		copilotMCPLog.Print("MCP gateway is enabled, adding gateway config to MCP config")
		options.GatewayConfig = workflowData.SandboxConfig.MCP
	}

	RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, options)
	//GITHUB_COPILOT_CLI_MODE
	yaml.WriteString("          echo \"HOME: $HOME\"\n")
	yaml.WriteString("          echo \"GITHUB_COPILOT_CLI_MODE: $GITHUB_COPILOT_CLI_MODE\"\n")
}

// renderCopilotMCPConfig generates custom MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderCopilotMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	copilotMCPLog.Printf("Rendering custom MCP config for tool: %s", toolName)
	// Use the shared renderer with copilot-specific requirements
	renderer := MCPConfigRenderer{
		Format:                "json",
		IndentLevel:           "                ",
		RequiresCopilotFields: true,
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
