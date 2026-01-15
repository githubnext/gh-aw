package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/types"
)

var mcpLog = logger.New("workflow:mcp-config")

// WellKnownContainer represents a container configuration for a well-known command
type WellKnownContainer struct {
	Image      string // Container image (e.g., "node:lts-alpine")
	Entrypoint string // Entrypoint command (e.g., "npx")
}

// getWellKnownContainer returns the appropriate container configuration for well-known commands
// This enables automatic containerization of stdio MCP servers based on their command
func getWellKnownContainer(command string) *WellKnownContainer {
	wellKnownContainers := map[string]*WellKnownContainer{
		"npx": {
			Image:      constants.DefaultNodeAlpineLTSImage,
			Entrypoint: "npx",
		},
		"uvx": {
			Image:      constants.DefaultPythonAlpineLTSImage,
			Entrypoint: "uvx",
		},
	}

	return wellKnownContainers[command]
}

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses Docker container to launch Playwright MCP for consistent browser environment
// This is a shared function used by both Claude and Custom engines
func renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	mcpLog.Print("Rendering Playwright MCP configuration")
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

	// Add volume mount for workspace access
	yaml.WriteString("                \"mounts\": [\"${{ github.workspace }}:${{ github.workspace }}:rw\"]\n")

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// BuiltinMCPServerOptions contains the options for rendering a built-in MCP server block
type BuiltinMCPServerOptions struct {
	Yaml                 *strings.Builder
	ServerID             string
	Command              string
	Args                 []string
	EnvVars              []string
	IsLast               bool
	IncludeCopilotFields bool
}

// renderBuiltinMCPServerBlock is a shared helper function that renders MCP server configuration blocks
// for built-in servers (Safe Outputs and Agentic Workflows) with consistent formatting.
// This eliminates code duplication between renderSafeOutputsMCPConfigWithOptions and
// renderAgenticWorkflowsMCPConfigWithOptions by extracting the common YAML generation pattern.
func renderBuiltinMCPServerBlock(opts BuiltinMCPServerOptions) {
	opts.Yaml.WriteString("              \"" + opts.ServerID + "\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if opts.IncludeCopilotFields {
		opts.Yaml.WriteString("                \"type\": \"stdio\",\n")
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

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

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
// This is a shared function used by both Claude and Custom engines
func renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	mcpLog.Print("Rendering Safe Outputs MCP configuration")
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
		"GITHUB_REF_NAME",
		"GITHUB_HEAD_REF",
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
	yaml.WriteString("                \"mounts\": [\"/opt/gh-aw:/opt/gh-aw:ro\", \"/tmp/gh-aw:/tmp/gh-aw:rw\"],\n")

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
func renderAgenticWorkflowsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	envVars := []string{
		"GITHUB_TOKEN",
	}

	renderBuiltinMCPServerBlock(BuiltinMCPServerOptions{
		Yaml:                 yaml,
		ServerID:             "agentic_workflows",
		Command:              "gh",
		Args:                 []string{"aw", "mcp-server"},
		EnvVars:              envVars,
		IsLast:               isLast,
		IncludeCopilotFields: includeCopilotFields,
	})
}

// renderPlaywrightMCPConfigTOML generates the Playwright MCP server configuration in TOML format for Codex
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypointArgs, mounts, and args fields.
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

// renderCustomMCPConfigWrapper generates custom MCP server configuration wrapper
// This is a shared function used by both Claude and Custom engines
func renderCustomMCPConfigWrapper(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	mcpLog.Printf("Rendering custom MCP config wrapper for tool: %s", toolName)
	fmt.Fprintf(yaml, "              \"%s\": {\n", toolName)

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel: "                ",
		Format:      "json",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}

// renderCustomMCPConfigWrapperWithContext generates custom MCP server configuration wrapper with workflow context
// This version includes workflowData to determine if localhost URLs should be rewritten
func renderCustomMCPConfigWrapperWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool, workflowData *WorkflowData) error {
	mcpLog.Printf("Rendering custom MCP config wrapper with context for tool: %s", toolName)
	fmt.Fprintf(yaml, "              \"%s\": {\n", toolName)

	// Determine if localhost URLs should be rewritten to host.docker.internal
	// This is needed when firewall is enabled (agent is not disabled)
	rewriteLocalhost := workflowData != nil && (workflowData.SandboxConfig == nil ||
		workflowData.SandboxConfig.Agent == nil ||
		!workflowData.SandboxConfig.Agent.Disabled)

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel:              "                ",
		Format:                   "json",
		RewriteLocalhostToDocker: rewriteLocalhost,
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}

// MCPConfigRenderer contains configuration options for rendering MCP config
type MCPConfigRenderer struct {
	// IndentLevel controls the indentation level for properties (e.g., "                " for JSON, "          " for TOML)
	IndentLevel string
	// Format specifies the output format ("json" for JSON-like, "toml" for TOML-like)
	Format string
	// RequiresCopilotFields indicates if the engine requires "type" and "tools" fields (true for copilot engine)
	RequiresCopilotFields bool
	// RewriteLocalhostToDocker indicates if localhost URLs should be rewritten to host.docker.internal
	// This is needed when the agent runs inside a firewall container and needs to access MCP servers on the host
	RewriteLocalhostToDocker bool
}

// rewriteLocalhostToDockerHost rewrites localhost URLs to use host.docker.internal
// This is necessary when MCP servers run on the host machine but are accessed from within
// a Docker container (e.g., when firewall/sandbox is enabled)
func rewriteLocalhostToDockerHost(url string) string {
	// Define the localhost patterns to replace and their docker equivalents
	// Each pattern is a (prefix, replacement) pair
	replacements := []struct {
		prefix      string
		replacement string
	}{
		{"http://localhost", "http://host.docker.internal"},
		{"https://localhost", "https://host.docker.internal"},
		{"http://127.0.0.1", "http://host.docker.internal"},
		{"https://127.0.0.1", "https://host.docker.internal"},
	}

	for _, r := range replacements {
		if strings.HasPrefix(url, r.prefix) {
			newURL := r.replacement + url[len(r.prefix):]
			mcpLog.Printf("Rewriting localhost URL for Docker access: %s -> %s", url, newURL)
			return newURL
		}
	}

	return url
}

// renderSharedMCPConfig generates MCP server configuration for a single tool using shared logic
// This function handles the common logic for rendering MCP configurations across different engines
func renderSharedMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, renderer MCPConfigRenderer) error {
	mcpLog.Printf("Rendering MCP config for tool: %s, format: %s", toolName, renderer.Format)

	// Get MCP configuration in the new format
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		mcpLog.Printf("Failed to parse MCP config for tool %s: %v", toolName, err)
		return fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}

	// Extract secrets from headers for HTTP MCP tools (copilot engine only)
	var headerSecrets map[string]string
	if mcpConfig.Type == "http" && renderer.RequiresCopilotFields {
		headerSecrets = ExtractSecretsFromMap(mcpConfig.Headers)
	}

	// Determine properties based on type
	var propertyOrder []string
	mcpType := mcpConfig.Type

	switch mcpType {
	case "stdio":
		if renderer.Format == "toml" {
			propertyOrder = []string{"command", "args", "env", "proxy-args", "registry"}
		} else {
			// JSON format - include copilot fields if required
			propertyOrder = []string{"type", "command", "tools", "args", "env", "proxy-args", "registry"}
		}
	case "http":
		if renderer.Format == "toml" {
			// TOML format for HTTP MCP servers uses url and http_headers
			propertyOrder = []string{"url", "http_headers"}
		} else {
			// JSON format - include copilot fields if required
			if renderer.RequiresCopilotFields {
				// For HTTP MCP with secrets in headers, env passthrough is needed
				if len(headerSecrets) > 0 {
					propertyOrder = []string{"type", "url", "headers", "tools", "env"}
				} else {
					propertyOrder = []string{"type", "url", "headers", "tools"}
				}
			} else {
				propertyOrder = []string{"type", "url", "headers"}
			}
		}
	default:
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio, http", toolName, mcpType)))
		return nil
	}

	// Find which properties actually exist in this config
	var existingProperties []string
	for _, prop := range propertyOrder {
		switch prop {
		case "type":
			// Include type field only for engines that require copilot fields
			existingProperties = append(existingProperties, prop)
		case "tools":
			// Include tools field only for engines that require copilot fields
			if renderer.RequiresCopilotFields {
				existingProperties = append(existingProperties, prop)
			}
		case "command":
			if mcpConfig.Command != "" {
				existingProperties = append(existingProperties, prop)
			}
		case "args":
			if len(mcpConfig.Args) > 0 {
				existingProperties = append(existingProperties, prop)
			}
		case "env":
			// Include env if there are existing env vars OR if there are header secrets to passthrough
			if len(mcpConfig.Env) > 0 || len(headerSecrets) > 0 {
				existingProperties = append(existingProperties, prop)
			}
		case "url":
			if mcpConfig.URL != "" {
				existingProperties = append(existingProperties, prop)
			}
		case "headers":
			if len(mcpConfig.Headers) > 0 {
				existingProperties = append(existingProperties, prop)
			}
		case "http_headers":
			if len(mcpConfig.Headers) > 0 {
				existingProperties = append(existingProperties, prop)
			}
		case "proxy-args":
			if len(mcpConfig.ProxyArgs) > 0 {
				existingProperties = append(existingProperties, prop)
			}
		case "registry":
			if mcpConfig.Registry != "" {
				existingProperties = append(existingProperties, prop)
			}
		}
	}

	// If no valid properties exist, skip rendering
	if len(existingProperties) == 0 {
		return nil
	}

	// Render properties based on format
	for propIndex, property := range existingProperties {
		isLast := propIndex == len(existingProperties)-1

		switch property {
		case "type":
			// Render type field for JSON format (copilot engine)
			comma := ","
			if isLast {
				comma = ""
			}
			// Type field - per MCP Gateway Specification v1.0.0
			// Use "stdio" for containerized servers, "http" for HTTP servers
			typeValue := mcpConfig.Type
			fmt.Fprintf(yaml, "%s\"type\": \"%s\"%s\n", renderer.IndentLevel, typeValue, comma)
		case "tools":
			// Render tools field for JSON format (copilot engine) - default to all tools
			comma := ","
			if isLast {
				comma = ""
			}
			// Check if allowed tools are specified, otherwise default to "*"
			if len(mcpConfig.Allowed) > 0 {
				fmt.Fprintf(yaml, "%s\"tools\": [\n", renderer.IndentLevel)
				for toolIndex, tool := range mcpConfig.Allowed {
					toolComma := ","
					if toolIndex == len(mcpConfig.Allowed)-1 {
						toolComma = ""
					}
					fmt.Fprintf(yaml, "%s  \"%s\"%s\n", renderer.IndentLevel, tool, toolComma)
				}
				fmt.Fprintf(yaml, "%s]%s\n", renderer.IndentLevel, comma)
			} else {
				fmt.Fprintf(yaml, "%s\"tools\": [\n", renderer.IndentLevel)
				fmt.Fprintf(yaml, "%s  \"*\"\n", renderer.IndentLevel)
				fmt.Fprintf(yaml, "%s]%s\n", renderer.IndentLevel, comma)
			}
		case "command":
			if renderer.Format == "toml" {
				fmt.Fprintf(yaml, "%scommand = \"%s\"\n", renderer.IndentLevel, mcpConfig.Command)
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"command\": \"%s\"%s\n", renderer.IndentLevel, mcpConfig.Command, comma)
			}
		case "args":
			if renderer.Format == "toml" {
				fmt.Fprintf(yaml, "%sargs = [\n", renderer.IndentLevel)
				for _, arg := range mcpConfig.Args {
					fmt.Fprintf(yaml, "%s  \"%s\",\n", renderer.IndentLevel, arg)
				}
				fmt.Fprintf(yaml, "%s]\n", renderer.IndentLevel)
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"args\": [\n", renderer.IndentLevel)
				for argIndex, arg := range mcpConfig.Args {
					argComma := ","
					if argIndex == len(mcpConfig.Args)-1 {
						argComma = ""
					}
					fmt.Fprintf(yaml, "%s  \"%s\"%s\n", renderer.IndentLevel, arg, argComma)
				}
				fmt.Fprintf(yaml, "%s]%s\n", renderer.IndentLevel, comma)
			}
		case "env":
			if renderer.Format == "toml" {
				fmt.Fprintf(yaml, "%senv = { ", renderer.IndentLevel)
				envKeys := make([]string, 0, len(mcpConfig.Env))
				for key := range mcpConfig.Env {
					envKeys = append(envKeys, key)
				}
				sort.Strings(envKeys)
				for i, envKey := range envKeys {
					if i > 0 {
						yaml.WriteString(", ")
					}
					fmt.Fprintf(yaml, "\"%s\" = \"%s\"", envKey, mcpConfig.Env[envKey])
				}
				yaml.WriteString(" }\n")
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"env\": {\n", renderer.IndentLevel)

				// CWE-190: Allocation Size Overflow Prevention
				// Instead of pre-calculating capacity (len(mcpConfig.Env)+len(headerSecrets)),
				// which could overflow if the maps are extremely large, we let Go's append
				// handle capacity growth automatically. This is safe and efficient for
				// environment variable maps which are typically small in practice.
				var envKeys []string
				for key := range mcpConfig.Env {
					envKeys = append(envKeys, key)
				}
				// Add header secrets for passthrough (copilot only)
				for varName := range headerSecrets {
					// Only add if not already in env
					if _, exists := mcpConfig.Env[varName]; !exists {
						envKeys = append(envKeys, varName)
					}
				}
				sort.Strings(envKeys)

				for envIndex, envKey := range envKeys {
					envComma := ","
					if envIndex == len(envKeys)-1 {
						envComma = ""
					}

					// Check if this is a header secret (needs passthrough)
					if _, isHeaderSecret := headerSecrets[envKey]; isHeaderSecret && renderer.RequiresCopilotFields {
						// Use passthrough syntax: "VAR_NAME": "\\${VAR_NAME}"
						fmt.Fprintf(yaml, "%s  \"%s\": \"\\${%s}\"%s\n", renderer.IndentLevel, envKey, envKey, envComma)
					} else {
						// Use existing env value
						fmt.Fprintf(yaml, "%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, envKey, mcpConfig.Env[envKey], envComma)
					}
				}
				fmt.Fprintf(yaml, "%s}%s\n", renderer.IndentLevel, comma)
			}
		case "url":
			// Rewrite localhost URLs to host.docker.internal when running inside firewall container
			// This allows MCP servers running on the host to be accessed from the container
			urlValue := mcpConfig.URL
			if renderer.RewriteLocalhostToDocker {
				urlValue = rewriteLocalhostToDockerHost(urlValue)
			}
			if renderer.Format == "toml" {
				fmt.Fprintf(yaml, "%surl = \"%s\"\n", renderer.IndentLevel, urlValue)
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"url\": \"%s\"%s\n", renderer.IndentLevel, urlValue, comma)
			}
		case "http_headers":
			// TOML format for HTTP headers (Codex style)
			if len(mcpConfig.Headers) > 0 {
				fmt.Fprintf(yaml, "%shttp_headers = { ", renderer.IndentLevel)
				headerKeys := make([]string, 0, len(mcpConfig.Headers))
				for key := range mcpConfig.Headers {
					headerKeys = append(headerKeys, key)
				}
				sort.Strings(headerKeys)
				for i, headerKey := range headerKeys {
					if i > 0 {
						yaml.WriteString(", ")
					}
					fmt.Fprintf(yaml, "\"%s\" = \"%s\"", headerKey, mcpConfig.Headers[headerKey])
				}
				yaml.WriteString(" }\n")
			}
		case "headers":
			comma := ","
			if isLast {
				comma = ""
			}
			fmt.Fprintf(yaml, "%s\"headers\": {\n", renderer.IndentLevel)
			headerKeys := make([]string, 0, len(mcpConfig.Headers))
			for key := range mcpConfig.Headers {
				headerKeys = append(headerKeys, key)
			}
			sort.Strings(headerKeys)
			for headerIndex, headerKey := range headerKeys {
				headerComma := ","
				if headerIndex == len(headerKeys)-1 {
					headerComma = ""
				}

				// Replace secret expressions with env var references for copilot
				headerValue := mcpConfig.Headers[headerKey]
				if renderer.RequiresCopilotFields && len(headerSecrets) > 0 {
					headerValue = ReplaceSecretsWithEnvVars(headerValue, headerSecrets)
				}

				fmt.Fprintf(yaml, "%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, headerKey, headerValue, headerComma)
			}
			fmt.Fprintf(yaml, "%s}%s\n", renderer.IndentLevel, comma)
		case "proxy-args":
			if renderer.Format == "toml" {
				fmt.Fprintf(yaml, "%sproxy_args = [\n", renderer.IndentLevel)
				for _, arg := range mcpConfig.ProxyArgs {
					fmt.Fprintf(yaml, "%s  \"%s\",\n", renderer.IndentLevel, arg)
				}
				fmt.Fprintf(yaml, "%s]\n", renderer.IndentLevel)
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"proxy-args\": [\n", renderer.IndentLevel)
				for argIndex, arg := range mcpConfig.ProxyArgs {
					argComma := ","
					if argIndex == len(mcpConfig.ProxyArgs)-1 {
						argComma = ""
					}
					fmt.Fprintf(yaml, "%s  \"%s\"%s\n", renderer.IndentLevel, arg, argComma)
				}
				fmt.Fprintf(yaml, "%s]%s\n", renderer.IndentLevel, comma)
			}
		case "registry":
			if renderer.Format == "toml" {
				fmt.Fprintf(yaml, "%sregistry = \"%s\"\n", renderer.IndentLevel, mcpConfig.Registry)
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"registry\": \"%s\"%s\n", renderer.IndentLevel, mcpConfig.Registry, comma)
			}
		}
	}

	return nil
}

// ToolConfig represents a tool configuration interface for type safety
type ToolConfig interface {
	GetString(key string) (string, bool)
	GetStringArray(key string) ([]string, bool)
	GetStringMap(key string) (map[string]string, bool)
	GetAny(key string) (any, bool)
}

// MapToolConfig implements ToolConfig for map[string]any
type MapToolConfig map[string]any

func (m MapToolConfig) GetString(key string) (string, bool) {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

func (m MapToolConfig) GetStringArray(key string) ([]string, bool) {
	if value, exists := m[key]; exists {
		if arr, ok := value.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result, true
		}
		if arr, ok := value.([]string); ok {
			return arr, true
		}
	}
	return nil, false
}

func (m MapToolConfig) GetStringMap(key string) (map[string]string, bool) {
	if value, exists := m[key]; exists {
		if mapVal, ok := value.(map[string]any); ok {
			result := make(map[string]string)
			for k, v := range mapVal {
				if str, ok := v.(string); ok {
					result[k] = str
				}
			}
			return result, true
		}
		if mapVal, ok := value.(map[string]string); ok {
			return mapVal, true
		}
	}
	return nil, false
}

func (m MapToolConfig) GetAny(key string) (any, bool) {
	value, exists := m[key]
	return value, exists
}

// collectHTTPMCPHeaderSecrets collects all secrets from HTTP MCP tool headers
// Returns a map of environment variable names to their secret expressions
func collectHTTPMCPHeaderSecrets(tools map[string]any) map[string]string {
	allSecrets := make(map[string]string)

	for toolName, toolValue := range tools {
		// Check if this is an MCP tool configuration
		if toolConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(toolConfig); hasMcp && mcpType == "http" {
				// Extract MCP config to get headers
				if mcpConfig, err := getMCPConfig(toolConfig, toolName); err == nil {
					secrets := ExtractSecretsFromMap(mcpConfig.Headers)
					for varName, expr := range secrets {
						allSecrets[varName] = expr
					}
				}
			}
		}
	}

	return allSecrets
}

// getMCPConfig extracts MCP configuration from a tool config and returns a structured MCPServerConfig
func getMCPConfig(toolConfig map[string]any, toolName string) (*parser.MCPServerConfig, error) {
	mcpLog.Printf("Extracting MCP config for tool: %s", toolName)

	config := MapToolConfig(toolConfig)
	result := &parser.MCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Env:     make(map[string]string),
			Headers: make(map[string]string),
		},
		Name: toolName,
	}

	// Validate known properties - fail if unknown properties are found
	knownProperties := map[string]bool{
		"type":           true,
		"mode":           true, // Added for MCPServerConfig struct
		"command":        true,
		"container":      true,
		"version":        true,
		"args":           true,
		"entrypoint":     true,
		"entrypointArgs": true,
		"mounts":         true,
		"env":            true,
		"proxy-args":     true,
		"url":            true,
		"headers":        true,
		"registry":       true,
		"allowed":        true,
		"toolsets":       true, // Added for MCPServerConfig struct
	}

	for key := range toolConfig {
		if !knownProperties[key] {
			mcpLog.Printf("Unknown property '%s' in MCP config for tool '%s'", key, toolName)
			// Build list of valid properties
			validProps := []string{}
			for prop := range knownProperties {
				validProps = append(validProps, prop)
			}
			sort.Strings(validProps)
			return nil, fmt.Errorf(
				"unknown property '%s' in MCP configuration for tool '%s'. Valid properties are: %s. "+
					"Example:\n"+
					"mcp-servers:\n"+
					"  %s:\n"+
					"    command: \"npx @my/tool\"\n"+
					"    args: [\"--port\", \"3000\"]",
				key, toolName, strings.Join(validProps, ", "), toolName)
		}
	}

	// Infer type from fields if not explicitly provided
	if typeStr, hasType := config.GetString("type"); hasType {
		mcpLog.Printf("MCP type explicitly set to: %s", typeStr)
		// Normalize "local" to "stdio"
		if typeStr == "local" {
			result.Type = "stdio"
		} else {
			result.Type = typeStr
		}
	} else {
		mcpLog.Print("No explicit MCP type, inferring from fields")
		// Infer type from presence of fields
		if _, hasURL := config.GetString("url"); hasURL {
			result.Type = "http"
			mcpLog.Printf("Inferred MCP type as http (has url field)")
		} else if _, hasCommand := config.GetString("command"); hasCommand {
			result.Type = "stdio"
			mcpLog.Printf("Inferred MCP type as stdio (has command field)")
		} else if _, hasContainer := config.GetString("container"); hasContainer {
			result.Type = "stdio"
			mcpLog.Printf("Inferred MCP type as stdio (has container field)")
		} else {
			mcpLog.Printf("Unable to determine MCP type for tool '%s': missing type, url, command, or container", toolName)
			return nil, fmt.Errorf(
				"unable to determine MCP type for tool '%s': missing type, url, command, or container. "+
					"Must specify one of: 'type' (stdio/http), 'url' (for HTTP MCP), 'command' (for command-based), or 'container' (for Docker-based). "+
					"Example:\n"+
					"mcp-servers:\n"+
					"  %s:\n"+
					"    command: \"npx @my/tool\"\n"+
					"    args: [\"--port\", \"3000\"]",
				toolName, toolName,
			)
		}
	}

	// Extract common fields (available for both stdio and http)
	if registry, hasRegistry := config.GetString("registry"); hasRegistry {
		result.Registry = registry
	}

	// Extract fields based on type
	mcpLog.Printf("Extracting fields for MCP type: %s", result.Type)
	switch result.Type {
	case "stdio":
		if command, hasCommand := config.GetString("command"); hasCommand {
			result.Command = command
		}
		if container, hasContainer := config.GetString("container"); hasContainer {
			result.Container = container
		}
		if version, hasVersion := config.GetString("version"); hasVersion {
			result.Version = version
		}
		if args, hasArgs := config.GetStringArray("args"); hasArgs {
			result.Args = args
		}
		if entrypoint, hasEntrypoint := config.GetString("entrypoint"); hasEntrypoint {
			result.Entrypoint = entrypoint
		}
		if entrypointArgs, hasEntrypointArgs := config.GetStringArray("entrypointArgs"); hasEntrypointArgs {
			result.EntrypointArgs = entrypointArgs
		}
		if mounts, hasMounts := config.GetStringArray("mounts"); hasMounts {
			result.Mounts = mounts
		}
		if env, hasEnv := config.GetStringMap("env"); hasEnv {
			result.Env = env
		}
		if proxyArgs, hasProxyArgs := config.GetStringArray("proxy-args"); hasProxyArgs {
			result.ProxyArgs = proxyArgs
		}
	case "http":
		if url, hasURL := config.GetString("url"); hasURL {
			result.URL = url
		} else {
			mcpLog.Printf("HTTP MCP tool '%s' missing required 'url' field", toolName)
			return nil, fmt.Errorf(
				"http MCP tool '%s' missing required 'url' field. HTTP MCP servers must specify a URL endpoint. "+
					"Example:\n"+
					"mcp-servers:\n"+
					"  %s:\n"+
					"    type: http\n"+
					"    url: \"https://api.example.com/mcp\"\n"+
					"    headers:\n"+
					"      Authorization: \"Bearer ${{ secrets.API_KEY }}\"",
				toolName, toolName,
			)
		}
		if headers, hasHeaders := config.GetStringMap("headers"); hasHeaders {
			result.Headers = headers
		}
	default:
		mcpLog.Printf("Unsupported MCP type '%s' for tool '%s'", result.Type, toolName)
		return nil, fmt.Errorf(
			"unsupported MCP type '%s' for tool '%s'. Valid types are: stdio, http. "+
				"Example:\n"+
				"mcp-servers:\n"+
				"  %s:\n"+
				"    type: stdio\n"+
				"    command: \"npx @my/tool\"\n"+
				"    args: [\"--port\", \"3000\"]",
			result.Type, toolName, toolName)
	}

	// Extract allowed tools
	if allowed, hasAllowed := config.GetStringArray("allowed"); hasAllowed {
		result.Allowed = allowed
	}

	// Automatically assign well-known containers for stdio MCP servers based on command
	// This ensures all stdio servers work with the MCP Gateway which requires containerization
	if result.Type == "stdio" && result.Container == "" && result.Command != "" {
		containerConfig := getWellKnownContainer(result.Command)
		if containerConfig != nil {
			mcpLog.Printf("Auto-assigning container for command '%s': %s", result.Command, containerConfig.Image)
			result.Container = containerConfig.Image
			result.Entrypoint = containerConfig.Entrypoint
			// Move command to entrypointArgs and preserve existing args after it
			if result.Command != "" {
				result.EntrypointArgs = append([]string{result.Command}, result.Args...)
				result.Args = nil // Clear args since they're now in entrypointArgs
			}
			result.Command = "" // Clear command since it's now the entrypoint
		}
	}

	// Handle container transformation for stdio type
	if result.Type == "stdio" && result.Container != "" {
		// Save user-provided args before transforming
		userProvidedArgs := result.Args
		entrypoint := result.Entrypoint
		entrypointArgs := result.EntrypointArgs
		mounts := result.Mounts

		// Transform container field to docker command and args
		result.Command = "docker"
		result.Args = []string{"run", "--rm", "-i"}

		// Add environment variables as -e flags (sorted for deterministic output)
		envKeys := make([]string, 0, len(result.Env))
		for envKey := range result.Env {
			envKeys = append(envKeys, envKey)
		}
		sort.Strings(envKeys)
		for _, envKey := range envKeys {
			result.Args = append(result.Args, "-e", envKey)
		}

		// Add volume mounts if configured (sorted for deterministic output)
		if len(mounts) > 0 {
			sortedMounts := make([]string, len(mounts))
			copy(sortedMounts, mounts)
			sort.Strings(sortedMounts)
			for _, mount := range sortedMounts {
				result.Args = append(result.Args, "-v", mount)
			}
		}

		// Insert user-provided args (e.g., additional docker flags) before the container image
		if len(userProvidedArgs) > 0 {
			result.Args = append(result.Args, userProvidedArgs...)
		}

		// Add entrypoint override if specified
		if entrypoint != "" {
			result.Args = append(result.Args, "--entrypoint", entrypoint)
		}

		// Build container image with version if provided
		containerImage := result.Container
		if result.Version != "" {
			containerImage = containerImage + ":" + result.Version
		}

		// Add the container image
		result.Args = append(result.Args, containerImage)

		// Add entrypoint args after the container image
		if len(entrypointArgs) > 0 {
			result.Args = append(result.Args, entrypointArgs...)
		}

		// Clear the container, version, entrypoint, entrypointArgs, and mounts fields since they're now part of the command
		result.Container = ""
		result.Version = ""
		result.Entrypoint = ""
		result.EntrypointArgs = nil
		result.Mounts = nil
	}

	return result, nil
}

// hasMCPConfig checks if a tool configuration has MCP configuration
func hasMCPConfig(toolConfig map[string]any) (bool, string) {
	// Check for direct type field
	if mcpType, hasType := toolConfig["type"]; hasType {
		if typeStr, ok := mcpType.(string); ok && parser.IsMCPType(typeStr) {
			// Normalize "local" to "stdio" for consistency
			if typeStr == "local" {
				return true, "stdio"
			}
			return true, typeStr
		}
	}

	// Infer type from presence of fields (same logic as getMCPConfig)
	if _, hasURL := toolConfig["url"]; hasURL {
		return true, "http"
	} else if _, hasCommand := toolConfig["command"]; hasCommand {
		return true, "stdio"
	} else if _, hasContainer := toolConfig["container"]; hasContainer {
		return true, "stdio"
	}

	return false, ""
}
