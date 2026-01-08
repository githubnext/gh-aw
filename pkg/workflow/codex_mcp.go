package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var codexMCPLog = logger.New("workflow:codex_mcp")

// RenderMCPConfig generates MCP server configuration for Codex
func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	if codexMCPLog.Enabled() {
		codexMCPLog.Printf("Rendering MCP config for Codex: mcp_tools=%v, tool_count=%d", mcpTools, len(tools))
	}

	// Create unified renderer with Codex-specific options
	// Codex uses TOML format without Copilot-specific fields and multi-line args
	createRenderer := func(isLast bool) *MCPConfigRendererUnified {
		return NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: false, // Codex doesn't use "type" and "tools" fields
			InlineArgs:           false, // Codex uses multi-line args format
			Format:               "toml",
			IsLast:               isLast,
		})
	}

	yaml.WriteString("          cat > /tmp/gh-aw/mcp-config/config.toml << EOF\n")

	// Add history configuration to disable persistence
	yaml.WriteString("          [history]\n")
	yaml.WriteString("          persistence = \"none\"\n")

	// Add shell environment policy to control which environment variables are passed through
	// This is a security feature to prevent accidental exposure of secrets
	e.renderShellEnvironmentPolicy(yaml, tools, mcpTools, workflowData)

	// Expand neutral tools (like playwright: null) to include the copilot agent tools
	expandedTools := e.expandNeutralToolsToCodexToolsFromMap(tools)

	// Generate [mcp_servers] section
	for _, toolName := range mcpTools {
		renderer := createRenderer(false) // isLast is always false in TOML format
		switch toolName {
		case "github":
			githubTool := expandedTools["github"]
			renderer.RenderGitHubMCP(yaml, githubTool, workflowData)
		case "playwright":
			playwrightTool := expandedTools["playwright"]
			renderer.RenderPlaywrightMCP(yaml, playwrightTool)
		case "serena":
			serenaTool := expandedTools["serena"]
			renderer.RenderSerenaMCP(yaml, serenaTool)
		case "agentic-workflows":
			renderer.RenderAgenticWorkflowsMCP(yaml)
		case "safe-outputs":
			// Add safe-outputs MCP server if safe-outputs are configured
			hasSafeOutputs := workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs)
			if hasSafeOutputs {
				renderer.RenderSafeOutputsMCP(yaml)
			}
		case "safe-inputs":
			// Add safe-inputs MCP server if safe-inputs are configured and feature flag is enabled
			hasSafeInputs := workflowData != nil && IsSafeInputsEnabled(workflowData.SafeInputs, workflowData)
			if hasSafeInputs {
				renderer.RenderSafeInputsMCP(yaml, workflowData.SafeInputs, workflowData)
			}
		case "web-fetch":
			renderMCPFetchServerConfig(yaml, "toml", "          ", false, false)
		default:
			// Handle custom MCP tools using shared helper (with adapter for isLast parameter)
			HandleCustomMCPToolInSwitch(yaml, toolName, expandedTools, false, func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
				return e.renderCodexMCPConfigWithContext(yaml, toolName, toolConfig, workflowData)
			})
		}
	}

	// Append custom config if provided
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Config != "" {
		yaml.WriteString("          \n")
		yaml.WriteString("          # Custom configuration\n")
		// Write the custom config line by line with proper indentation
		configLines := strings.Split(workflowData.EngineConfig.Config, "\n")
		for _, line := range configLines {
			if strings.TrimSpace(line) != "" {
				yaml.WriteString("          " + line + "\n")
			} else {
				yaml.WriteString("          \n")
			}
		}
	}

	yaml.WriteString("          EOF\n")
}

// renderCodexMCPConfigWithContext generates custom MCP server configuration for a single tool in codex workflow config.toml
// This version includes workflowData to determine if localhost URLs should be rewritten
func (e *CodexEngine) renderCodexMCPConfigWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, workflowData *WorkflowData) error {
	yaml.WriteString("          \n")
	fmt.Fprintf(yaml, "          [mcp_servers.%s]\n", toolName)

	// Determine if localhost URLs should be rewritten to host.docker.internal
	// This is needed when firewall is enabled (agent is not disabled)
	rewriteLocalhost := workflowData != nil && (workflowData.SandboxConfig == nil ||
		workflowData.SandboxConfig.Agent == nil ||
		!workflowData.SandboxConfig.Agent.Disabled)

	// Use the shared MCP config renderer with TOML format
	renderer := MCPConfigRenderer{
		IndentLevel:              "          ",
		Format:                   "toml",
		RewriteLocalhostToDocker: rewriteLocalhost,
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		return err
	}

	return nil
}
