package workflow

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
// This is a shared function used by both Claude and Custom engines
func renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	yaml.WriteString("              \"playwright\": {\n")
	yaml.WriteString("                \"command\": \"npx\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"@playwright/mcp@latest\",\n")
	yaml.WriteString("                  \"--output-dir\",\n")
	yaml.WriteString("                  \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                  \"--allowed-origins\",\n")
		yaml.WriteString("                  \"" + strings.Join(args.AllowedDomains, ";") + "\"")
	}

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "                  ")

	yaml.WriteString("\n")
	yaml.WriteString("                ]\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
// This is a shared function used by both Claude and Custom engines
func renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	yaml.WriteString("              \"safe_outputs\": {\n")
	yaml.WriteString("                \"command\": \"node\",\n")
	yaml.WriteString("                \"args\": [\"/tmp/gh-aw/safe-outputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS\": \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS_CONFIG\": ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }},\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_BRANCH\": \"${{ env.GITHUB_AW_ASSETS_BRANCH }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_MAX_SIZE_KB\": \"${{ env.GITHUB_AW_ASSETS_MAX_SIZE_KB }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_ALLOWED_EXTS\": \"${{ env.GITHUB_AW_ASSETS_ALLOWED_EXTS }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
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
			propertyOrder = []string{"command", "args", "env", "proxy-args", "registry"}
		} else {
			// JSON format - include copilot fields if required
			propertyOrder = []string{"type", "command", "tools", "args", "env", "proxy-args", "registry"}
		}
	case "http":
		if renderer.Format == "toml" {
			// TOML format doesn't support HTTP type in some engines
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has type '%s', but %s only supports 'stdio'. Ignoring this server.", toolName, mcpType, renderer.Format)))
			return nil
		} else {
			// JSON format - include copilot fields if required
			if renderer.RequiresCopilotFields {
				propertyOrder = []string{"type", "url", "headers", "tools"}
			} else {
				propertyOrder = []string{"type", "url", "headers"}
			}
		}
	default:
		if renderer.Format == "toml" {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio", toolName, mcpType)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio, http", toolName, mcpType)))
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
				envKeys := make([]string, 0, len(mcpConfig.Env))
				for key := range mcpConfig.Env {
					envKeys = append(envKeys, key)
				}
				sort.Strings(envKeys)
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
			sort.Strings(headerKeys)
			for headerIndex, headerKey := range headerKeys {
				headerComma := ","
				if headerIndex == len(headerKeys)-1 {
					headerComma = ""
				}
				fmt.Fprintf(yaml, "%s  \"%s\": \"%s\"%s\n", renderer.IndentLevel, headerKey, mcpConfig.Headers[headerKey], headerComma)
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

// getMCPConfig extracts MCP configuration from a tool config and returns a structured MCPServerConfig
func getMCPConfig(toolConfig map[string]any, toolName string) (*parser.MCPServerConfig, error) {
	config := MapToolConfig(toolConfig)
	result := &parser.MCPServerConfig{
		Name:    toolName,
		Env:     make(map[string]string),
		Headers: make(map[string]string),
	}

	// Validate known properties - fail if unknown properties are found
	knownProperties := map[string]bool{
		"type":       true,
		"command":    true,
		"container":  true,
		"version":    true,
		"args":       true,
		"env":        true,
		"proxy-args": true,
		"url":        true,
		"headers":    true,
		"registry":   true,
		"allowed":    true,
		"network":    true,
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

		// Add the container image as the last argument
		result.Args = append(result.Args, containerImage)

		// Clear the container and version fields since they're now part of the command
		result.Container = ""
		result.Version = ""
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
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			// Extract raw MCP configuration (without transformation)
			mcpConfig, err := getRawMCPConfig(config)
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
func getRawMCPConfig(toolConfig map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// List of MCP fields that can be direct children of the tool config
	// Note: "args" is NOT included here because it's used for built-in tools (github, playwright)
	// to add custom arguments without triggering custom MCP tool processing logic. Including "args"
	// would incorrectly classify built-in tools as custom MCP tools, changing their processing behavior
	// and causing validation errors.
	mcpFields := []string{"type", "url", "command", "container", "env", "headers", "network"}

	// Check new format: direct fields in tool config
	for _, field := range mcpFields {
		if value, exists := toolConfig[field]; exists {
			result[field] = value
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
	extract := func(network any) (bool, []string) {
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

	// Check direct network field at tool level
	if network, hasNetwork := toolConfig["network"]; hasNetwork {
		if ok, domains := extract(network); ok {
			return true, domains
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
