package workflow

import "strings"

// RenderMCPConfig renders the MCP configuration for Claude engine
func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Use shared JSON MCP config renderer
	RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
		ConfigPath: "/tmp/gh-aw/mcp-config/mcp-servers.json",
		Renderers: MCPToolRenderers{
			RenderGitHub:           e.renderGitHubClaudeMCPConfig,
			RenderPlaywright:       e.renderPlaywrightMCPConfig,
			RenderCacheMemory:      e.renderCacheMemoryMCPConfig,
			RenderAgenticWorkflows: e.renderAgenticWorkflowsMCPConfig,
			RenderSafeOutputs:      e.renderSafeOutputsMCPConfig,
			RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
				renderMCPFetchServerConfig(yaml, "json", "              ", isLast, false)
			},
			RenderCustomMCPConfig: e.renderClaudeMCPConfig,
		},
	})
}

// renderGitHubClaudeMCPConfig generates the GitHub MCP server configuration
// Supports both local (Docker) and remote (hosted) modes
func (e *ClaudeEngine) renderGitHubClaudeMCPConfig(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
	githubType := getGitHubType(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	toolsets := getGitHubToolsets(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Check if remote mode is enabled (type: remote)
	if githubType == "remote" {
		// Use shell environment variable instead of GitHub Actions expression to prevent template injection
		// The actual GitHub expression is set in the step's env: block
		// Render remote configuration using shared helper
		RenderGitHubMCPRemoteConfig(yaml, GitHubMCPRemoteOptions{
			ReadOnly:           readOnly,
			Toolsets:           toolsets,
			AuthorizationValue: "Bearer $GITHUB_MCP_SERVER_TOKEN",
			IncludeToolsField:  false, // Claude doesn't use tools field
			AllowedTools:       nil,
			IncludeEnvSection:  false, // Claude doesn't use env section
		})
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)

		// Use shell environment variable instead of GitHub Actions expression to prevent template injection
		// The actual GitHub expression is set in the step's env: block
		RenderGitHubMCPDockerConfig(yaml, GitHubMCPDockerOptions{
			ReadOnly:           readOnly,
			Toolsets:           toolsets,
			DockerImageVersion: githubDockerImageVersion,
			CustomArgs:         customArgs,
			IncludeTypeField:   false, // Claude doesn't include "type" field
			AllowedTools:       nil,   // Claude doesn't use tools field
			EffectiveToken:     "",    // Not used anymore - token passed via env
		})
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
func (e *ClaudeEngine) renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	renderPlaywrightMCPConfig(yaml, playwrightTool, isLast)
}

// renderClaudeMCPConfig generates custom MCP server configuration for a single tool in Claude workflow mcp-servers.json
func (e *ClaudeEngine) renderClaudeMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	return renderCustomMCPConfigWrapper(yaml, toolName, toolConfig, isLast)
}

// renderCacheMemoryMCPConfig handles cache-memory configuration without MCP server mounting
// Cache-memory is now a simple file share, not an MCP server
func (e *ClaudeEngine) renderCacheMemoryMCPConfig(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
	// Cache-memory no longer uses MCP server mounting
	// The cache folder is available as a simple file share at /tmp/gh-aw/cache-memory/
	// The folder is created by the cache step and is accessible to all tools
	// No MCP configuration is needed for simple file access
}

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
func (e *ClaudeEngine) renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	renderSafeOutputsMCPConfig(yaml, isLast)
}

// renderAgenticWorkflowsMCPConfig generates the Agentic Workflows MCP server configuration
func (e *ClaudeEngine) renderAgenticWorkflowsMCPConfig(yaml *strings.Builder, isLast bool) {
	renderAgenticWorkflowsMCPConfig(yaml, isLast)
}
