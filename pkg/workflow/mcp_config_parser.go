package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/types"
)

var mcpParserLog = logger.New("workflow:mcp-parser")

// getMCPConfig extracts MCP configuration from a tool config and returns a structured MCPServerConfig
func getMCPConfig(toolConfig map[string]any, toolName string) (*parser.MCPServerConfig, error) {
	mcpParserLog.Printf("Extracting MCP config for tool: %s", toolName)

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
			mcpParserLog.Printf("Unknown property '%s' in MCP config for tool '%s'", key, toolName)
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
		mcpParserLog.Printf("MCP type explicitly set to: %s", typeStr)
		// Normalize "local" to "stdio"
		if typeStr == "local" {
			result.Type = "stdio"
		} else {
			result.Type = typeStr
		}
	} else {
		mcpParserLog.Print("No explicit MCP type, inferring from fields")
		// Infer type from presence of fields
		if _, hasURL := config.GetString("url"); hasURL {
			result.Type = "http"
			mcpParserLog.Printf("Inferred MCP type as http (has url field)")
		} else if _, hasCommand := config.GetString("command"); hasCommand {
			result.Type = "stdio"
			mcpParserLog.Printf("Inferred MCP type as stdio (has command field)")
		} else if _, hasContainer := config.GetString("container"); hasContainer {
			result.Type = "stdio"
			mcpParserLog.Printf("Inferred MCP type as stdio (has container field)")
		} else {
			mcpParserLog.Printf("Unable to determine MCP type for tool '%s': missing type, url, command, or container", toolName)
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
	mcpParserLog.Printf("Extracting fields for MCP type: %s", result.Type)
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
			mcpParserLog.Printf("HTTP MCP tool '%s' missing required 'url' field", toolName)
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
		mcpParserLog.Printf("Unsupported MCP type '%s' for tool '%s'", result.Type, toolName)
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
			mcpParserLog.Printf("Auto-assigning container for command '%s': %s", result.Command, containerConfig.Image)
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
