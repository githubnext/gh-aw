package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpLog = logger.New("workflow:mcp-config")

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses Docker container to launch Playwright MCP for consistent browser environment
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
		if renderer.Format == "toml" {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio, http", toolName, mcpType)))
		} else {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio, http", toolName, mcpType)))
		}
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
			// For copilot CLI, convert "stdio" to "local"
			typeValue := mcpConfig.Type
			if typeValue == "stdio" && renderer.RequiresCopilotFields {
				typeValue = "local"
			}
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



