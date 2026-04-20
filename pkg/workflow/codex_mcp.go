package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var codexMCPLog = logger.New("workflow:codex_mcp")

// RenderMCPConfig generates MCP server configuration for Codex
func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	if codexMCPLog.Enabled() {
		codexMCPLog.Printf("Rendering MCP config for Codex: mcp_tools=%v, tool_count=%d", mcpTools, len(tools))
	}

	// Create unified renderer with Codex-specific options
	// Codex uses TOML format without Copilot-specific fields and multi-line args
	createRenderer := func(isLast bool) *MCPConfigRendererUnified {
		return NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields:   false, // Codex doesn't use "type" and "tools" fields
			InlineArgs:             false, // Codex uses multi-line args format
			Format:                 "toml",
			IsLast:                 isLast,
			ActionMode:             GetActionModeFromWorkflowData(workflowData),
			WriteSinkGuardPolicies: deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
		})
	}

	delimiter := GenerateHeredocDelimiterFromSeed("MCP_CONFIG", workflowData.FrontmatterHash)
	yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/mcp-config/config.toml\" << " + delimiter + "\n")

	// Add history configuration to disable persistence
	yaml.WriteString("          [history]\n")
	yaml.WriteString("          persistence = \"none\"\n")

	// Add shell environment policy to control which environment variables are passed through
	// This is a security feature to prevent accidental exposure of secrets
	e.renderShellEnvironmentPolicy(yaml, tools, mcpTools)

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
		case "agentic-workflows":
			renderer.RenderAgenticWorkflowsMCP(yaml)
		case "safe-outputs":
			// Add safe-outputs MCP server if safe-outputs are configured
			hasSafeOutputs := workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs)
			if hasSafeOutputs {
				renderer.RenderSafeOutputsMCP(yaml, workflowData)
			}
		case "mcp-scripts":
			// Add mcp-scripts MCP server if mcp-scripts are configured and feature flag is enabled
			hasMCPScripts := workflowData != nil && IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData)
			if hasMCPScripts {
				renderer.RenderMCPScriptsMCP(yaml, workflowData.MCPScripts, workflowData)
			}
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
		configLines := strings.SplitSeq(workflowData.EngineConfig.Config, "\n")
		for line := range configLines {
			if strings.TrimSpace(line) != "" {
				yaml.WriteString("          " + line + "\n")
			} else {
				yaml.WriteString("          \n")
			}
		}
	}

	// End the heredoc for config.toml
	yaml.WriteString("          " + delimiter + "\n")

	// Also generate JSON config for MCP gateway
	// Per MCP Gateway Specification v1.0.0 section 4.1, the gateway requires JSON input
	// This JSON config is used by the gateway, while the TOML config above is used by Codex
	yaml.WriteString("          \n")
	yaml.WriteString("          # Generate JSON config for MCP gateway\n")

	// Gateway uses JSON format without Copilot-specific fields and multi-line args
	if err := renderStandardJSONMCPConfig(yaml, tools, mcpTools, workflowData,
		"${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json", false, false,
		func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return e.renderCodexJSONMCPConfigWithContext(yaml, toolName, toolConfig, isLast, workflowData)
		}, nil); err != nil {
		return err
	}

	// start_mcp_gateway.cjs converts the gateway output and writes Codex config to
	// ${RUNNER_TEMP}/gh-aw/mcp-config/config.toml. Codex reads config from
	// $CODEX_HOME/config.toml, so copy the converted config into writable CODEX_HOME
	// and prepend shell policy (converter output does not include this section).
	yaml.WriteString("          \n")
	yaml.WriteString("          # Sync converter output to writable CODEX_HOME for Codex\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-config\n")

	shellPolicyDelimiter := GenerateHeredocDelimiterFromSeed("CODEX_SHELL_POLICY", workflowData.FrontmatterHash)
	yaml.WriteString("          cat > \"/tmp/gh-aw/mcp-config/config.toml\" << " + shellPolicyDelimiter + "\n")
	e.renderShellEnvironmentPolicyToml(yaml, tools, mcpTools, "          ")
	yaml.WriteString("          " + shellPolicyDelimiter + "\n")
	yaml.WriteString("          cat \"${RUNNER_TEMP}/gh-aw/mcp-config/config.toml\" >> \"/tmp/gh-aw/mcp-config/config.toml\"\n")
	if workflowData.EngineConfig != nil && strings.TrimSpace(workflowData.EngineConfig.Config) != "" {
		customConfigDelimiter := GenerateHeredocDelimiterFromSeed("CODEX_CUSTOM_CONFIG", workflowData.FrontmatterHash)
		yaml.WriteString("          \n")
		yaml.WriteString("          # Append engine-level custom Codex config\n")
		yaml.WriteString("          cat >> \"/tmp/gh-aw/mcp-config/config.toml\" << " + customConfigDelimiter + "\n")
		yaml.WriteString(workflowData.EngineConfig.Config)
		if !strings.HasSuffix(workflowData.EngineConfig.Config, "\n") {
			yaml.WriteString("\n")
		}
		yaml.WriteString("          " + customConfigDelimiter + "\n")
	}
	yaml.WriteString("          chmod 600 \"/tmp/gh-aw/mcp-config/config.toml\"\n")
	yaml.WriteString("          mkdir -p \"${CODEX_HOME}\"\n")
	yaml.WriteString("          cp \"/tmp/gh-aw/mcp-config/config.toml\" \"${CODEX_HOME}/config.toml\"\n")
	yaml.WriteString("          chmod 600 \"${CODEX_HOME}/config.toml\"\n")

	return nil
}

// renderCodexMCPConfigWithContext generates custom MCP server configuration for a single tool in codex workflow config.toml
// This version includes workflowData to determine if localhost URLs should be rewritten
func (e *CodexEngine) renderCodexMCPConfigWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, workflowData *WorkflowData) error {
	// Determine if localhost URLs should be rewritten to host.docker.internal
	// This is needed when firewall is enabled (agent is not disabled)
	rewriteLocalhost := shouldRewriteLocalhostToDocker(workflowData)
	codexMCPLog.Printf("Rendering TOML MCP config for custom tool: %s (rewrite_localhost=%v)", toolName, rewriteLocalhost)

	yaml.WriteString("          \n")
	fmt.Fprintf(yaml, "          [mcp_servers.%s]\n", toolName)

	// Use the shared MCP config renderer with TOML format
	renderer := MCPConfigRenderer{
		IndentLevel:              "          ",
		Format:                   "toml",
		RewriteLocalhostToDocker: rewriteLocalhost,
		GuardPolicies:            deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		codexMCPLog.Printf("Failed to render TOML MCP config for tool %s: %v", toolName, err)
		return err
	}

	return nil
}

// renderCodexJSONMCPConfigWithContext generates custom MCP server configuration in JSON format for gateway
// This is used to generate the JSON config file that the MCP gateway reads
func (e *CodexEngine) renderCodexJSONMCPConfigWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool, workflowData *WorkflowData) error {
	// Determine if localhost URLs should be rewritten to host.docker.internal
	rewriteLocalhost := shouldRewriteLocalhostToDocker(workflowData)
	codexMCPLog.Printf("Rendering JSON MCP config for gateway tool: %s (isLast=%v, rewrite_localhost=%v)", toolName, isLast, rewriteLocalhost)

	// Use the shared renderer with JSON format for gateway
	renderer := MCPConfigRenderer{
		Format:                   "json",
		IndentLevel:              "              ",
		RewriteLocalhostToDocker: rewriteLocalhost,
		GuardPolicies:            deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
	}

	yaml.WriteString("              \"" + toolName + "\": {\n")

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		codexMCPLog.Printf("Failed to render JSON MCP config for tool %s: %v", toolName, err)
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}
