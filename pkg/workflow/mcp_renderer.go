// Package workflow provides YAML rendering for MCP server configurations.
//
// # MCP Configuration Renderer Module
//
// The renderer subsystem is split across focused files for maintainability:
//
//   - mcp_renderer.go         — Factory (NewMCPConfigRenderer), custom-tool switch handler
//     (HandleCustomMCPToolInSwitch), top-level JSON orchestrator (RenderJSONMCPConfig).
//   - mcp_renderer_types.go   — All struct and func-type definitions (MCPRendererOptions,
//     MCPConfigRendererUnified, RenderCustomMCPToolConfigHandler, MCPToolRenderers,
//     JSONMCPConfigOptions, GitHubMCPDockerOptions, GitHubMCPRemoteOptions).
//   - mcp_renderer_github.go  — GitHub MCP rendering: RenderGitHubMCP, renderGitHubTOML,
//     RenderGitHubMCPDockerConfig, RenderGitHubMCPRemoteConfig.
//   - mcp_renderer_builtin.go — Built-in MCP server renderers: Playwright,
//     SafeOutputs, MCPScripts, AgenticWorkflows (JSON + TOML for each).
//   - mcp_renderer_guard.go   — Guard / access-control policy rendering:
//     renderGuardPoliciesJSON, renderGuardPoliciesToml.
//
// All files belong to package workflow — no import path changes required.
//
// Renderer architecture:
// The renderer uses the MCPConfigRendererUnified struct with MCPRendererOptions
// to configure engine-specific behaviors:
//   - IncludeCopilotFields: Add "type" and "tools" fields for Copilot
//   - InlineArgs: Render args inline (Copilot) vs multi-line (Claude/Custom)
//   - Format: "json" for JSON-like or "toml" for TOML-like output
//   - IsLast: Control trailing commas in rendered configuration
//
// Engine-specific rendering:
//   - Copilot: JSON format with "type" and "tools" fields, inline args
//   - Claude: JSON format without Copilot fields, multi-line args
//   - Codex: TOML format for MCP configuration
//   - Custom: Same as Claude (JSON, multi-line args)
//
// Example usage:
//
//	renderer := NewMCPConfigRenderer(MCPRendererOptions{
//	   IncludeCopilotFields: true,
//	   InlineArgs: true,
//	   Format: "json",
//	   IsLast: false,
//	})
//
// renderer.RenderGitHubMCP(yaml, githubTool, workflowData)
package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpRendererLog = logger.New("workflow:mcp_renderer")

// safeMCPServerIDRE matches MCP server IDs that are safe to embed inside an unquoted bash heredoc.
// The pattern allows alphanumeric characters, hyphens, and underscores — the same characters used
// for built-in tool names and typical custom mcp-server keys.
var safeMCPServerIDRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// isSafeMCPServerID reports whether id can be safely embedded inside an unquoted bash heredoc
// without triggering shell expansion.
func isSafeMCPServerID(id string) bool {
	return id != "" && safeMCPServerIDRE.MatchString(id)
}

func durationStringToSeconds(durationValue string) (int, error) {
	parsedDuration, err := time.ParseDuration(durationValue)
	if err != nil {
		return 0, err
	}
	return int(parsedDuration.Round(time.Second) / time.Second), nil
}

// NewMCPConfigRenderer creates a new unified MCP config renderer with the specified options
func NewMCPConfigRenderer(opts MCPRendererOptions) *MCPConfigRendererUnified {
	mcpRendererLog.Printf("Creating MCP renderer: format=%s, copilot_fields=%t, inline_args=%t, is_last=%t",
		opts.Format, opts.IncludeCopilotFields, opts.InlineArgs, opts.IsLast)
	return &MCPConfigRendererUnified{
		options: opts,
	}
}

// HandleCustomMCPToolInSwitch processes custom MCP tools in the default case of a switch statement.
// This shared function extracts the common pattern used across all workflow engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - toolName: The name of the tool being processed
//   - tools: The tools map containing tool configurations (supports both expanded and non-expanded tools)
//   - isLast: Whether this is the last tool in the list
//   - renderFunc: Engine-specific function to render the MCP configuration
//
// Returns:
//   - bool: true if a custom MCP tool was handled, false otherwise
func HandleCustomMCPToolInSwitch(
	yaml *strings.Builder,
	toolName string,
	tools map[string]any,
	isLast bool,
	renderFunc RenderCustomMCPToolConfigHandler,
) bool {
	// Handle custom MCP tools (those with MCP-compatible type)
	if toolConfig, ok := tools[toolName].(map[string]any); ok {
		if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
			if err := renderFunc(yaml, toolName, toolConfig, isLast); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating custom MCP configuration for %s: %v\n", toolName, err)
			}
			return true
		}
	}
	return false
}

// RenderJSONMCPConfig renders MCP configuration in JSON format with the common mcpServers structure.
// This shared function extracts the duplicate pattern from Claude, Copilot, and Custom engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - tools: Map of tool configurations
//   - mcpTools: Ordered list of MCP tool names to render
//   - workflowData: Workflow configuration data
//   - options: JSON MCP config rendering options
func RenderJSONMCPConfig(
	yaml *strings.Builder,
	tools map[string]any,
	mcpTools []string,
	workflowData *WorkflowData,
	options JSONMCPConfigOptions,
) error {
	mcpRendererLog.Printf("Rendering JSON MCP config: %d tools", len(mcpTools))
	var configBuilder strings.Builder
	configBuilder.WriteString("          {\n")
	configBuilder.WriteString("            \"mcpServers\": {\n")
	renderJSONMCPServers(&configBuilder, tools, filterMCPTools(mcpTools, options), workflowData, options)
	if err := renderJSONMCPGateway(&configBuilder, options.GatewayConfig); err != nil {
		return err
	}
	configBuilder.WriteString("          }\n")

	generatedConfig := configBuilder.String()
	delimiter := GenerateHeredocDelimiterFromContent("MCP_CONFIG", generatedConfig)
	yaml.WriteString("          GH_AW_NODE=$(which node 2>/dev/null || command -v node 2>/dev/null || echo node)\n")
	yaml.WriteString("          cat << " + delimiter + " | \"$GH_AW_NODE\" \"${RUNNER_TEMP}/gh-aw/actions/start_mcp_gateway.cjs\"\n")
	yaml.WriteString(generatedConfig)
	yaml.WriteString("          " + delimiter + "\n")
	return nil
}

func filterMCPTools(mcpTools []string, options JSONMCPConfigOptions) []string {
	var filteredTools []string
	for _, toolName := range mcpTools {
		if options.FilterTool != nil && !options.FilterTool(toolName) {
			mcpRendererLog.Printf("Filtering out MCP tool: %s", toolName)
			continue
		}
		filteredTools = append(filteredTools, toolName)
	}
	mcpRendererLog.Printf("Rendering %d MCP tools after filtering", len(filteredTools))
	return filteredTools
}

func renderJSONMCPServers(configBuilder *strings.Builder, tools map[string]any, filteredTools []string, workflowData *WorkflowData, options JSONMCPConfigOptions) {
	totalServers := len(filteredTools)
	serverCount := 0
	for _, toolName := range filteredTools {
		serverCount++
		isLast := serverCount == totalServers
		renderJSONMCPServer(configBuilder, tools, toolName, isLast, workflowData, options)
	}
}

func renderJSONMCPServer(configBuilder *strings.Builder, tools map[string]any, toolName string, isLast bool, workflowData *WorkflowData, options JSONMCPConfigOptions) {
	switch toolName {
	case "github":
		githubTool, _ := tools["github"].(map[string]any)
		options.Renderers.RenderGitHub(configBuilder, githubTool, isLast, workflowData)
	case "playwright":
		options.Renderers.RenderPlaywright(configBuilder, tools["playwright"], isLast)
	case "cache-memory":
		options.Renderers.RenderCacheMemory(configBuilder, isLast, workflowData)
	case "agentic-workflows":
		options.Renderers.RenderAgenticWorkflows(configBuilder, isLast)
	case "safe-outputs":
		options.Renderers.RenderSafeOutputs(configBuilder, isLast, workflowData)
	case "mcp-scripts":
		if options.Renderers.RenderMCPScripts != nil {
			options.Renderers.RenderMCPScripts(configBuilder, workflowData.MCPScripts, isLast)
		}
	default:
		HandleCustomMCPToolInSwitch(configBuilder, toolName, tools, isLast, options.Renderers.RenderCustomMCPConfig)
	}
}

func renderJSONMCPGateway(configBuilder *strings.Builder, gatewayConfig *MCPGatewayRuntimeConfig) error {
	if gatewayConfig == nil {
		configBuilder.WriteString("            }\n")
		return nil
	}
	configBuilder.WriteString("            },\n")
	configBuilder.WriteString("            \"gateway\": {\n")
	fmt.Fprintf(configBuilder, "              \"port\": $MCP_GATEWAY_PORT,\n")
	fmt.Fprintf(configBuilder, "              \"domain\": \"%s\",\n", gatewayConfig.Domain)
	fmt.Fprintf(configBuilder, "              \"apiKey\": \"%s\"", gatewayConfig.APIKey)
	if err := renderJSONMCPGatewayOptionalFields(configBuilder, gatewayConfig); err != nil {
		return err
	}
	configBuilder.WriteString("\n")
	configBuilder.WriteString("            }\n")
	return nil
}

func renderJSONMCPGatewayOptionalFields(configBuilder *strings.Builder, gatewayConfig *MCPGatewayRuntimeConfig) error {
	if gatewayConfig.PayloadDir != "" {
		fmt.Fprintf(configBuilder, ",\n              \"payloadDir\": \"%s\"", gatewayConfig.PayloadDir)
	}
	renderJSONMCPGatewayTrustedBots(configBuilder, gatewayConfig.TrustedBots)
	if gatewayConfig.KeepaliveInterval != 0 {
		fmt.Fprintf(configBuilder, ",\n              \"keepaliveInterval\": %d", gatewayConfig.KeepaliveInterval)
	}
	if gatewayConfig.SessionTimeout != "" {
		fmt.Fprintf(configBuilder, ",\n              \"sessionTimeout\": %q", gatewayConfig.SessionTimeout)
	}
	if gatewayConfig.ToolTimeout != "" {
		toolTimeoutSeconds, err := durationStringToSeconds(gatewayConfig.ToolTimeout)
		if err != nil {
			return fmt.Errorf("failed to parse engine.mcp.tool-timeout %q for gateway.toolTimeout: %w", gatewayConfig.ToolTimeout, err)
		}
		fmt.Fprintf(configBuilder, ",\n              \"toolTimeout\": %d", toolTimeoutSeconds)
	}
	if gatewayConfig.ForcePublicRepos != nil && !*gatewayConfig.ForcePublicRepos {
		configBuilder.WriteString(",\n              \"forcePublicRepos\": false")
	}
	return renderJSONMCPGatewayAdvancedFields(configBuilder, gatewayConfig)
}

func renderJSONMCPGatewayTrustedBots(configBuilder *strings.Builder, trustedBots []string) {
	if len(trustedBots) == 0 {
		return
	}
	configBuilder.WriteString(",\n              \"trustedBots\": [")
	for i, bot := range trustedBots {
		if i > 0 {
			configBuilder.WriteString(", ")
		}
		fmt.Fprintf(configBuilder, "%q", bot)
	}
	configBuilder.WriteString("]")
}

func renderJSONMCPGatewayAdvancedFields(configBuilder *strings.Builder, gatewayConfig *MCPGatewayRuntimeConfig) error {
	if len(gatewayConfig.SinkVisibilityExemptServers) > 0 {
		configBuilder.WriteString(",\n              \"sinkVisibilityExemptServers\": [")
		for i, serverID := range gatewayConfig.SinkVisibilityExemptServers {
			if i > 0 {
				configBuilder.WriteString(", ")
			}
			if !isSafeMCPServerID(serverID) {
				return fmt.Errorf("private-to-public-flows: server ID %q contains characters that are unsafe for shell heredoc emission; IDs must match [A-Za-z0-9_-]+", serverID)
			}
			fmt.Fprintf(configBuilder, "%q", serverID)
		}
		configBuilder.WriteString("]")
	}
	if gatewayConfig.OTLPEndpoint != "" {
		configBuilder.WriteString(",\n              \"opentelemetry\": {\n")
		configBuilder.WriteString("                \"endpoint\": \"${OTEL_EXPORTER_OTLP_ENDPOINT}\",\n")
		configBuilder.WriteString("                \"traceId\": \"${GITHUB_AW_OTEL_TRACE_ID}\",\n")
		configBuilder.WriteString("                \"spanId\": \"${GITHUB_AW_OTEL_PARENT_SPAN_ID}\"\n")
		configBuilder.WriteString("              }")
	}
	return nil
}
