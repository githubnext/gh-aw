package workflow

import (
"strings"

"github.com/githubnext/gh-aw/pkg/constants"
)

// mcp_config_builtin_servers.go contains built-in MCP server rendering functions.
// This file handles the rendering of Playwright, Serena, Safe Outputs, and Agentic Workflows MCP servers.

// This is a shared function used by both Claude and Custom engines
func renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	mcpLog.Print("Rendering Playwright MCP configuration")
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightTool, isLast, false, false)
}

// renderPlaywrightMCPConfigWithOptions generates the Playwright MCP server configuration with engine-specific options
// Uses Docker container with the versioned Playwright MCP image for consistent browser environment
func renderPlaywrightMCPConfigWithOptions(yaml *strings.Builder, playwrightTool any, isLast bool, includeCopilotFields bool, inlineArgs bool) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	// Extract all expressions from playwright arguments and replace them with env var references
	expressions := extractExpressionsFromPlaywrightArgs(args.AllowedDomains, customArgs)
	allowedDomains := replaceExpressionsInPlaywrightArgs(args.AllowedDomains, expressions)

	// Also replace expressions in custom args
	if len(customArgs) > 0 {
		customArgs = replaceExpressionsInPlaywrightArgs(customArgs, expressions)
	}

	// Use official Playwright MCP Docker image (no version tag - only one image)
	playwrightImage := "mcr.microsoft.com/playwright/mcp"
	// Use MCP package version from constants for output-dir identification
	_ = "@playwright/mcp@" + args.MCPPackageVersion

	yaml.WriteString("              \"playwright\": {\n")

	// Add type field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"docker\",\n")

	if inlineArgs {
		// Inline format for Copilot
		yaml.WriteString("                \"args\": [\"run\", \"-i\", \"--rm\", \"--init\", \"--network\", \"host\", \"" + playwrightImage + "\", \"--output-dir\", \"/tmp/gh-aw/mcp-logs/playwright\"")
		if len(allowedDomains) > 0 {
			domainsStr := strings.Join(allowedDomains, ";")
			yaml.WriteString(", \"--allowed-hosts\", \"" + domainsStr + "\"")
			yaml.WriteString(", \"--allowed-origins\", \"" + domainsStr + "\"")
		}
		// Append custom args if present
		writeArgsToYAMLInline(yaml, customArgs)
		yaml.WriteString("]")
	} else {
		// Multi-line format for Claude/Custom
		yaml.WriteString("                \"args\": [\n")
		yaml.WriteString("                  \"run\",\n")
		yaml.WriteString("                  \"-i\",\n")
		yaml.WriteString("                  \"--rm\",\n")
		yaml.WriteString("                  \"--init\",\n")
		yaml.WriteString("                  \"--network\",\n")
		yaml.WriteString("                  \"host\",\n")
		yaml.WriteString("                  \"" + playwrightImage + "\",\n")
		yaml.WriteString("                  \"--output-dir\",\n")
		yaml.WriteString("                  \"/tmp/gh-aw/mcp-logs/playwright\"")
		if len(allowedDomains) > 0 {
			domainsStr := strings.Join(allowedDomains, ";")
			yaml.WriteString(",\n")
			yaml.WriteString("                  \"--allowed-hosts\",\n")
			yaml.WriteString("                  \"" + domainsStr + "\",\n")
			yaml.WriteString("                  \"--allowed-origins\",\n")
			yaml.WriteString("                  \"" + domainsStr + "\"")
		}
		// Append custom args if present
		writeArgsToYAML(yaml, customArgs, "                  ")
		yaml.WriteString("\n")
		yaml.WriteString("                ]")
	}

	// Add tools field for Copilot
	if includeCopilotFields {
		yaml.WriteString(",\n")
		yaml.WriteString("                \"tools\": [\"*\"]")
	}

	yaml.WriteString("\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderSerenaMCPConfigWithOptions generates the Serena MCP server configuration with engine-specific options
func renderSerenaMCPConfigWithOptions(yaml *strings.Builder, serenaTool any, isLast bool, includeCopilotFields bool, inlineArgs bool) {
	customArgs := getSerenaCustomArgs(serenaTool)

	yaml.WriteString("              \"serena\": {\n")

	// Add type field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"uvx\",\n")

	if inlineArgs {
		// Inline format for Copilot
		yaml.WriteString("                \"args\": [\"--from\", \"git+https://github.com/oraios/serena\", \"serena\", \"start-mcp-server\", \"--context\", \"codex\", \"--project\", \"${{ github.workspace }}\"")
		// Append custom args if present
		writeArgsToYAMLInline(yaml, customArgs)
		yaml.WriteString("]")
	} else {
		// Multi-line format for Claude/Custom
		yaml.WriteString("                \"args\": [\n")
		yaml.WriteString("                  \"--from\",\n")
		yaml.WriteString("                  \"git+https://github.com/oraios/serena\",\n")
		yaml.WriteString("                  \"serena\",\n")
		yaml.WriteString("                  \"start-mcp-server\",\n")
		yaml.WriteString("                  \"--context\",\n")
		yaml.WriteString("                  \"codex\",\n")
		yaml.WriteString("                  \"--project\",\n")
		yaml.WriteString("                  \"${{ github.workspace }}\"")
		// Append custom args if present
		writeArgsToYAML(yaml, customArgs, "                  ")
		yaml.WriteString("\n")
		yaml.WriteString("                ]")
	}

	// Add tools field for Copilot
	if includeCopilotFields {
		yaml.WriteString(",\n")
		yaml.WriteString("                \"tools\": [\"*\"]")
	}

	yaml.WriteString("\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderBuiltinMCPServerBlock is a shared helper function that renders MCP server configuration blocks
// for built-in servers (Safe Outputs and Agentic Workflows) with consistent formatting.
// This eliminates code duplication between renderSafeOutputsMCPConfigWithOptions and
// renderAgenticWorkflowsMCPConfigWithOptions by extracting the common YAML generation pattern.
func renderBuiltinMCPServerBlock(yaml *strings.Builder, serverID string, command string, args []string, envVars []string, isLast bool, includeCopilotFields bool) {
	yaml.WriteString("              \"" + serverID + "\": {\n")

	// Add type field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"" + command + "\",\n")

	// Write args array
	yaml.WriteString("                \"args\": [")
	for i, arg := range args {
		if i > 0 {
			yaml.WriteString(", ")
		}
		yaml.WriteString("\"" + arg + "\"")
	}
	yaml.WriteString("],\n")

	// Add tools field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	yaml.WriteString("                \"env\": {\n")

	// Write environment variables with appropriate escaping
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

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
// This is a shared function used by both Claude and Custom engines
func renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	mcpLog.Print("Rendering Safe Outputs MCP configuration")
	renderSafeOutputsMCPConfigWithOptions(yaml, isLast, false)
}

// renderSafeOutputsMCPConfigWithOptions generates the Safe Outputs MCP server configuration with engine-specific options
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

	renderBuiltinMCPServerBlock(
		yaml,
		constants.SafeOutputsMCPServerID,
		"node",
		[]string{"/opt/gh-aw/safeoutputs/mcp-server.cjs"},
		envVars,
		isLast,
		includeCopilotFields,
	)
}

// renderAgenticWorkflowsMCPConfigWithOptions generates the Agentic Workflows MCP server configuration with engine-specific options
func renderAgenticWorkflowsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	envVars := []string{
		"GITHUB_TOKEN",
	}

	renderBuiltinMCPServerBlock(
		yaml,
		"agentic_workflows",
		"gh",
		[]string{"aw", "mcp-server"},
		envVars,
		isLast,
		includeCopilotFields,
	)
}

// renderPlaywrightMCPConfigTOML generates the Playwright MCP server configuration in TOML format for Codex
// Uses Docker container with the versioned Playwright MCP image for consistent browser environment
func renderPlaywrightMCPConfigTOML(yaml *strings.Builder, playwrightTool any) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	// Use official Playwright MCP Docker image (no version tag - only one image)
	playwrightImage := "mcr.microsoft.com/playwright/mcp"

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")
	yaml.WriteString("          command = \"docker\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"run\",\n")
	yaml.WriteString("            \"-i\",\n")
	yaml.WriteString("            \"--rm\",\n")
	yaml.WriteString("            \"--init\",\n")
	yaml.WriteString("            \"--network\",\n")
	yaml.WriteString("            \"host\",\n")
	yaml.WriteString("            \"" + playwrightImage + "\",\n")
	yaml.WriteString("            \"--output-dir\",\n")
	yaml.WriteString("            \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		domainsStr := strings.Join(args.AllowedDomains, ";")
		yaml.WriteString(",\n")
		yaml.WriteString("            \"--allowed-hosts\",\n")
		yaml.WriteString("            \"" + domainsStr + "\",\n")
		yaml.WriteString("            \"--allowed-origins\",\n")
		yaml.WriteString("            \"" + domainsStr + "\"")
	}

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "            ")

	yaml.WriteString("\n")
	yaml.WriteString("          ]\n")
}

// renderSafeOutputsMCPConfigTOML generates the Safe Outputs MCP server configuration in TOML format for Codex
func renderSafeOutputsMCPConfigTOML(yaml *strings.Builder) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.SafeOutputsMCPServerID + "]\n")
	yaml.WriteString("          command = \"node\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"/opt/gh-aw/safeoutputs/mcp-server.cjs\",\n")
	yaml.WriteString("          ]\n")
	// Use env_vars array to reference environment variables instead of embedding GitHub Actions expressions
	yaml.WriteString("          env_vars = [\"GH_AW_SAFE_OUTPUTS\", \"GH_AW_ASSETS_BRANCH\", \"GH_AW_ASSETS_MAX_SIZE_KB\", \"GH_AW_ASSETS_ALLOWED_EXTS\", \"GITHUB_REPOSITORY\", \"GITHUB_SERVER_URL\", \"GITHUB_SHA\", \"GITHUB_WORKSPACE\", \"DEFAULT_BRANCH\"]\n")
}

// renderAgenticWorkflowsMCPConfigTOML generates the Agentic Workflows MCP server configuration in TOML format for Codex
func renderAgenticWorkflowsMCPConfigTOML(yaml *strings.Builder) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.agentic_workflows]\n")
	yaml.WriteString("          command = \"gh\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"aw\",\n")
	yaml.WriteString("            \"mcp-server\",\n")
	yaml.WriteString("          ]\n")
	// Use env_vars array to reference environment variables instead of embedding secrets
	yaml.WriteString("          env_vars = [\"GITHUB_TOKEN\"]\n")
}
