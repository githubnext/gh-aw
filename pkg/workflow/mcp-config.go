package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
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
func renderSharedMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, renderer MCPConfigRenderer) error {
	// Get MCP configuration in the new format
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}

	// Determine properties based on type
	var propertyOrder []string
	mcpType := mcpConfig.Type

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
		switch prop {
		case "command":
			if mcpConfig.Command != "" {
				existingProperties = append(existingProperties, prop)
			}
		case "args":
			if len(mcpConfig.Args) > 0 {
				existingProperties = append(existingProperties, prop)
			}
		case "env":
			if len(mcpConfig.Env) > 0 {
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
				first := true
				for envKey, envValue := range mcpConfig.Env {
					if !first {
						yaml.WriteString(", ")
					}
					fmt.Fprintf(yaml, "\"%s\" = \"%s\"", envKey, envValue)
					first = false
				}
				yaml.WriteString(" }\n")
			} else {
				comma := ","
				if isLast {
					comma = ""
				}
				fmt.Fprintf(yaml, "%s\"env\": {\n", renderer.IndentLevel)
				envKeys := make([]string, 0, len(mcpConfig.Env))
				for key := range mcpConfig.Env {
					envKeys = append(envKeys, key)
				}
				for envIndex, envKey := range envKeys {
					envComma := ","
					if envIndex == len(envKeys)-1 {
						envComma = ""
					}
					fmt.Fprintf(yaml, "%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, envKey, mcpConfig.Env[envKey], envComma)
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
			for headerIndex, headerKey := range headerKeys {
				headerComma := ","
				if headerIndex == len(headerKeys)-1 {
					headerComma = ""
				}
				fmt.Fprintf(yaml, "%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, headerKey, mcpConfig.Headers[headerKey], headerComma)
			}
			fmt.Fprintf(yaml, "%s}%s\n", renderer.IndentLevel, comma)
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

// getMCPConfig extracts MCP configuration from a tool config and returns a structured MCPServerConfig
func getMCPConfig(toolConfig map[string]any, toolName string) (*parser.MCPServerConfig, error) {
	config := MapToolConfig(toolConfig)
	result := &parser.MCPServerConfig{
		Name:    toolName,
		Env:     make(map[string]string),
		Headers: make(map[string]string),
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
		if args, hasArgs := config.GetStringArray("args"); hasArgs {
			result.Args = args
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
		// Transform container field to docker command and args
		result.Command = "docker"
		result.Args = []string{"run", "--rm", "-i"}

		// Add environment variables as -e flags
		for envKey := range result.Env {
			result.Args = append(result.Args, "-e", envKey)
		}

		// Add the container image as the last argument
		result.Args = append(result.Args, result.Container)

		// Clear the container field since it's now part of the command
		result.Container = ""
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

	return false, ""
}

// validateMCPConfigs validates all MCP configurations in the tools section using JSON schema
func ValidateMCPConfigs(tools map[string]any) error {
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			// Extract raw MCP configuration (without transformation)
			mcpConfig, err := getRawMCPConfig(config, toolName)
			if err != nil {
				return fmt.Errorf("tool '%s' has invalid MCP configuration: %w", toolName, err)
			}

			// Skip validation if no MCP configuration found
			if len(mcpConfig) == 0 {
				continue
			}

			// Validate MCP configuration requirements (before transformation)
			if err := validateMCPRequirements(toolName, mcpConfig, config); err != nil {
				return err
			}
		}
	}
	return nil
}

// getRawMCPConfig extracts MCP configuration without any transformations for validation
func getRawMCPConfig(toolConfig map[string]any, toolName string) (map[string]any, error) {
	result := make(map[string]any)

	// List of MCP fields that can be direct children of the tool config
	// Note: "permissions" stays at the tool level, not an MCP field
	mcpFields := []string{"type", "url", "command", "container", "args", "env", "headers"}

	// Check new format: direct fields in tool config
	hasDirectFields := false
	for _, field := range mcpFields {
		if value, exists := toolConfig[field]; exists {
			result[field] = value
			hasDirectFields = true
		}
	}

	// If we have direct fields, use them and skip legacy format
	if hasDirectFields {
		return result, nil
	}

	// Fall back to legacy format: mcp.type, mcp.url, mcp.command, etc.
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
