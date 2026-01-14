package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpBuiltinLog = logger.New("workflow:mcp-builtin")

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses Docker container to launch Playwright MCP for consistent browser environment
// This is a shared function used by both Claude and Custom engines
func renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	mcpBuiltinLog.Print("Rendering Playwright MCP configuration")
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightTool, isLast, false, false)
}

// renderPlaywrightMCPConfigWithOptions generates the Playwright MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypointArgs, mounts, and args fields.
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

	yaml.WriteString("              \"playwright\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// MCP Gateway spec fields for containerized stdio servers
	yaml.WriteString("                \"container\": \"" + playwrightImage + "\",\n")

	// Docker runtime args (goes before container image in docker run command)
	// These are additional flags for docker run like --init and --network
	dockerArgs := []string{"--init", "--network", "host"}
	if inlineArgs {
		yaml.WriteString("                \"args\": [")
		for i, arg := range dockerArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("],\n")
	} else {
		yaml.WriteString("                \"args\": [\n")
		for i, arg := range dockerArgs {
			yaml.WriteString("                  \"" + arg + "\"")
			if i < len(dockerArgs)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		yaml.WriteString("                ],\n")
	}

	// Build entrypoint args for Playwright MCP server (goes after container image)
	entrypointArgs := []string{"--output-dir", "/tmp/gh-aw/mcp-logs/playwright"}
	if len(allowedDomains) > 0 {
		domainsStr := strings.Join(allowedDomains, ";")
		entrypointArgs = append(entrypointArgs, "--allowed-hosts", domainsStr)
		entrypointArgs = append(entrypointArgs, "--allowed-origins", domainsStr)
	}
	// Append custom args if present
	if len(customArgs) > 0 {
		entrypointArgs = append(entrypointArgs, customArgs...)
	}

	// Render entrypointArgs
	if inlineArgs {
		yaml.WriteString("                \"entrypointArgs\": [")
		for i, arg := range entrypointArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("],\n")
	} else {
		yaml.WriteString("                \"entrypointArgs\": [\n")
		for i, arg := range entrypointArgs {
			yaml.WriteString("                  \"" + arg + "\"")
			if i < len(entrypointArgs)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		yaml.WriteString("                ],\n")
	}

	// Add volume mounts
	yaml.WriteString("                \"mounts\": [\"/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw\"]\n")

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderSerenaMCPConfigWithOptions generates the Serena MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses Docker container format as specified by Serena: ghcr.io/oraios/serena:latest
func renderSerenaMCPConfigWithOptions(yaml *strings.Builder, serenaTool any, isLast bool, includeCopilotFields bool, inlineArgs bool) {
	customArgs := getSerenaCustomArgs(serenaTool)

	yaml.WriteString("              \"serena\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// Use Serena's Docker container image
	yaml.WriteString("                \"container\": \"ghcr.io/oraios/serena:latest\",\n")

	// Docker runtime args (--network host for network access)
	if inlineArgs {
		yaml.WriteString("                \"args\": [\"--network\", \"host\"],\n")
	} else {
		yaml.WriteString("                \"args\": [\n")
		yaml.WriteString("                  \"--network\",\n")
		yaml.WriteString("                  \"host\"\n")
		yaml.WriteString("                ],\n")
	}

	// Serena entrypoint
	yaml.WriteString("                \"entrypoint\": \"serena\",\n")

	// Entrypoint args for Serena MCP server
	if inlineArgs {
		yaml.WriteString("                \"entrypointArgs\": [\"start-mcp-server\", \"--context\", \"codex\", \"--project\", \"${{ github.workspace }}\"")
		// Append custom args if present
		writeArgsToYAMLInline(yaml, customArgs)
		yaml.WriteString("],\n")
	} else {
		yaml.WriteString("                \"entrypointArgs\": [\n")
		yaml.WriteString("                  \"start-mcp-server\",\n")
		yaml.WriteString("                  \"--context\",\n")
		yaml.WriteString("                  \"codex\",\n")
		yaml.WriteString("                  \"--project\",\n")
		yaml.WriteString("                  \"${{ github.workspace }}\"")
		// Append custom args if present
		writeArgsToYAML(yaml, customArgs, "                  ")
		yaml.WriteString("\n")
		yaml.WriteString("                ],\n")
	}

	// Add volume mounts for workspace access
	yaml.WriteString("                \"mounts\": [\"${{ github.workspace }}:${{ github.workspace }}:ro\"]\n")

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
func renderBuiltinMCPServerBlock(opts BuiltinMCPServerOptions) {
	opts.Yaml.WriteString("              \"" + opts.ServerID + "\": {\n")

	// Add type field for Copilot (use "local" for command-based MCP servers)
	if opts.IncludeCopilotFields {
		opts.Yaml.WriteString("                \"type\": \"local\",\n")
	}

	opts.Yaml.WriteString("                \"command\": \"" + opts.Command + "\",\n")

	// Write args array
	opts.Yaml.WriteString("                \"args\": [")
	for i, arg := range opts.Args {
		if i > 0 {
			opts.Yaml.WriteString(", ")
		}
		opts.Yaml.WriteString("\"" + arg + "\"")
	}
	opts.Yaml.WriteString("],\n")

	// Add tools field for Copilot (defaults to all tools)
	if opts.IncludeCopilotFields {
		opts.Yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	opts.Yaml.WriteString("                \"env\": {\n")

	// Write environment variables with appropriate escaping
	for i, envVar := range opts.EnvVars {
		isLastEnvVar := i == len(opts.EnvVars)-1
		comma := ""
		if !isLastEnvVar {
			comma = ","
		}

		if opts.IncludeCopilotFields {
			// Copilot format: backslash-escaped shell variable reference
			opts.Yaml.WriteString("                  \"" + envVar + "\": \"\\${" + envVar + "}\"" + comma + "\n")
		} else {
			// Claude/Custom format: direct shell variable reference
			opts.Yaml.WriteString("                  \"" + envVar + "\": \"$" + envVar + "\"" + comma + "\n")
		}
	}

	opts.Yaml.WriteString("                }\n")

	if opts.IsLast {
		opts.Yaml.WriteString("              }\n")
	} else {
		opts.Yaml.WriteString("              },\n")
	}
}

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
// Uses shell command to launch Node.js MCP server
// This is a shared function used by both Claude and Custom engines
func renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	renderSafeOutputsMCPConfigWithOptions(yaml, isLast, false)
}

// renderSafeOutputsMCPConfigWithOptions generates the Safe Outputs MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses shell command to launch Node.js MCP server with environment variable passthrough.
func renderSafeOutputsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	args := []string{
		"/opt/gh-aw/safeoutputs/mcp-server.cjs",
	}

	envVars := []string{
		"GH_AW_SAFE_OUTPUTS",
		"GH_AW_ASSETS_BRANCH",
		"GH_AW_ASSETS_MAX_SIZE_KB",
		"GH_AW_ASSETS_ALLOWED_EXTS",
		"GITHUB_REPOSITORY",
		"GITHUB_SERVER_URL",
		"GITHUB_SHA",
		"GITHUB_WORKSPACE",
		"DEFAULT_BRANCH",
	}

	renderBuiltinMCPServerBlock(BuiltinMCPServerOptions{
		Yaml:                 yaml,
		ServerID:             constants.SafeOutputsMCPServerID,
		Command:              "node",
		Args:                 args,
		EnvVars:              envVars,
		IsLast:               isLast,
		IncludeCopilotFields: includeCopilotFields,
	})
}

// renderAgenticWorkflowsMCPConfigWithOptions generates the Agentic Workflows MCP server configuration with engine-specific options
func renderAgenticWorkflowsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	args := []string{
		"aw",
		"mcp-server",
	}

	envVars := []string{
		"GITHUB_TOKEN",
	}

	renderBuiltinMCPServerBlock(BuiltinMCPServerOptions{
		Yaml:                 yaml,
		ServerID:             "agentic_workflows",
		Command:              "gh",
		Args:                 args,
		EnvVars:              envVars,
		IsLast:               isLast,
		IncludeCopilotFields: includeCopilotFields,
	})
}

// renderPlaywrightMCPConfigTOML generates the Playwright MCP server configuration in TOML format for Codex
func renderPlaywrightMCPConfigTOML(yaml *strings.Builder, playwrightTool any) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	// Use official Playwright MCP Docker image (no version tag - only one image)
	playwrightImage := "mcr.microsoft.com/playwright/mcp"

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")
	yaml.WriteString("          container = \"" + playwrightImage + "\"\n")

	// Docker runtime args (goes before container image in docker run command)
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"--init\",\n")
	yaml.WriteString("            \"--network\",\n")
	yaml.WriteString("            \"host\",\n")
	yaml.WriteString("          ]\n")

	// Entrypoint args for Playwright MCP server (goes after container image)
	yaml.WriteString("          entrypointArgs = [\n")
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

	// Add volume mounts
	yaml.WriteString("          mounts = [\"/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw\"]\n")
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
	yaml.WriteString("          mounts = [\"/opt/gh-aw:/opt/gh-aw:ro\", \"/tmp/gh-aw:/tmp/gh-aw:rw\"]\n")
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
