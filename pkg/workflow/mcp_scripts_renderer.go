package workflow

import (
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var mcpScriptsRendererLog = logger.New("workflow:mcp_scripts_renderer")

// collectMCPScriptsSecrets collects all secrets from mcp-scripts configuration
func collectMCPScriptsSecrets(mcpScripts *MCPScriptsConfig) map[string]string {
	secrets := make(map[string]string)

	if mcpScripts == nil {
		mcpScriptsRendererLog.Print("No mcp-scripts configuration provided for secret collection")
		return secrets
	}

	mcpScriptsRendererLog.Printf("Collecting secrets from %d mcp-scripts tools", len(mcpScripts.Tools))

	// Sort tool names for consistent behavior when same env var appears in multiple tools
	toolNames := make([]string, 0, len(mcpScripts.Tools))
	for toolName := range mcpScripts.Tools {
		toolNames = append(toolNames, toolName)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		toolConfig := mcpScripts.Tools[toolName]
		// Sort env var names for consistent order within each tool
		envNames := make([]string, 0, len(toolConfig.Env))
		for envName := range toolConfig.Env {
			envNames = append(envNames, envName)
		}
		sort.Strings(envNames)

		for _, envName := range envNames {
			secrets[envName] = toolConfig.Env[envName]
		}
	}

	mcpScriptsRendererLog.Printf("Collected %d secrets from mcp-scripts configuration", len(secrets))
	return secrets
}

// renderMCPScriptsMCPConfigWithOptions generates the MCP Scripts server configuration with engine-specific options
// Always uses HTTP transport mode
func renderMCPScriptsMCPConfigWithOptions(yaml *strings.Builder, mcpScripts *MCPScriptsConfig, isLast bool, includeCopilotFields bool, workflowData *WorkflowData) {
	mcpScriptsRendererLog.Printf("Rendering MCP Scripts config: includeCopilotFields=%t, isLast=%t",
		includeCopilotFields, isLast)

	yaml.WriteString("              \"" + constants.MCPScriptsMCPServerID.String() + "\": {\n")

	// HTTP transport configuration - server started in separate step
	// Add type field for HTTP (required by MCP specification for HTTP transport)
	yaml.WriteString("                \"type\": \"http\",\n")

	// Determine host based on whether agent is disabled
	host := "host.docker.internal"
	if workflowData != nil && workflowData.SandboxConfig != nil && workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled {
		// When agent is disabled (no firewall), use localhost instead of host.docker.internal
		host = "localhost"
		mcpScriptsRendererLog.Print("Agent disabled, using localhost for MCP Scripts server")
	}

	// HTTP URL using environment variable - NOT escaped so shell expands it before awmg validation
	// Use host.docker.internal to allow access from firewall container (or localhost if agent disabled)
	// Note: awmg validates URL format before variable resolution, so we must expand the port variable
	yaml.WriteString("                \"url\": \"http://" + host + ":$GH_AW_MCP_SCRIPTS_PORT\",\n")

	// Add Authorization header with API key
	yaml.WriteString("                \"headers\": {\n")
	if includeCopilotFields {
		// Copilot format: backslash-escaped shell variable reference
		yaml.WriteString("                  \"Authorization\": \"\\${GH_AW_MCP_SCRIPTS_API_KEY}\"\n")
	} else {
		// Claude/Custom format: direct shell variable reference
		yaml.WriteString("                  \"Authorization\": \"$GH_AW_MCP_SCRIPTS_API_KEY\"\n")
	}
	// Close headers - no trailing comma since this is the last field
	// Note: env block is NOT included for HTTP servers because the old MCP Gateway schema
	// doesn't allow env in httpServerConfig. The variables are resolved via URL templates.
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}
