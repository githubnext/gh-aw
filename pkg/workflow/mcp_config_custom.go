package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/types"
)

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
		"entrypointArgs": true,
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
