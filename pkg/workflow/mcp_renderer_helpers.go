package workflow

import "strings"

// buildMCPRendererFactory creates a factory function for MCPConfigRendererUnified instances.
// The returned function accepts isLast as a parameter and creates a renderer with engine-specific
// options derived from the provided parameters and workflowData at call time.
func buildMCPRendererFactory(workflowData *WorkflowData, format string, includeCopilotFields, inlineArgs bool) func(bool) *MCPConfigRendererUnified {
	return func(isLast bool) *MCPConfigRendererUnified {
		return NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields:   includeCopilotFields,
			InlineArgs:             inlineArgs,
			Format:                 format,
			IsLast:                 isLast,
			ActionMode:             GetActionModeFromWorkflowData(workflowData),
			WriteSinkGuardPolicies: deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
		})
	}
}

// buildStandardJSONMCPRenderers constructs MCPToolRenderers with the standard rendering callbacks
// shared across JSON-format engines (Claude, Gemini, Copilot, Codex gateway).
//
// All eight standard tool callbacks (GitHub, Playwright, Serena, CacheMemory, AgenticWorkflows,
// SafeOutputs, MCPScripts, WebFetch) are wired to the corresponding unified renderer methods
// via createRenderer. Cache-memory is always a no-op for these engines.
//
// webFetchIncludeTools controls whether the web-fetch server includes a tools field:
// set to true for Copilot (which uses inline args) and false for all other engines.
//
// renderCustom is the engine-specific handler for custom MCP tool configuration entries.
func buildStandardJSONMCPRenderers(
	workflowData *WorkflowData,
	createRenderer func(bool) *MCPConfigRendererUnified,
	webFetchIncludeTools bool,
	renderCustom RenderCustomMCPToolConfigHandler,
) MCPToolRenderers {
	return MCPToolRenderers{
		RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
			createRenderer(isLast).RenderGitHubMCP(yaml, githubTool, workflowData)
		},
		RenderPlaywright: func(yaml *strings.Builder, playwrightTool any, isLast bool) {
			createRenderer(isLast).RenderPlaywrightMCP(yaml, playwrightTool)
		},
		RenderSerena: func(yaml *strings.Builder, serenaTool any, isLast bool) {
			createRenderer(isLast).RenderSerenaMCP(yaml, serenaTool)
		},
		RenderCacheMemory: noOpCacheMemoryRenderer,
		RenderAgenticWorkflows: func(yaml *strings.Builder, isLast bool) {
			createRenderer(isLast).RenderAgenticWorkflowsMCP(yaml)
		},
		RenderSafeOutputs: func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
			createRenderer(isLast).RenderSafeOutputsMCP(yaml, workflowData)
		},
		RenderMCPScripts: func(yaml *strings.Builder, mcpScripts *MCPScriptsConfig, isLast bool) {
			createRenderer(isLast).RenderMCPScriptsMCP(yaml, mcpScripts, workflowData)
		},
		RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
			renderMCPFetchServerConfig(yaml, "json", "              ", isLast, webFetchIncludeTools, deriveWriteSinkGuardPolicyFromWorkflow(workflowData))
		},
		RenderCustomMCPConfig: renderCustom,
	}
}
