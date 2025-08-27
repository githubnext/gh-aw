package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// MCPConfigRenderer contains configuration options for rendering MCP config
type MCPConfigRenderer struct {
	// IndentLevel controls the indentation level for properties (e.g., "                " for JSON, "          " for TOML)
	IndentLevel string
	// Format specifies the output format ("json" for JSON-like, "toml" for TOML-like)
	Format string
}

// renderSharedMCPConfig generates MCP server configuration for a single tool using shared logic
// This function handles the common logic for rendering MCP configurations across different engines
func renderSharedMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool, renderer MCPConfigRenderer) error {
	// Get MCP configuration in the new format
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}

	// Determine properties based on type
	var propertyOrder []string
	mcpType := "stdio" // default
	if serverType, exists := mcpConfig["type"]; exists {
		if typeStr, ok := serverType.(string); ok {
			mcpType = typeStr
		}
	}

	switch mcpType {
	case "stdio":
		if renderer.Format == "toml" {
			propertyOrder = []string{"command", "args", "env"}
		} else {
			propertyOrder = []string{"command", "args", "env"}
		}
	case "http":
		if renderer.Format == "toml" {
			// TOML format doesn't support HTTP type in some engines
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has type '%s', but %s only supports 'stdio'. Ignoring this server.", toolName, mcpType, renderer.Format)))
			return nil
		} else {
			propertyOrder = []string{"url", "headers"}
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
		if _, exists := mcpConfig[prop]; exists {
			existingProperties = append(existingProperties, prop)
		}
	}

	// If no valid properties exist, skip rendering
	if len(existingProperties) == 0 {
		return nil
	}

	// Render properties based on format
	for propIndex, property := range existingProperties {
		isLastProperty := propIndex == len(existingProperties)-1

		switch property {
		case "command":
			if command, exists := mcpConfig["command"]; exists {
				if cmdStr, ok := command.(string); ok {
					if renderer.Format == "toml" {
						yaml.WriteString(fmt.Sprintf("%scommand = \"%s\"\n", renderer.IndentLevel, cmdStr))
					} else {
						comma := ","
						if isLastProperty {
							comma = ""
						}
						yaml.WriteString(fmt.Sprintf("%s\"command\": \"%s\"%s\n", renderer.IndentLevel, cmdStr, comma))
					}
				}
			}
		case "args":
			if args, exists := mcpConfig["args"]; exists {
				if argsSlice, ok := args.([]any); ok {
					if renderer.Format == "toml" {
						yaml.WriteString(fmt.Sprintf("%sargs = [\n", renderer.IndentLevel))
						for _, arg := range argsSlice {
							if argStr, ok := arg.(string); ok {
								yaml.WriteString(fmt.Sprintf("%s  \"%s\",\n", renderer.IndentLevel, argStr))
							}
						}
						yaml.WriteString(fmt.Sprintf("%s]\n", renderer.IndentLevel))
					} else {
						comma := ","
						if isLastProperty {
							comma = ""
						}
						yaml.WriteString(fmt.Sprintf("%s\"args\": [\n", renderer.IndentLevel))
						for argIndex, arg := range argsSlice {
							if argStr, ok := arg.(string); ok {
								argComma := ","
								if argIndex == len(argsSlice)-1 {
									argComma = ""
								}
								yaml.WriteString(fmt.Sprintf("%s  \"%s\"%s\n", renderer.IndentLevel, argStr, argComma))
							}
						}
						yaml.WriteString(fmt.Sprintf("%s]%s\n", renderer.IndentLevel, comma))
					}
				}
			}
		case "env":
			if env, exists := mcpConfig["env"]; exists {
				if envMap, ok := env.(map[string]any); ok {
					if renderer.Format == "toml" {
						yaml.WriteString(fmt.Sprintf("%senv = { ", renderer.IndentLevel))
						first := true
						for envKey, envValue := range envMap {
							if !first {
								yaml.WriteString(", ")
							}
							if envStr, ok := envValue.(string); ok {
								yaml.WriteString(fmt.Sprintf("\"%s\" = \"%s\"", envKey, envStr))
							}
							first = false
						}
						yaml.WriteString(" }\n")
					} else {
						comma := ","
						if isLastProperty {
							comma = ""
						}
						yaml.WriteString(fmt.Sprintf("%s\"env\": {\n", renderer.IndentLevel))
						envKeys := make([]string, 0, len(envMap))
						for key := range envMap {
							envKeys = append(envKeys, key)
						}
						for envIndex, envKey := range envKeys {
							if envValue, ok := envMap[envKey].(string); ok {
								envComma := ","
								if envIndex == len(envKeys)-1 {
									envComma = ""
								}
								yaml.WriteString(fmt.Sprintf("%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, envKey, envValue, envComma))
							}
						}
						yaml.WriteString(fmt.Sprintf("%s}%s\n", renderer.IndentLevel, comma))
					}
				}
			}
		case "url":
			if url, exists := mcpConfig["url"]; exists {
				if urlStr, ok := url.(string); ok {
					comma := ","
					if isLastProperty {
						comma = ""
					}
					yaml.WriteString(fmt.Sprintf("%s\"url\": \"%s\"%s\n", renderer.IndentLevel, urlStr, comma))
				}
			}
		case "headers":
			if headers, exists := mcpConfig["headers"]; exists {
				if headersMap, ok := headers.(map[string]any); ok {
					comma := ","
					if isLastProperty {
						comma = ""
					}
					yaml.WriteString(fmt.Sprintf("%s\"headers\": {\n", renderer.IndentLevel))
					headerKeys := make([]string, 0, len(headersMap))
					for key := range headersMap {
						headerKeys = append(headerKeys, key)
					}
					for headerIndex, headerKey := range headerKeys {
						if headerValue, ok := headersMap[headerKey].(string); ok {
							headerComma := ","
							if headerIndex == len(headerKeys)-1 {
								headerComma = ""
							}
							yaml.WriteString(fmt.Sprintf("%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, headerKey, headerValue, headerComma))
						}
					}
					yaml.WriteString(fmt.Sprintf("%s}%s\n", renderer.IndentLevel, comma))
				}
			}
		}
	}

	return nil
}

// getMCPConfig extracts MCP configuration from a tool config in the new format
func getMCPConfig(toolConfig map[string]any, toolName string) (map[string]any, error) {
	result := make(map[string]any)

	// Check new format: mcp.type, mcp.url, mcp.command, etc.
	if mcpSection, hasMcp := toolConfig["mcp"]; hasMcp {
		if mcpMap, ok := mcpSection.(map[string]any); ok {
			// Copy all MCP properties
			for key, value := range mcpMap {
				result[key] = value
			}
		} else if mcpString, ok := mcpSection.(string); ok {
			// Handle JSON string format
			var parsedMcp map[string]any
			if err := json.Unmarshal([]byte(mcpString), &parsedMcp); err != nil {
				return nil, fmt.Errorf("invalid JSON in mcp configuration: %w", err)
			}
			// Copy all MCP properties from parsed JSON
			for key, value := range parsedMcp {
				result[key] = value
			}
		}
	}

	// Check if this container needs proxy support
	if _, hasContainer := result["container"]; hasContainer {
		if hasNetPerms, _ := hasNetworkPermissions(toolConfig); hasNetPerms {
			// Mark this configuration as proxy-enabled
			result["__uses_proxy"] = true
		}
	}

	// Transform container field to docker command if present
	if err := transformContainerToDockerCommand(result, toolName); err != nil {
		return nil, err
	}

	return result, nil
}

// transformContainerToDockerCommand converts a container field to docker command and args
// For proxy-enabled containers, it sets special markers instead of docker commands
func transformContainerToDockerCommand(mcpConfig map[string]any, toolName string) error {
	container, hasContainer := mcpConfig["container"]
	if !hasContainer {
		return nil // No container field, nothing to transform
	}

	// Ensure container is a string
	containerStr, ok := container.(string)
	if !ok {
		return fmt.Errorf("'container' must be a string")
	}

	// Check for conflicting command field
	if _, hasCommand := mcpConfig["command"]; hasCommand {
		return fmt.Errorf("cannot specify both 'container' and 'command' fields")
	}

	// Check if this is a proxy-enabled container (has special marker)
	if _, hasProxyFlag := mcpConfig["__uses_proxy"]; hasProxyFlag {
		// For proxy-enabled containers, use docker compose run to connect to the MCP server
		mcpConfig["command"] = "docker"
		if toolName != "" {
			mcpConfig["args"] = []any{"compose", "-f", fmt.Sprintf("docker-compose-%s.yml", toolName), "run", "--rm", toolName}
		}
		// Keep the container field for compose file generation
		return nil
	}

	// Set docker command
	mcpConfig["command"] = "docker"

	// Build args
	args := []any{"run", "--rm", "-i"}

	// Add environment variable flags
	if env, hasEnv := mcpConfig["env"]; hasEnv {
		if envMap, ok := env.(map[string]any); ok {
			// Sort env keys for consistent output
			var envKeys []string
			for envKey := range envMap {
				envKeys = append(envKeys, envKey)
			}
			// Sort for consistent output
			for i := 0; i < len(envKeys)-1; i++ {
				for j := i + 1; j < len(envKeys); j++ {
					if envKeys[i] > envKeys[j] {
						envKeys[i], envKeys[j] = envKeys[j], envKeys[i]
					}
				}
			}

			for _, envKey := range envKeys {
				args = append(args, "-e", envKey)
			}
		}
	}

	// Add container name as the last argument
	args = append(args, containerStr)

	// Set the args
	mcpConfig["args"] = args

	// Remove the container field as it's been transformed
	delete(mcpConfig, "container")

	return nil
}

// isMCPType checks if a type string represents an MCP-compatible type
func isMCPType(typeStr string) bool {
	switch typeStr {
	case "stdio", "http":
		return true
	default:
		return false
	}
}

// hasMCPConfig checks if a tool configuration has MCP configuration
func hasMCPConfig(toolConfig map[string]any) (bool, string) {
	// Check new format: mcp.type
	if mcpSection, hasMcp := toolConfig["mcp"]; hasMcp {
		if mcpMap, ok := mcpSection.(map[string]any); ok {
			if mcpType, hasType := mcpMap["type"]; hasType {
				if typeStr, ok := mcpType.(string); ok && isMCPType(typeStr) {
					return true, typeStr
				}
			}
		} else if mcpString, ok := mcpSection.(string); ok {
			// Handle JSON string format
			var parsedMcp map[string]any
			if err := json.Unmarshal([]byte(mcpString), &parsedMcp); err == nil {
				if mcpType, hasType := parsedMcp["type"]; hasType {
					if typeStr, ok := mcpType.(string); ok && isMCPType(typeStr) {
						return true, typeStr
					}
				}
			}
		}
	}

	return false, ""
}

// validateMCPConfigs validates all MCP configurations in the tools section using JSON schema
func ValidateMCPConfigs(tools map[string]any) error {
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			if mcpSection, hasMcp := config["mcp"]; hasMcp {
				var mcpConfig map[string]any

				if mcpString, ok := mcpSection.(string); ok {
					// Validate JSON string format
					var parsedMcp map[string]any
					if err := json.Unmarshal([]byte(mcpString), &parsedMcp); err != nil {
						return fmt.Errorf("tool '%s' has invalid JSON in mcp configuration: %w", toolName, err)
					}
					mcpConfig = parsedMcp
				} else if mcpMap, ok := mcpSection.(map[string]any); ok {
					// Create a copy to avoid modifying the original during validation
					mcpConfig = make(map[string]any)
					for k, v := range mcpMap {
						mcpConfig[k] = v
					}
				} else {
					return fmt.Errorf("tool '%s' has invalid mcp configuration format", toolName)
				}

				// Validate MCP configuration requirements (before transformation)
				if err := validateMCPRequirements(toolName, mcpConfig, config); err != nil {
					return err
				}
			}
		}
	}
	return nil
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

// hasNetworkPermissions checks if a tool configuration has network permissions
func hasNetworkPermissions(toolConfig map[string]any) (bool, []string) {
	extract := func(perms any) (bool, []string) {
		permsMap, ok := perms.(map[string]any)
		if !ok {
			return false, nil
		}
		network, hasNetwork := permsMap["network"]
		if !hasNetwork {
			return false, nil
		}
		networkMap, ok := network.(map[string]any)
		if !ok {
			return false, nil
		}
		allowed, hasAllowed := networkMap["allowed"]
		if !hasAllowed {
			return false, nil
		}
		allowedSlice, ok := allowed.([]any)
		if !ok {
			return false, nil
		}
		var domains []string
		for _, item := range allowedSlice {
			if str, ok := item.(string); ok {
				domains = append(domains, str)
			}
		}
		return len(domains) > 0, domains
	}

	// First, check top-level permissions
	if permissions, hasPerms := toolConfig["permissions"]; hasPerms {
		if ok, domains := extract(permissions); ok {
			return true, domains
		}
	}

	// Then, check permissions nested under mcp (alternate schema used in some configs)
	if mcpSection, hasMcp := toolConfig["mcp"]; hasMcp {
		if m, ok := mcpSection.(map[string]any); ok {
			if permissions, hasPerms := m["permissions"]; hasPerms {
				if ok, domains := extract(permissions); ok {
					return true, domains
				}
			}
		}
	}

	return false, nil
}

// validateMCPRequirements validates the specific requirements for MCP configuration
func validateMCPRequirements(toolName string, mcpConfig map[string]any, toolConfig map[string]any) error {
	// Validate 'type' property
	mcpType, hasType := mcpConfig["type"]
	if err := validateStringProperty(toolName, "type", mcpType, hasType); err != nil {
		return err
	}

	typeStr, ok := mcpType.(string)
	if !ok {
		// This should never happen since validateStringProperty passed, but be defensive
		return fmt.Errorf("tool '%s' mcp configuration 'type' validation error", toolName)
	}

	// Validate type is one of the supported types
	if !isMCPType(typeStr) {
		return fmt.Errorf("tool '%s' mcp configuration 'type' value must be one of: stdio, http", toolName)
	}

	// Validate network permissions usage first
	hasNetPerms, _ := hasNetworkPermissions(toolConfig)
	if !hasNetPerms {
		// Also check if permissions are nested in the mcp config itself
		hasNetPerms, _ = hasNetworkPermissions(map[string]any{"mcp": mcpConfig})
	}
	if hasNetPerms {
		switch typeStr {
		case "http":
			return fmt.Errorf("tool '%s' has network permissions configured, but network egress permissions do not apply to remote 'type: http' servers", toolName)
		case "stdio":
			// Network permissions only apply to stdio servers with container
			_, hasContainer := mcpConfig["container"]
			if !hasContainer {
				return fmt.Errorf("tool '%s' has network permissions configured, but network egress permissions only apply to stdio MCP servers that specify a 'container'", toolName)
			}
		}
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
