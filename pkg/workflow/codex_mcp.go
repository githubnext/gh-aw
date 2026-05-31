package workflow

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var codexMCPLog = logger.New("workflow:codex_mcp")

const (
	codexOpenAIProxyProviderID   = "openai-proxy"
	codexOpenAIProxyProviderName = "OpenAI AWF proxy"
)

// RenderMCPConfig generates MCP server configuration for Codex
func (e *CodexEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	if codexMCPLog.Enabled() {
		codexMCPLog.Printf("Rendering MCP config for Codex: mcp_tools=%v, tool_count=%d", mcpTools, len(tools))
	}

	delimiter := e.beginCodexMCPConfigHeredoc(yaml, workflowData)
	e.renderCodexMCPConfigPreamble(yaml, tools, mcpTools)
	e.renderCodexMCPToolSections(yaml, e.expandNeutralToolsToCodexToolsFromMap(tools), mcpTools, workflowData)
	e.appendCodexIndentedEngineConfig(yaml, workflowData)
	yaml.WriteString("          " + delimiter + "\n")
	if err := e.renderCodexGatewayJSONConfig(yaml, tools, mcpTools, workflowData); err != nil {
		return err
	}
	e.renderCodexWritableConfigSync(yaml, tools, mcpTools, workflowData)
	return nil
}

func (e *CodexEngine) beginCodexMCPConfigHeredoc(yaml *strings.Builder, workflowData *WorkflowData) string {
	delimiter := GenerateHeredocDelimiterFromSeed("MCP_CONFIG", workflowData.FrontmatterHash)
	yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/mcp-config/config.toml\" << " + delimiter + "\n")
	return delimiter
}

func (e *CodexEngine) renderCodexMCPConfigPreamble(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	yaml.WriteString("          [history]\n")
	yaml.WriteString("          persistence = \"none\"\n")
	e.renderShellEnvironmentPolicy(yaml, tools, mcpTools)
}

func (e *CodexEngine) renderCodexMCPToolSections(yaml *strings.Builder, expandedTools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	for _, toolName := range mcpTools {
		e.renderCodexMCPToolSection(yaml, toolName, expandedTools, workflowData)
	}
}

func (e *CodexEngine) renderCodexMCPToolSection(yaml *strings.Builder, toolName string, expandedTools map[string]any, workflowData *WorkflowData) {
	renderer := e.newCodexUnifiedMCPRenderer(workflowData, false)
	switch toolName {
	case "github":
		renderer.RenderGitHubMCP(yaml, expandedTools["github"], workflowData)
	case "playwright":
		renderer.RenderPlaywrightMCP(yaml, expandedTools["playwright"])
	case "agentic-workflows":
		renderer.RenderAgenticWorkflowsMCP(yaml)
	case "safe-outputs":
		if workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs) {
			renderer.RenderSafeOutputsMCP(yaml, workflowData)
		}
	case "mcp-scripts":
		if workflowData != nil && IsMCPScriptsEnabled(workflowData.MCPScripts) {
			renderer.RenderMCPScriptsMCP(yaml, workflowData.MCPScripts, workflowData)
		}
	default:
		HandleCustomMCPToolInSwitch(yaml, toolName, expandedTools, false, func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return e.renderCodexMCPConfigWithContext(yaml, toolName, toolConfig, workflowData)
		})
	}
}

func (e *CodexEngine) newCodexUnifiedMCPRenderer(workflowData *WorkflowData, isLast bool) *MCPConfigRendererUnified {
	return NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields:   false,
		InlineArgs:             false,
		Format:                 "toml",
		IsLast:                 isLast,
		ActionMode:             GetActionModeFromWorkflowData(workflowData),
		WriteSinkGuardPolicies: deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
	})
}

func (e *CodexEngine) appendCodexIndentedEngineConfig(yaml *strings.Builder, workflowData *WorkflowData) {
	if workflowData.EngineConfig == nil || workflowData.EngineConfig.Config == "" {
		return
	}
	yaml.WriteString("          \n")
	yaml.WriteString("          # Custom configuration\n")
	for _, line := range strings.Split(workflowData.EngineConfig.Config, "\n") {
		if strings.TrimSpace(line) != "" {
			yaml.WriteString("          " + line + "\n")
		} else {
			yaml.WriteString("          \n")
		}
	}
}

func (e *CodexEngine) renderCodexGatewayJSONConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	yaml.WriteString("          \n")
	yaml.WriteString("          # Generate JSON config for MCP gateway\n")
	return renderStandardJSONMCPConfig(yaml, renderStandardJSONMCPConfigOptions{
		tools:        tools,
		mcpTools:     mcpTools,
		workflowData: workflowData,
		configPath:   "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json",
		renderCustom: func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
			return e.renderCodexJSONMCPConfigWithContext(yaml, toolName, toolConfig, isLast, workflowData)
		},
	})
}

func (e *CodexEngine) renderCodexWritableConfigSync(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	yaml.WriteString("          \n")
	yaml.WriteString("          # Sync converter output to writable CODEX_HOME for Codex\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-config\n")
	delimiter := GenerateHeredocDelimiterFromSeed("CODEX_SHELL_POLICY", workflowData.FrontmatterHash)
	yaml.WriteString("          cat > \"/tmp/gh-aw/mcp-config/config.toml\" << " + delimiter + "\n")
	if isFirewallEnabled(workflowData) {
		e.renderOpenAIProxyProviderToml(yaml, "          ")
	}
	e.renderShellEnvironmentPolicyToml(yaml, tools, mcpTools, "          ")
	yaml.WriteString("          " + delimiter + "\n")
	if isFirewallEnabled(workflowData) {
		e.renderAppendConvertedConfigWithoutOpenAIProxy(yaml)
	} else {
		yaml.WriteString("          cat \"${RUNNER_TEMP}/gh-aw/mcp-config/config.toml\" >> \"/tmp/gh-aw/mcp-config/config.toml\"\n")
	}
	e.appendCodexWritableEngineConfig(yaml, workflowData)
	yaml.WriteString("          chmod 600 \"/tmp/gh-aw/mcp-config/config.toml\"\n")
	yaml.WriteString("          mkdir -p \"${CODEX_HOME}\"\n")
	yaml.WriteString("          if [ \"/tmp/gh-aw/mcp-config/config.toml\" != \"${CODEX_HOME}/config.toml\" ]; then cp \"/tmp/gh-aw/mcp-config/config.toml\" \"${CODEX_HOME}/config.toml\"; fi\n")
	yaml.WriteString("          chmod 600 \"${CODEX_HOME}/config.toml\"\n")
}

func (e *CodexEngine) appendCodexWritableEngineConfig(yaml *strings.Builder, workflowData *WorkflowData) {
	if workflowData.EngineConfig == nil || strings.TrimSpace(workflowData.EngineConfig.Config) == "" {
		return
	}
	delimiter := GenerateHeredocDelimiterFromSeed("CODEX_CUSTOM_CONFIG", workflowData.FrontmatterHash)
	yaml.WriteString("          \n")
	yaml.WriteString("          # Append engine-level custom Codex config\n")
	yaml.WriteString("          cat >> \"/tmp/gh-aw/mcp-config/config.toml\" << " + delimiter + "\n")
	yaml.WriteString(workflowData.EngineConfig.Config)
	if !strings.HasSuffix(workflowData.EngineConfig.Config, "\n") {
		yaml.WriteString("\n")
	}
	yaml.WriteString("          " + delimiter + "\n")
}

func (e *CodexEngine) renderOpenAIProxyProviderToml(yaml *strings.Builder, indent string) {
	yaml.WriteString("\n")
	yaml.WriteString(indent + "model_provider = \"" + codexOpenAIProxyProviderID + "\"\n")
	yaml.WriteString("\n")
	yaml.WriteString(indent + "[model_providers." + codexOpenAIProxyProviderID + "]\n")
	yaml.WriteString(indent + "name = \"" + codexOpenAIProxyProviderName + "\"\n")
	yaml.WriteString(indent + "base_url = \"" + e.getOpenAIProxyProviderBaseURL() + "\"\n")
	yaml.WriteString(indent + "env_key = \"OPENAI_API_KEY\"\n")
	yaml.WriteString(indent + "supports_websockets = false\n")
}

func (e *CodexEngine) getOpenAIProxyProviderBaseURL() string {
	// AWF exposes the OpenAI-compatible provider on the shared OpenAI/Responses
	// gateway port (10000).
	return "http://" + net.JoinHostPort(constants.AWFAPIProxyContainerIP, strconv.Itoa(constants.ClaudeLLMGatewayPort))
}

func (e *CodexEngine) renderAppendConvertedConfigWithoutOpenAIProxy(yaml *strings.Builder) {
	yaml.WriteString("          awk '\n")
	yaml.WriteString("            BEGIN { skip_openai_proxy = 0 }\n")
	yaml.WriteString("            /^[[:space:]]*model_provider[[:space:]]*=/ { next }\n")
	yaml.WriteString("            /^\\[model_providers\\.openai-proxy\\][[:space:]]*$/ { skip_openai_proxy = 1; next }\n")
	yaml.WriteString("            /^\\[/ { skip_openai_proxy = 0 }\n")
	yaml.WriteString("            !skip_openai_proxy { print }\n")
	yaml.WriteString("          ' \"${RUNNER_TEMP}/gh-aw/mcp-config/config.toml\" >> \"/tmp/gh-aw/mcp-config/config.toml\"\n")
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
