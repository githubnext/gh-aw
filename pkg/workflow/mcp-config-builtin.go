package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpBuiltinLog = logger.New("workflow:mcp-config-builtin")

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
// This is a shared function used by both Claude and Custom engines
func renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	mcpBuiltinLog.Print("Rendering Safe Outputs MCP configuration")
	renderSafeOutputsMCPConfigWithOptions(yaml, isLast, false)
}

// renderSafeOutputsMCPConfigWithOptions generates the Safe Outputs MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypoint, entrypointArgs, and mounts fields.
func renderSafeOutputsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	envVars := []string{
		"GH_AW_MCP_LOG_DIR",
		"GH_AW_SAFE_OUTPUTS",
		"GH_AW_SAFE_OUTPUTS_CONFIG_PATH",
		"GH_AW_SAFE_OUTPUTS_TOOLS_PATH",
		"GH_AW_ASSETS_BRANCH",
		"GH_AW_ASSETS_MAX_SIZE_KB",
		"GH_AW_ASSETS_ALLOWED_EXTS",
		"GITHUB_REPOSITORY",
		"GITHUB_SERVER_URL",
		"GITHUB_SHA",
		"GITHUB_WORKSPACE",
		"DEFAULT_BRANCH",
	}

	// Use MCP Gateway spec format with container, entrypoint, entrypointArgs, and mounts
	// This will be transformed to Docker command by getMCPConfig transformation logic
	yaml.WriteString("              \"" + constants.SafeOutputsMCPServerID + "\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// MCP Gateway spec fields for containerized stdio servers
	yaml.WriteString("                \"container\": \"" + constants.DefaultNodeAlpineLTSImage + "\",\n")
	yaml.WriteString("                \"entrypoint\": \"node\",\n")
	yaml.WriteString("                \"entrypointArgs\": [\"/opt/gh-aw/safeoutputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"mounts\": [\"" + constants.DefaultGhAwMount + "\", \"" + constants.DefaultTmpGhAwMount + "\"],\n")

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	// Write environment variables
	yaml.WriteString("                \"env\": {\n")
	for i, envVar := range envVars {
		isLastEnvVar := i == len(envVars)-1
		comma := ""
		if !isLastEnvVar {
			comma = ","
		}

		if includeCopilotFields {
			// Copilot format: backslash-escaped shell variable reference
			yaml.WriteString("                  \"" + envVar + "\": \"\\${" + envVar + "}\"" + comma + "\n")
		} else {
			// Claude/Custom format: direct shell variable reference
			yaml.WriteString("                  \"" + envVar + "\": \"$" + envVar + "\"" + comma + "\n")
		}
	}
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderAgenticWorkflowsMCPConfigWithOptions generates the Agentic Workflows MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypoint, entrypointArgs, and mounts fields.
func renderAgenticWorkflowsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	envVars := []string{
		"GITHUB_TOKEN",
	}

	// Use MCP Gateway spec format with container, entrypoint, entrypointArgs, and mounts
	// The gh-aw binary is mounted from /opt/gh-aw and executed directly inside a minimal Alpine container
	yaml.WriteString("              \"agentic_workflows\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// MCP Gateway spec fields for containerized stdio servers
	yaml.WriteString("                \"container\": \"" + constants.DefaultAlpineImage + "\",\n")
	yaml.WriteString("                \"entrypoint\": \"/opt/gh-aw/gh-aw\",\n")
	yaml.WriteString("                \"entrypointArgs\": [\"mcp-server\"],\n")
	yaml.WriteString("                \"mounts\": [\"" + constants.DefaultGhAwMount + "\"],\n")

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	// Write environment variables
	yaml.WriteString("                \"env\": {\n")
	for i, envVar := range envVars {
		isLastEnvVar := i == len(envVars)-1
		comma := ""
		if !isLastEnvVar {
			comma = ","
		}

		if includeCopilotFields {
			// Copilot format: backslash-escaped shell variable reference
			yaml.WriteString("                  \"" + envVar + "\": \"\\${" + envVar + "}\"" + comma + "\n")
		} else {
			// Claude/Custom format: direct shell variable reference
			yaml.WriteString("                  \"" + envVar + "\": \"$" + envVar + "\"" + comma + "\n")
		}
	}
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderSafeOutputsMCPConfigTOML generates the Safe Outputs MCP server configuration in TOML format for Codex
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypoint, entrypointArgs, and mounts fields.
func renderSafeOutputsMCPConfigTOML(yaml *strings.Builder) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.SafeOutputsMCPServerID + "]\n")
	yaml.WriteString("          container = \"" + constants.DefaultNodeAlpineLTSImage + "\"\n")
	yaml.WriteString("          entrypoint = \"node\"\n")
	yaml.WriteString("          entrypointArgs = [\"/opt/gh-aw/safeoutputs/mcp-server.cjs\"]\n")
	yaml.WriteString("          mounts = [\"" + constants.DefaultGhAwMount + "\", \"" + constants.DefaultTmpGhAwMount + "\"]\n")
	// Use env_vars array to reference environment variables instead of embedding GitHub Actions expressions
	yaml.WriteString("          env_vars = [\"GH_AW_SAFE_OUTPUTS\", \"GH_AW_ASSETS_BRANCH\", \"GH_AW_ASSETS_MAX_SIZE_KB\", \"GH_AW_ASSETS_ALLOWED_EXTS\", \"GITHUB_REPOSITORY\", \"GITHUB_SERVER_URL\", \"GITHUB_SHA\", \"GITHUB_WORKSPACE\", \"DEFAULT_BRANCH\"]\n")
}

// renderAgenticWorkflowsMCPConfigTOML generates the Agentic Workflows MCP server configuration in TOML format for Codex
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypoint, entrypointArgs, and mounts fields.
func renderAgenticWorkflowsMCPConfigTOML(yaml *strings.Builder) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.agentic_workflows]\n")
	yaml.WriteString("          container = \"" + constants.DefaultAlpineImage + "\"\n")
	yaml.WriteString("          entrypoint = \"/opt/gh-aw/gh-aw\"\n")
	yaml.WriteString("          entrypointArgs = [\"mcp-server\"]\n")
	yaml.WriteString("          mounts = [\"" + constants.DefaultGhAwMount + "\"]\n")
	// Use env_vars array to reference environment variables instead of embedding secrets
	yaml.WriteString("          env_vars = [\"GITHUB_TOKEN\"]\n")
}
