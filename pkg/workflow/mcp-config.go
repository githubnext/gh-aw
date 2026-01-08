package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpLog = logger.New("workflow:mcp-config")

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses Docker container to launch Playwright MCP for consistent browser environment
// This is a shared function used by both Claude and Custom engines




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



