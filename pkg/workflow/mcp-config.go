package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var mcpLog = logger.New("workflow:mcp-config")

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
// This is a shared function used by both Claude and Custom engines
func renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	mcpLog.Print("Rendering Playwright MCP configuration")
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightTool, isLast, false, false)
}

// renderPlaywrightMCPConfigWithOptions generates the Playwright MCP server configuration with engine-specific options
func renderPlaywrightMCPConfigWithOptions(yaml *strings.Builder, playwrightTool any, isLast bool, includeCopilotFields bool, inlineArgs bool) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	// Extract secrets from allowed domains and replace them with env var references
	secrets := extractSecretsFromAllowedDomains(args.AllowedDomains)
	allowedDomains := replaceSecretsInAllowedDomains(args.AllowedDomains, secrets)

	// Determine version to use - respect version configuration if provided
	playwrightPackage := "@playwright/mcp@latest"
	if includeCopilotFields && args.ImageVersion != "" && args.ImageVersion != "latest" {
		playwrightPackage = "@playwright/mcp@" + args.ImageVersion
	}

	yaml.WriteString("              \"playwright\": {\n")

	// Add type field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"npx\",\n")

	if inlineArgs {
		// Inline format for Copilot
		yaml.WriteString("                \"args\": [\"" + playwrightPackage + "\", \"--output-dir\", \"/tmp/gh-aw/mcp-logs/playwright\"")
		if len(allowedDomains) > 0 {
			yaml.WriteString(", \"--allowed-origins\", \"" + strings.Join(allowedDomains, ";") + "\"")
		}
		// Append custom args if present
		writeArgsToYAMLInline(yaml, customArgs)
		yaml.WriteString("]")
	} else {
		// Multi-line format for Claude/Custom
		yaml.WriteString("                \"args\": [\n")
		yaml.WriteString("                  \"" + playwrightPackage + "\",\n")
		yaml.WriteString("                  \"--output-dir\",\n")
		yaml.WriteString("                  \"/tmp/gh-aw/mcp-logs/playwright\"")
		if len(allowedDomains) > 0 {
			yaml.WriteString(",\n")
			yaml.WriteString("                  \"--allowed-origins\",\n")
			yaml.WriteString("                  \"" + strings.Join(allowedDomains, ";") + "\"")
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

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
// This is a shared function used by both Claude and Custom engines
func renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	mcpLog.Print("Rendering Safe Outputs MCP configuration")
	renderSafeOutputsMCPConfigWithOptions(yaml, isLast, false)
}

// renderSafeOutputsMCPConfigWithOptions generates the Safe Outputs MCP server configuration with engine-specific options
func renderSafeOutputsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	yaml.WriteString("              \"" + constants.SafeOutputsMCPServerID + "\": {\n")

	// Add type field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"node\",\n")
	yaml.WriteString("                \"args\": [\"/tmp/gh-aw/safeoutputs/mcp-server.cjs\"],\n")

	// Add tools field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	yaml.WriteString("                \"env\": {\n")

	// Use shell environment variables instead of GitHub Actions expressions to prevent template injection
	// For both Copilot and Claude/Custom engines, reference shell env vars
	// The actual GitHub expressions are set in the step's env: block
	if includeCopilotFields {
		yaml.WriteString("                  \"GH_AW_SAFE_OUTPUTS\": \"\\${GH_AW_SAFE_OUTPUTS}\",\n")
		yaml.WriteString("                  \"GH_AW_SAFE_OUTPUTS_CONFIG\": \"\\${GH_AW_SAFE_OUTPUTS_CONFIG}\",\n")
		yaml.WriteString("                  \"GH_AW_ASSETS_BRANCH\": \"\\${GH_AW_ASSETS_BRANCH}\",\n")
		yaml.WriteString("                  \"GH_AW_ASSETS_MAX_SIZE_KB\": \"\\${GH_AW_ASSETS_MAX_SIZE_KB}\",\n")
		yaml.WriteString("                  \"GH_AW_ASSETS_ALLOWED_EXTS\": \"\\${GH_AW_ASSETS_ALLOWED_EXTS}\",\n")
		yaml.WriteString("                  \"GITHUB_REPOSITORY\": \"\\${GITHUB_REPOSITORY}\",\n")
		yaml.WriteString("                  \"GITHUB_SERVER_URL\": \"\\${GITHUB_SERVER_URL}\"\n")
	} else {
		yaml.WriteString("                  \"GH_AW_SAFE_OUTPUTS\": \"$GH_AW_SAFE_OUTPUTS\",\n")
		yaml.WriteString("                  \"GH_AW_SAFE_OUTPUTS_CONFIG\": $GH_AW_SAFE_OUTPUTS_CONFIG,\n")
		yaml.WriteString("                  \"GH_AW_ASSETS_BRANCH\": \"$GH_AW_ASSETS_BRANCH\",\n")
		yaml.WriteString("                  \"GH_AW_ASSETS_MAX_SIZE_KB\": \"$GH_AW_ASSETS_MAX_SIZE_KB\",\n")
		yaml.WriteString("                  \"GH_AW_ASSETS_ALLOWED_EXTS\": \"$GH_AW_ASSETS_ALLOWED_EXTS\",\n")
		yaml.WriteString("                  \"GITHUB_REPOSITORY\": \"$GITHUB_REPOSITORY\",\n")
		yaml.WriteString("                  \"GITHUB_SERVER_URL\": \"$GITHUB_SERVER_URL\"\n")
	}

	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderAgenticWorkflowsMCPConfig generates the Agentic Workflows MCP server configuration
// This is a shared function used by both Claude and Custom engines
func renderAgenticWorkflowsMCPConfig(yaml *strings.Builder, isLast bool) {
	renderAgenticWorkflowsMCPConfigWithOptions(yaml, isLast, false)
}

// renderAgenticWorkflowsMCPConfigWithOptions generates the Agentic Workflows MCP server configuration with engine-specific options
func renderAgenticWorkflowsMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool) {
	yaml.WriteString("              \"agentic_workflows\": {\n")

	// Add type field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"gh\",\n")
	yaml.WriteString("                \"args\": [\"aw\", \"mcp-server\"],\n")

	// Add tools field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	yaml.WriteString("                \"env\": {\n")

	// Use shell environment variables instead of GitHub Actions expressions to prevent template injection
	// For both Copilot and Claude/Custom engines, reference shell env vars
	// The actual GitHub expressions are set in the step's env: block
	if includeCopilotFields {
		yaml.WriteString("                  \"GITHUB_TOKEN\": \"\\${GITHUB_TOKEN}\"\n")
	} else {
		yaml.WriteString("                  \"GITHUB_TOKEN\": \"$GITHUB_TOKEN\"\n")
	}

	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightMCPConfigTOML generates the Playwright MCP server configuration in TOML format for Codex
func renderPlaywrightMCPConfigTOML(yaml *strings.Builder, playwrightTool any) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")
	yaml.WriteString("          command = \"npx\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"@playwright/mcp@latest\",\n")
	yaml.WriteString("            \"--output-dir\",\n")
	yaml.WriteString("            \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("            \"--allowed-origins\",\n")
		yaml.WriteString("            \"" + strings.Join(args.AllowedDomains, ";") + "\"")
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
	yaml.WriteString("            \"/tmp/gh-aw/safeoutputs/mcp-server.cjs\",\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          env = { \"GH_AW_SAFE_OUTPUTS\" = \"${{ env.GH_AW_SAFE_OUTPUTS }}\", \"GH_AW_SAFE_OUTPUTS_CONFIG\" = ${{ toJSON(env.GH_AW_SAFE_OUTPUTS_CONFIG) }}, \"GH_AW_ASSETS_BRANCH\" = \"${{ env.GH_AW_ASSETS_BRANCH }}\", \"GH_AW_ASSETS_MAX_SIZE_KB\" = \"${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}\", \"GH_AW_ASSETS_ALLOWED_EXTS\" = \"${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}\", \"GITHUB_REPOSITORY\" = \"${{ github.repository }}\", \"GITHUB_SERVER_URL\" = \"${{ github.server_url }}\" }\n")
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
	yaml.WriteString("          env = { \"GITHUB_TOKEN\" = \"${{ secrets.GITHUB_TOKEN }}\" }\n")
}

// renderCustomMCPConfigWrapper generates custom MCP server configuration wrapper
// This is a shared function used by both Claude and Custom engines
func renderCustomMCPConfigWrapper(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
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

// MCPConfigRenderer contains configuration options for rendering MCP config
type MCPConfigRenderer struct {
	// IndentLevel controls the indentation level for properties (e.g., "                " for JSON, "          " for TOML)
	IndentLevel string
	// Format specifies the output format ("json" for JSON-like, "toml" for TOML-like)
	Format string
	// RequiresCopilotFields indicates if the engine requires "type" and "tools" fields (true for copilot engine)
	RequiresCopilotFields bool
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
		headerSecrets = extractSecretsFromHeaders(mcpConfig.Headers)
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
			// TOML format doesn't support HTTP type in some engines
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has type '%s', but %s only supports 'stdio'. Ignoring this server.", toolName, mcpType, renderer.Format)))
			return nil
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
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio", toolName, mcpType)))
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
			comma := ","
			if isLast {
				comma = ""
			}
			fmt.Fprintf(yaml, "%s\"url\": \"%s\"%s\n", renderer.IndentLevel, mcpConfig.URL, comma)
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
					headerValue = replaceSecretsWithEnvVars(headerValue, headerSecrets)
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

// extractSecretsFromValue extracts GitHub Actions secret expressions from a string value
// Returns a map of environment variable names to their secret expressions
// Example: "${{ secrets.DD_API_KEY }}" -> {"DD_API_KEY": "${{ secrets.DD_API_KEY }}"}
// Example: "${{ secrets.DD_SITE || 'datadoghq.com' }}" -> {"DD_SITE": "${{ secrets.DD_SITE || 'datadoghq.com' }}"}
func extractSecretsFromValue(value string) map[string]string {
	secrets := make(map[string]string)

	// Pattern to match ${{ secrets.VARIABLE_NAME }} or ${{ secrets.VARIABLE_NAME || 'default' }}
	// We need to extract the variable name and the full expression
	start := 0
	for {
		// Find the start of an expression
		startIdx := strings.Index(value[start:], "${{ secrets.")
		if startIdx == -1 {
			break
		}
		startIdx += start

		// Find the end of the expression
		endIdx := strings.Index(value[startIdx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + 2 // Include the closing }}

		// Extract the full expression
		fullExpr := value[startIdx:endIdx]

		// Extract the variable name from "secrets.VARIABLE_NAME" or "secrets.VARIABLE_NAME ||"
		secretsPart := strings.TrimPrefix(fullExpr, "${{ secrets.")
		secretsPart = strings.TrimSuffix(secretsPart, "}}")
		secretsPart = strings.TrimSpace(secretsPart)

		// Find the variable name (everything before space, ||, or end)
		varName := secretsPart
		if spaceIdx := strings.IndexAny(varName, " |"); spaceIdx != -1 {
			varName = varName[:spaceIdx]
		}

		// Store the variable name and full expression
		if varName != "" {
			secrets[varName] = fullExpr
		}

		start = endIdx
	}

	return secrets
}

// extractSecretsFromHeaders extracts all secrets from HTTP MCP headers
// Returns a map of environment variable names to their secret expressions
func extractSecretsFromHeaders(headers map[string]string) map[string]string {
	allSecrets := make(map[string]string)

	for _, headerValue := range headers {
		secrets := extractSecretsFromValue(headerValue)
		for varName, expr := range secrets {
			allSecrets[varName] = expr
		}
	}

	return allSecrets
}

// replaceSecretsWithEnvVars replaces secret expressions in a value with environment variable references
// Example: "${{ secrets.DD_API_KEY }}" -> "\${DD_API_KEY}"
func replaceSecretsWithEnvVars(value string, secrets map[string]string) string {
	result := value
	for varName, secretExpr := range secrets {
		// Replace ${{ secrets.VAR }} with \${VAR} (backslash-escaped for copilot JSON config)
		result = strings.ReplaceAll(result, secretExpr, "\\${"+varName+"}")
	}
	return result
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
					secrets := extractSecretsFromHeaders(mcpConfig.Headers)
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
		Name:    toolName,
		Env:     make(map[string]string),
		Headers: make(map[string]string),
	}

	// Validate known properties - fail if unknown properties are found
	knownProperties := map[string]bool{
		"type":           true,
		"command":        true,
		"container":      true,
		"version":        true,
		"args":           true,
		"entrypointArgs": true,
		"env":            true,
		"proxy-args":     true,
		"url":            true,
		"headers":        true,
		"registry":       true,
		"allowed":        true,
	}

	for key := range toolConfig {
		if !knownProperties[key] {
			return nil, fmt.Errorf("unknown property '%s' in MCP configuration for tool '%s'", key, toolName)
		}
	}

	// Infer type from fields if not explicitly provided
	if typeStr, hasType := config.GetString("type"); hasType {
		// Normalize "local" to "stdio"
		if typeStr == "local" {
			result.Type = "stdio"
		} else {
			result.Type = typeStr
		}
	} else {
		// Infer type from presence of fields
		if _, hasURL := config.GetString("url"); hasURL {
			result.Type = "http"
		} else if _, hasCommand := config.GetString("command"); hasCommand {
			result.Type = "stdio"
		} else if _, hasContainer := config.GetString("container"); hasContainer {
			result.Type = "stdio"
		} else {
			return nil, fmt.Errorf("unable to determine MCP type for tool '%s': missing type, url, command, or container", toolName)
		}
	}

	// Extract common fields (available for both stdio and http)
	if registry, hasRegistry := config.GetString("registry"); hasRegistry {
		result.Registry = registry
	}

	// Extract fields based on type
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
		if entrypointArgs, hasEntrypointArgs := config.GetStringArray("entrypointArgs"); hasEntrypointArgs {
			result.EntrypointArgs = entrypointArgs
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
			return nil, fmt.Errorf("http MCP tool '%s' missing required 'url' field", toolName)
		}
		if headers, hasHeaders := config.GetStringMap("headers"); hasHeaders {
			result.Headers = headers
		}
	default:
		return nil, fmt.Errorf("unsupported MCP type '%s' for tool '%s'", result.Type, toolName)
	}

	// Extract allowed tools
	if allowed, hasAllowed := config.GetStringArray("allowed"); hasAllowed {
		result.Allowed = allowed
	}

	// Handle container transformation for stdio type
	if result.Type == "stdio" && result.Container != "" {
		// Save user-provided args before transforming
		userProvidedArgs := result.Args
		entrypointArgs := result.EntrypointArgs

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

		// Insert user-provided args (e.g., volume mounts) before the container image
		if len(userProvidedArgs) > 0 {
			result.Args = append(result.Args, userProvidedArgs...)
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

		// Clear the container, version, and entrypointArgs fields since they're now part of the command
		result.Container = ""
		result.Version = ""
		result.EntrypointArgs = nil
	}

	return result, nil
}

// isMCPType checks if a type string represents an MCP-compatible type
func isMCPType(typeStr string) bool {
	switch typeStr {
	case "stdio", "http", "local":
		return true
	default:
		return false
	}
}

// hasMCPConfig checks if a tool configuration has MCP configuration
func hasMCPConfig(toolConfig map[string]any) (bool, string) {
	// Check for direct type field
	if mcpType, hasType := toolConfig["type"]; hasType {
		if typeStr, ok := mcpType.(string); ok && isMCPType(typeStr) {
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

// validateMCPConfigs validates all MCP configurations in the tools section using JSON schema
func ValidateMCPConfigs(tools map[string]any) error {
	mcpLog.Printf("Validating MCP configurations for %d tools", len(tools))

	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			// Extract raw MCP configuration (without transformation)
			mcpConfig, err := getRawMCPConfig(config)
			if err != nil {
				mcpLog.Printf("Invalid MCP configuration for tool %s: %v", toolName, err)
				return fmt.Errorf("tool '%s' has invalid MCP configuration: %w", toolName, err)
			}

			// Skip validation if no MCP configuration found
			if len(mcpConfig) == 0 {
				continue
			}

			mcpLog.Printf("Validating MCP requirements for tool: %s", toolName)

			// Validate MCP configuration requirements (before transformation)
			if err := validateMCPRequirements(toolName, mcpConfig, config); err != nil {
				return err
			}
		}
	}

	mcpLog.Print("MCP configuration validation completed successfully")
	return nil
}

// getRawMCPConfig extracts MCP configuration without any transformations for validation
func getRawMCPConfig(toolConfig map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// List of MCP fields that can be direct children of the tool config
	// Note: "args" is NOT included here because it's used for built-in tools (github, playwright)
	// to add custom arguments without triggering custom MCP tool processing logic. Including "args"
	// would incorrectly classify built-in tools as custom MCP tools, changing their processing behavior
	// and causing validation errors.
	mcpFields := []string{"type", "url", "command", "container", "env", "headers"}

	// List of all known tool config fields (not just MCP)
	knownToolFields := map[string]bool{
		"type":            true,
		"url":             true,
		"command":         true,
		"container":       true,
		"env":             true,
		"headers":         true,
		"version":         true,
		"args":            true,
		"entrypointArgs":  true,
		"proxy-args":      true,
		"registry":        true,
		"allowed":         true,
		"mode":            true, // for github tool
		"github-token":    true, // for github tool
		"read-only":       true, // for github tool
		"toolsets":        true, // for github tool
		"id":              true, // for cache-memory (array notation)
		"key":             true, // for cache-memory
		"description":     true, // for cache-memory
		"retention-days":  true, // for cache-memory
		"allowed_domains": true, // for playwright tool
		"allowed-domains": true, // for playwright tool (alternative notation)
	}

	// Check new format: direct fields in tool config
	for _, field := range mcpFields {
		if value, exists := toolConfig[field]; exists {
			result[field] = value
		}
	}

	// Check for unknown fields that might be typos or deprecated (like "network")
	for field := range toolConfig {
		if !knownToolFields[field] {
			return nil, fmt.Errorf("unknown property '%s' in tool configuration", field)
		}
	}

	return result, nil
}

// getTypeString returns a human-readable type name for error messages
func getTypeString(value any) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case int, int64, float64, float32:
		return "number"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case string:
		return "string"
	default:
		// Check if it's any kind of slice/array by examining the type string
		typeStr := fmt.Sprintf("%T", value)
		if strings.HasPrefix(typeStr, "[]") {
			return "array"
		}
		return "unknown"
	}
}

// validateStringProperty validates that a property is a string and returns appropriate error message
func validateStringProperty(toolName, propertyName string, value any, exists bool) error {
	if !exists {
		return fmt.Errorf("tool '%s' mcp configuration missing property '%s'", toolName, propertyName)
	}
	if _, ok := value.(string); !ok {
		actualType := getTypeString(value)
		return fmt.Errorf("tool '%s' mcp configuration '%s' got %s, want string", toolName, propertyName, actualType)
	}
	return nil
}

// validateMCPRequirements validates the specific requirements for MCP configuration
func validateMCPRequirements(toolName string, mcpConfig map[string]any, toolConfig map[string]any) error {
	// Validate 'type' property - allow inference from other fields
	mcpType, hasType := mcpConfig["type"]
	var typeStr string

	if hasType {
		// Explicit type provided
		if err := validateStringProperty(toolName, "type", mcpType, hasType); err != nil {
			return err
		}
		var ok bool
		typeStr, ok = mcpType.(string)
		if !ok {
			return fmt.Errorf("tool '%s' mcp configuration 'type' validation error", toolName)
		}
	} else {
		// Infer type from presence of fields
		if _, hasURL := mcpConfig["url"]; hasURL {
			typeStr = "http"
		} else if _, hasCommand := mcpConfig["command"]; hasCommand {
			typeStr = "stdio"
		} else if _, hasContainer := mcpConfig["container"]; hasContainer {
			typeStr = "stdio"
		} else {
			return fmt.Errorf("tool '%s' unable to determine MCP type: missing type, url, command, or container", toolName)
		}
	}

	// Normalize "local" to "stdio" for validation
	if typeStr == "local" {
		typeStr = "stdio"
	}

	// Validate type is one of the supported types
	if !isMCPType(typeStr) {
		return fmt.Errorf("tool '%s' mcp configuration 'type' value must be one of: stdio, http, local", toolName)
	}

	// Validate type-specific requirements
	switch typeStr {
	case "http":
		// HTTP type requires 'url' property
		url, hasURL := mcpConfig["url"]

		// HTTP type cannot use container field
		if _, hasContainer := mcpConfig["container"]; hasContainer {
			return fmt.Errorf("tool '%s' mcp configuration with type 'http' cannot use 'container' field", toolName)
		}

		return validateStringProperty(toolName, "url", url, hasURL)

	case "stdio":
		// stdio type requires either 'command' or 'container' property (but not both)
		command, hasCommand := mcpConfig["command"]
		container, hasContainer := mcpConfig["container"]

		if hasCommand && hasContainer {
			return fmt.Errorf("tool '%s' mcp configuration cannot specify both 'container' and 'command'", toolName)
		}

		if hasCommand {
			if err := validateStringProperty(toolName, "command", command, true); err != nil {
				return err
			}
		} else if hasContainer {
			if err := validateStringProperty(toolName, "container", container, true); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("tool '%s' mcp configuration must specify either 'command' or 'container'", toolName)
		}
	}

	return nil
}
