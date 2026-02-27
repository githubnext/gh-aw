package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var geminiMCPLog = logger.New("workflow:gemini_mcp")

// RenderMCPConfig renders MCP server configuration for Gemini CLI
func (e *GeminiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	geminiMCPLog.Printf("Rendering MCP config for Gemini: tool_count=%d, mcp_tool_count=%d", len(tools), len(mcpTools))

	// Create unified renderer with Gemini-specific options
	createRenderer := func(isLast bool) *MCPConfigRendererUnified {
		return NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: false,
			InlineArgs:           false,
			Format:               "json", // Gemini uses JSON format like Claude/Codex
			IsLast:               isLast,
			ActionMode:           GetActionModeFromWorkflowData(workflowData),
		})
	}

	// Use shared JSON MCP config renderer
	return RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
		ConfigPath:    "/tmp/gh-aw/mcp-config/mcp-servers.json",
		GatewayConfig: buildMCPGatewayConfig(workflowData),
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
			RenderCacheMemory: noOpCacheMemoryRenderer,
			RenderAgenticWorkflows: func(yaml *strings.Builder, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderAgenticWorkflowsMCP(yaml)
			},
			RenderSafeOutputs: func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
				renderer := createRenderer(isLast)
				renderer.RenderSafeOutputsMCP(yaml, workflowData)
			},
			RenderSafeInputs: func(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSafeInputsMCP(yaml, safeInputs, workflowData)
			},
			RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
				renderMCPFetchServerConfig(yaml, "json", "              ", isLast, false)
			},
			RenderCustomMCPConfig: func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
				return renderCustomMCPConfigWrapperWithContext(yaml, toolName, toolConfig, isLast, workflowData)
			},
		},
	})
}
