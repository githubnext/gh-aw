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
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpRendererLog = logger.New("workflow:mcp_renderer")

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

	// Build the JSON configuration in a separate builder for validation
	var configBuilder strings.Builder
	configBuilder.WriteString("          {\n")
	configBuilder.WriteString("            \"mcpServers\": {\n")

	// Filter tools if needed (e.g., Copilot filters out cache-memory)
	var filteredTools []string
	for _, toolName := range mcpTools {
		if options.FilterTool != nil && !options.FilterTool(toolName) {
			mcpRendererLog.Printf("Filtering out MCP tool: %s", toolName)
			continue
		}
		filteredTools = append(filteredTools, toolName)
	}

	mcpRendererLog.Printf("Rendering %d MCP tools after filtering", len(filteredTools))

	// Process each MCP tool
	totalServers := len(filteredTools)
	serverCount := 0

	for _, toolName := range filteredTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			options.Renderers.RenderGitHub(&configBuilder, githubTool, isLast, workflowData)
		case "playwright":
			playwrightTool := tools["playwright"]
			options.Renderers.RenderPlaywright(&configBuilder, playwrightTool, isLast)
		case "cache-memory":
			options.Renderers.RenderCacheMemory(&configBuilder, isLast, workflowData)
		case "agentic-workflows":
			options.Renderers.RenderAgenticWorkflows(&configBuilder, isLast)
		case "safe-outputs":
			options.Renderers.RenderSafeOutputs(&configBuilder, isLast, workflowData)
		case "mcp-scripts":
			if options.Renderers.RenderMCPScripts != nil {
				options.Renderers.RenderMCPScripts(&configBuilder, workflowData.MCPScripts, isLast)
			}
		default:
			// Handle custom MCP tools using shared helper
			HandleCustomMCPToolInSwitch(&configBuilder, toolName, tools, isLast, options.Renderers.RenderCustomMCPConfig)
		}
	}

	// Write config file footer - but don't add newline yet if we need to add gateway
	if options.GatewayConfig != nil {
		configBuilder.WriteString("            },\n")
		// Add gateway section (needed for gateway to process)
		// Per MCP Gateway Specification v1.0.0 section 4.2, use "${VARIABLE_NAME}" syntax for variable expressions
		configBuilder.WriteString("            \"gateway\": {\n")
		// Port as unquoted variable - shell expands to integer (e.g., 8080) for valid JSON
		fmt.Fprintf(&configBuilder, "              \"port\": $MCP_GATEWAY_PORT,\n")
		fmt.Fprintf(&configBuilder, "              \"domain\": \"%s\",\n", options.GatewayConfig.Domain)
		fmt.Fprintf(&configBuilder, "              \"apiKey\": \"%s\"", options.GatewayConfig.APIKey)

		// Add optional fields if specified (apiKey always precedes them without a trailing comma)
		if options.GatewayConfig.PayloadDir != "" {
			fmt.Fprintf(&configBuilder, ",\n              \"payloadDir\": \"%s\"", options.GatewayConfig.PayloadDir)
		}
		if len(options.GatewayConfig.TrustedBots) > 0 {
			configBuilder.WriteString(",\n              \"trustedBots\": [")
			for i, bot := range options.GatewayConfig.TrustedBots {
				if i > 0 {
					configBuilder.WriteString(", ")
				}
				fmt.Fprintf(&configBuilder, "%q", bot)
			}
			configBuilder.WriteString("]")
		}
		if options.GatewayConfig.KeepaliveInterval != 0 {
			fmt.Fprintf(&configBuilder, ",\n              \"keepaliveInterval\": %d", options.GatewayConfig.KeepaliveInterval)
		}
		// When OTLP tracing is configured, add the opentelemetry section directly to the
		// gateway config. The endpoint is written as a literal value (including GitHub Actions
		// expressions such as ${{ secrets.X }} which GH Actions expands at runtime).
		// Headers must be a JSON object per the gateway schema, so they are written via
		// the bash variable ${_GH_AW_OTLP_HEADERS_JSON} which is set by the preamble
		// generated just before the heredoc. traceId and spanId use ${VARIABLE_NAME}
		// expressions which bash expands from the GITHUB_ENV values set by actions/setup.
		// Per MCP Gateway Specification §4.1.3.6 and the opentelemetryConfig schema.
		if options.GatewayConfig.OTLPEndpoint != "" {
			configBuilder.WriteString(",\n              \"opentelemetry\": {\n")
			fmt.Fprintf(&configBuilder, "                \"endpoint\": %q,\n", options.GatewayConfig.OTLPEndpoint)
			if options.GatewayConfig.OTLPHeaders != "" {
				// ${_GH_AW_OTLP_HEADERS_JSON} is set by the bash preamble emitted below
				// (before the heredoc) which converts OTEL_EXPORTER_OTLP_HEADERS from
				// "key=value,..." format to a proper JSON object as required by the schema.
				configBuilder.WriteString("                \"headers\": ${_GH_AW_OTLP_HEADERS_JSON},\n")
			}
			configBuilder.WriteString("                \"traceId\": \"${GITHUB_AW_OTEL_TRACE_ID}\",\n")
			configBuilder.WriteString("                \"spanId\": \"${GITHUB_AW_OTEL_PARENT_SPAN_ID}\"\n")
			configBuilder.WriteString("              }")
		}
		configBuilder.WriteString("\n")
		configBuilder.WriteString("            }\n")
	} else {
		configBuilder.WriteString("            }\n")
	}

	configBuilder.WriteString("          }\n")

	// Get the generated configuration
	generatedConfig := configBuilder.String()

	delimiter := GenerateHeredocDelimiterFromSeed("MCP_CONFIG", workflowData.FrontmatterHash)
	// When OTLP headers are configured, emit a bash preamble that converts
	// OTEL_EXPORTER_OTLP_HEADERS (injected as a job-level env var by injectOTLPConfig)
	// from "key=value,..." OTLP string format to a JSON object. The heredoc then
	// references ${_GH_AW_OTLP_HEADERS_JSON} which bash expands to the object value,
	// satisfying the gateway schema which requires headers to be a JSON object.
	if options.GatewayConfig != nil && options.GatewayConfig.OTLPEndpoint != "" && options.GatewayConfig.OTLPHeaders != "" {
		yaml.WriteString("          # Convert OTLP headers (key=value,...) to JSON object for gateway config\n")
		yaml.WriteString("          _GH_AW_OTLP_HEADERS_JSON=\"{}\"\n")
		yaml.WriteString("          if [ -n \"${OTEL_EXPORTER_OTLP_HEADERS:-}\" ]; then\n")
		yaml.WriteString("            _GH_AW_OTLP_HEADERS_JSON=$(node -e " +
			`'const h=process.env["OTEL_EXPORTER_OTLP_HEADERS"]||"";const o={};` +
			`h.split(",").forEach(function(p){const i=p.indexOf("=");` +
			`if(i>0)o[p.slice(0,i).trim()]=p.slice(i+1).trim();});` +
			`console.log(JSON.stringify(o));'` +
			" 2>/dev/null || echo \"{}\")\n")
		yaml.WriteString("          fi\n")
	}
	// Write the configuration to the YAML output
	yaml.WriteString("          cat << " + delimiter + " | bash \"${RUNNER_TEMP}/gh-aw/actions/start_mcp_gateway.sh\"\n")
	yaml.WriteString(generatedConfig)
	yaml.WriteString("          " + delimiter + "\n")

	// Note: Post-EOF commands are no longer needed since we pipe directly to the gateway script
	return nil
}
