// Package workflow provides utility functions for MCP configuration processing
// and rendering.
//
// # MCP Rendering
//
// This file consolidates MCP infrastructure helpers: URL rewriting for Docker
// networking and shared rendering functions used across multiple engines
// (Claude, Gemini, Copilot, Codex).
//
// URL rewriting:
// When MCP servers run on the host machine (like safe-outputs HTTP server
// on port 3001) but need to be accessed from within a Docker container
// (like the firewall container running the AI agent), localhost URLs must
// be rewritten to use host.docker.internal.
//
// Supported URL patterns:
//   - http://localhost:port → http://host.docker.internal:port
//   - https://localhost:port → https://host.docker.internal:port
//   - http://127.0.0.1:port → http://host.docker.internal:port
//   - https://127.0.0.1:port → https://host.docker.internal:port
//
// Related files:
//   - mcp_renderer.go: Uses URL rewriting for HTTP MCP servers
//   - safe_outputs.go: Safe outputs HTTP server configuration
//   - mcp_scripts.go: MCP Scripts HTTP server configuration
package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpRenderingLog = logger.New("workflow:mcp-rendering")

// rewriteLocalhostToDockerHost rewrites localhost URLs to use host.docker.internal
// This is necessary when MCP servers run on the host machine but are accessed from within
// a Docker container (e.g., when firewall/sandbox is enabled)
func rewriteLocalhostToDockerHost(url string) string {
	// Define the localhost patterns to replace and their docker equivalents
	// Each pattern is a (prefix, replacement) pair
	replacements := []struct {
		prefix      string
		replacement string
	}{
		{"http://localhost", "http://host.docker.internal"},
		{"https://localhost", "https://host.docker.internal"},
		{"http://127.0.0.1", "http://host.docker.internal"},
		{"https://127.0.0.1", "https://host.docker.internal"},
	}

	for _, r := range replacements {
		if strings.HasPrefix(url, r.prefix) {
			newURL := r.replacement + url[len(r.prefix):]
			mcpRenderingLog.Printf("Rewriting localhost URL for Docker access: %s -> %s", url, newURL)
			return newURL
		}
	}

	return url
}

// shouldRewriteLocalhostToDocker returns true when MCP server localhost URLs should be
// rewritten to host.docker.internal so that containerised AI agents can reach servers
// running on the host. Rewriting is enabled whenever the agent sandbox is active
// (i.e. sandbox.agent is not explicitly disabled).
func shouldRewriteLocalhostToDocker(workflowData *WorkflowData) bool {
	result := workflowData != nil && (workflowData.SandboxConfig == nil ||
		workflowData.SandboxConfig.Agent == nil ||
		!workflowData.SandboxConfig.Agent.Disabled)
	mcpRenderingLog.Printf("shouldRewriteLocalhostToDocker: %v (agent sandbox active)", result)
	return result
}

// noOpCacheMemoryRenderer is a no-op MCPToolRenderers.RenderCacheMemory function for engines
// that do not need an MCP server entry for cache-memory. Cache-memory is a simple file share
// accessible at /tmp/gh-aw/cache-memory/ and requires no MCP configuration.
func noOpCacheMemoryRenderer(_ *strings.Builder, _ bool, _ *WorkflowData) {}

// renderStandardJSONMCPConfig is a shared helper for JSON MCP config rendering used by
// Claude, Gemini, Copilot, and Codex engines. It consolidates the repeated sequence of:
// buildMCPRendererFactory → buildMCPGatewayConfig → buildStandardJSONMCPRenderers → RenderJSONMCPConfig.
//
// Parameters:
//   - yaml: output builder
//   - tools: tool configurations from frontmatter
//   - mcpTools: list of enabled MCP tool names
//   - workflowData: compiled workflow context
//   - configPath: engine-specific MCP config file path
//   - includeCopilotFields: whether to include "type" and "tools" fields (true for Copilot)
//   - inlineArgs: whether to render args inline (true for Copilot) vs multi-line
//   - webFetchIncludeTools: whether the web-fetch server includes a tools field (true for Copilot)
//   - renderCustom: engine-specific handler for custom MCP tool entries
//   - filterTool: optional tool filter function; nil to include all tools
func renderStandardJSONMCPConfig(
	yaml *strings.Builder,
	tools map[string]any,
	mcpTools []string,
	workflowData *WorkflowData,
	configPath string,
	includeCopilotFields bool,
	inlineArgs bool,
	webFetchIncludeTools bool,
	renderCustom RenderCustomMCPToolConfigHandler,
	filterTool func(string) bool,
) error {
	mcpRenderingLog.Printf("Rendering standard JSON MCP config: config_path=%s tools=%d mcp_tools=%d", configPath, len(tools), len(mcpTools))
	createRenderer := buildMCPRendererFactory(workflowData, "json", includeCopilotFields, inlineArgs)
	return RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
		ConfigPath:    configPath,
		GatewayConfig: buildMCPGatewayConfig(workflowData),
		Renderers:     buildStandardJSONMCPRenderers(workflowData, createRenderer, webFetchIncludeTools, renderCustom),
		FilterTool:    filterTool,
	})
}

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
// All standard tool callbacks (GitHub, Playwright, CacheMemory, AgenticWorkflows,
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
		RenderQmd: func(yaml *strings.Builder, qmdTool any, isLast bool, workflowData *WorkflowData) {
			createRenderer(isLast).RenderQmdMCP(yaml, qmdTool, workflowData)
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
