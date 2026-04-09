// This file provides MCP (Model Context Protocol) configuration validation.
//
// # MCP Configuration Validation
//
// This file validates MCP server configurations in agentic workflows.
// It ensures that MCP configurations have required fields, correct types,
// and satisfy type-specific requirements (stdio vs http).
//
// # Validation Functions
//
//   - ValidateMCPConfigs() - Validates MCP server configurations (from mcp-servers) in the merged tools map
//   - ValidateToolsSection() - Validates the tools: frontmatter section only allows built-in tool names
//   - validateStringProperty() - Validates that a property is a string type
//   - validateMCPRequirements() - Validates type-specific MCP requirements
//
// # Validation Pattern: Schema and Requirements Validation
//
// MCP validation uses multiple patterns:
//   - Type inference: Determines MCP type from fields if not explicit
//   - Required field validation: Ensures necessary fields exist
//   - Type-specific validation: Different requirements for stdio vs http
//   - Property type checking: Validates field types match expectations
//
// # MCP Types and Requirements
//
// ## stdio type
//   - Requires either 'command' or 'container' (but not both)
//   - Optional: version, args, entrypointArgs, env, proxy-args, registry
//
// ## http type
//   - Requires 'url' field
//   - Cannot use 'container' field
//   - Optional: headers, registry
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates MCP server configuration
//   - It checks MCP-specific field requirements
//   - It validates MCP type compatibility
//   - It ensures MCP configuration correctness
//
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

var mcpValidationLog = newValidationLogger("mcp_config")

// builtInToolNames is the canonical set of recognized built-in tool names for the tools: section.
// Any key in tools: that is not in this set is a compile error.
// Custom MCP servers must be placed under mcp-servers: instead.
var builtInToolNames = map[string]bool{
	"github":            true,
	"playwright":        true,
	"agentic-workflows": true,
	"cache-memory":      true,
	"repo-memory":       true,
	"bash":              true,
	"edit":              true,
	"web-fetch":         true,
	"web-search":        true,
	"safety-prompt":     true,
	"timeout":           true,
	"startup-timeout":   true,
}

// builtInToolNamesForError is the sorted, comma-separated list of built-in tool names
// used in error messages, derived once from builtInToolNames.
var builtInToolNamesForError = func() string {
	names := make([]string, 0, len(builtInToolNames))
	for name := range builtInToolNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}()

// ValidateMCPConfigs validates all MCP configurations in the tools section using JSON schema.
// It validates MCP server entries (from mcp-servers, merged into tools) but does not check
// for unknown tool names — that is done earlier by ValidateToolsSection.
func ValidateMCPConfigs(tools map[string]any) error {
	mcpValidationLog.Printf("Validating MCP configurations for %d tools", len(tools))

	// Collect and sort tool names for deterministic error messages
	toolNames := make([]string, 0, len(tools))
	for name := range tools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		toolConfig := tools[toolName]

		// Skip built-in tools - they have their own schema validation
		if builtInToolNames[toolName] {
			mcpValidationLog.Printf("Skipping MCP validation for built-in tool: %s", toolName)
			continue
		}

		config, ok := toolConfig.(map[string]any)
		if !ok {
			// Non-map configs for custom MCP servers (from mcp-servers section) are skipped here
			continue
		}

		// Extract raw MCP configuration (without transformation)
		mcpConfig, err := getRawMCPConfig(config)
		if err != nil {
			mcpValidationLog.Printf("Invalid MCP configuration for tool %s: %v", toolName, err)
			return fmt.Errorf("tool '%s' has invalid MCP configuration: %w", toolName, err)
		}

		// Skip validation if no MCP configuration found
		if len(mcpConfig) == 0 {
			continue
		}

		mcpValidationLog.Printf("Validating MCP requirements for tool: %s", toolName)

		// Validate MCP configuration requirements first (before transformation).
		// Custom validation runs before schema validation to provide better error messages
		// for the most common mistakes (matching the pattern in ValidateMainWorkflowFrontmatterWithSchemaAndLocation).
		if err := validateMCPRequirements(toolName, mcpConfig, config); err != nil {
			return err
		}

		// Run JSON schema validation as a catch-all after custom validation. Build a
		// schema-compatible view of the config by extracting only the properties defined
		// in mcp_config_schema.json. Tool-specific fields (e.g. auth, proxy-args) are
		// excluded because the schema uses additionalProperties: false.
		if err := parser.ValidateMCPConfigWithSchema(buildSchemaMCPConfig(config)); err != nil {
			mcpValidationLog.Printf("JSON schema validation failed for tool %s: %v", toolName, err)
			return fmt.Errorf("tool '%s' has invalid MCP configuration: %w", toolName, err)
		}
	}

	mcpValidationLog.Print("MCP configuration validation completed successfully")
	return nil
}

// ValidateToolsSection validates that all entries in the user-facing tools: frontmatter section
// are recognized built-in tool names. Custom MCP servers must be placed under mcp-servers: instead.
// This is called on topTools (before merging with mcp-servers) to give accurate user-facing errors.
func ValidateToolsSection(tools map[string]any) error {
	if len(tools) == 0 {
		return nil
	}

	// Collect and sort names for deterministic error messages
	toolNames := make([]string, 0, len(tools))
	for name := range tools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		if !builtInToolNames[toolName] {
			mcpValidationLog.Printf("Unknown tool in tools section: %s", toolName)
			return fmt.Errorf("tools.%s: unknown tool name. The 'tools' section only accepts built-in tool names.\n\nValid built-in tools: %s.\n\nIf '%s' is a custom MCP server, define it under 'mcp-servers' instead:\nmcp-servers:\n  %s:\n    command: \"node server.js\"\n    args: [\"--port\", \"3000\"]\n\nSee: %s", toolName, builtInToolNamesForError, toolName, toolName, constants.DocsToolsURL)
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
	mcpFields := []string{"type", "url", "command", "container", "env", "headers"}

	// List of all known tool config fields (not just MCP)
	knownToolFields := map[string]bool{
		"type":           true,
		"url":            true,
		"command":        true,
		"container":      true,
		"env":            true,
		"headers":        true,
		"auth":           true, // upstream OIDC authentication (HTTP servers only)
		"version":        true,
		"args":           true,
		"entrypoint":     true,
		"entrypointArgs": true,
		"mounts":         true,
		"proxy-args":     true,
		"registry":       true,
		"allowed":        true,
		"mode":           true, // for github tool
		"github-token":   true, // for github tool
		"read-only":      true, // for github tool
		"toolsets":       true, // for github tool
		"id":             true, // for cache-memory (array notation)
		"key":            true, // for cache-memory
		"description":    true, // for cache-memory
		"retention-days": true, // for cache-memory
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
			// Build list of valid fields for the error message
			validFields := []string{}
			for k := range knownToolFields {
				validFields = append(validFields, k)
			}
			sort.Strings(validFields)
			maxFields := min(10, len(validFields))
			return nil, fmt.Errorf("unknown property '%s' in tool configuration. Valid properties include: %s.\n\nExample:\ntools:\n  my-tool:\n    command: \"node server.js\"\n    args: [\"--verbose\"]\n\nSee: %s", field, strings.Join(validFields[:maxFields], ", "), constants.DocsToolsURL)
		}
	}

	return result, nil
}

// inferMCPType infers the MCP connection type from the fields present in a config map.
// Returns "http" when a url field is present, "stdio" when command or container is present,
// and an empty string when the type cannot be determined. It does not validate the explicit
// 'type' field — that is done by the caller.
func inferMCPType(config map[string]any) string {
	if _, hasURL := config["url"]; hasURL {
		return "http"
	}
	if _, hasCommand := config["command"]; hasCommand {
		return "stdio"
	}
	if _, hasContainer := config["container"]; hasContainer {
		return "stdio"
	}
	return ""
}

// validateStringProperty validates that a property is a string and returns appropriate error message
func validateStringProperty(toolName, propertyName string, value any, exists bool) error {
	if !exists {
		return fmt.Errorf("tool '%s' mcp configuration missing required property '%s'.\n\nExample:\ntools:\n  %s:\n    %s: \"value\"\n\nSee: %s", toolName, propertyName, toolName, propertyName, constants.DocsToolsURL)
	}
	if _, ok := value.(string); !ok {
		return fmt.Errorf("tool '%s' mcp configuration property '%s' must be a string, got %T.\n\nExample:\ntools:\n  %s:\n    %s: \"my-value\"\n\nSee: %s", toolName, propertyName, value, toolName, propertyName, constants.DocsToolsURL)
	}
	return nil
}

// validateMCPRequirements validates the specific requirements for MCP configuration
func validateMCPRequirements(toolName string, mcpConfig map[string]any, toolConfig map[string]any) error {
	// Validate 'type' property - allow inference from other fields
	mcpType, hasType := mcpConfig["type"]
	var typeStr string

	if hasType {
		// Explicit type provided - validate it's a string
		if _, ok := mcpType.(string); !ok {
			return fmt.Errorf("tool '%s' mcp configuration 'type' must be a string, got %T. Valid types per MCP Gateway Specification: stdio, http. Note: 'local' is accepted for backward compatibility and treated as 'stdio'.\n\nExample:\ntools:\n  %s:\n    type: \"stdio\"\n    command: \"node server.js\"\n\nSee: %s", toolName, mcpType, toolName, constants.DocsToolsURL)
		}
		typeStr = mcpType.(string)
	} else {
		// Infer type from presence of fields
		typeStr = inferMCPType(mcpConfig)
		if typeStr == "" {
			return fmt.Errorf("tool '%s' unable to determine MCP type: missing type, url, command, or container.\n\nExample:\ntools:\n  %s:\n    command: \"node server.js\"\n    args: [\"--port\", \"3000\"]\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
		}
	}

	// Normalize "local" to "stdio" for validation
	if typeStr == "local" {
		typeStr = "stdio"
	}

	// Validate type is one of the supported types
	if !parser.IsMCPType(typeStr) {
		return fmt.Errorf("tool '%s' mcp configuration 'type' must be one of: stdio, http (per MCP Gateway Specification). Note: 'local' is accepted for backward compatibility and treated as 'stdio'. Got: %s.\n\nExample:\ntools:\n  %s:\n    type: \"stdio\"\n    command: \"node server.js\"\n\nSee: %s", toolName, typeStr, toolName, constants.DocsToolsURL)
	}

	// Validate type-specific requirements
	switch typeStr {
	case "http":
		// HTTP type requires 'url' property
		url, hasURL := mcpConfig["url"]

		// HTTP type cannot use container field
		if _, hasContainer := mcpConfig["container"]; hasContainer {
			return fmt.Errorf("tool '%s' mcp configuration with type 'http' cannot use 'container' field. HTTP MCP uses URL endpoints, not containers.\n\nExample:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n    headers:\n      Authorization: \"Bearer ${{ secrets.API_KEY }}\"\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
		}

		// HTTP type cannot use mounts field (MCP Gateway v0.1.5+)
		if _, hasMounts := toolConfig["mounts"]; hasMounts {
			return fmt.Errorf("tool '%s' mcp configuration with type 'http' cannot use 'mounts' field. Volume mounts are only supported for stdio (containerized) MCP servers.\n\nExample:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
		}

		// Validate auth if present: must have a valid type field
		if authRaw, hasAuth := toolConfig["auth"]; hasAuth {
			authMap, ok := authRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("tool '%s' mcp configuration 'auth' must be an object.\n\nExample:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n    auth:\n      type: github-oidc\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
			}
			authType, hasAuthType := authMap["type"]
			if !hasAuthType {
				return fmt.Errorf("tool '%s' mcp configuration 'auth.type' is required.\n\nExample:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n    auth:\n      type: github-oidc\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
			}
			authTypeStr, ok := authType.(string)
			if !ok || authTypeStr == "" {
				return fmt.Errorf("tool '%s' mcp configuration 'auth.type' must be a non-empty string. Currently only 'github-oidc' is supported.\n\nExample:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n    auth:\n      type: github-oidc\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
			}
			if authTypeStr != "github-oidc" {
				return fmt.Errorf("tool '%s' mcp configuration 'auth.type' value %q is not supported. Currently only 'github-oidc' is supported.\n\nExample:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n    auth:\n      type: github-oidc\n\nSee: %s", toolName, authTypeStr, toolName, constants.DocsToolsURL)
			}
		}

		return validateStringProperty(toolName, "url", url, hasURL)

	case "stdio":
		// stdio type does not support auth (auth is only valid for HTTP servers)
		if _, hasAuth := toolConfig["auth"]; hasAuth {
			return fmt.Errorf("tool '%s' mcp configuration 'auth' is only supported for HTTP servers (type: 'http'). Stdio servers do not support upstream authentication.\n\nIf you need upstream auth, use an HTTP MCP server:\ntools:\n  %s:\n    type: http\n    url: \"https://api.example.com/mcp\"\n    auth:\n      type: github-oidc\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
		}

		// stdio type requires either 'command' or 'container' property (but not both)
		command, hasCommand := mcpConfig["command"]
		container, hasContainer := mcpConfig["container"]

		if hasCommand && hasContainer {
			return fmt.Errorf("tool '%s' mcp configuration cannot specify both 'container' and 'command'. Choose one.\n\nExample (command):\ntools:\n  %s:\n    command: \"node server.js\"\n\nExample (container):\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    version: \"latest\"\n\nSee: %s", toolName, toolName, toolName, constants.DocsToolsURL)
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
			return fmt.Errorf("tool '%s' mcp configuration must specify either 'command' or 'container'.\n\nExample (command):\ntools:\n  %s:\n    command: \"node server.js\"\n    args: [\"--port\", \"3000\"]\n\nExample (container):\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    version: \"latest\"\n\nSee: %s", toolName, toolName, toolName, constants.DocsToolsURL)
		}

		// Validate mount syntax if mounts are specified (MCP Gateway v0.1.5+ requires explicit mode)
		if mountsRaw, hasMounts := toolConfig["mounts"]; hasMounts {
			if err := validateMCPMountsSyntax(toolName, mountsRaw); err != nil {
				return err
			}
		}
	}

	return nil
}

// mcpSchemaTopLevelFields is the set of properties defined at the top level of
// mcp_config_schema.json. Only these fields should be passed to
// parser.ValidateMCPConfigWithSchema; the schema uses additionalProperties: false
// so any extra field would cause a spurious validation failure.
//
// WARNING: This map must be kept in sync with the properties defined in
// pkg/parser/schemas/mcp_config_schema.json. If you add or remove a property
// from that schema, update this map accordingly.
var mcpSchemaTopLevelFields = map[string]bool{
	"type":           true,
	"registry":       true,
	"url":            true,
	"command":        true,
	"container":      true,
	"args":           true,
	"entrypoint":     true,
	"entrypointArgs": true,
	"mounts":         true,
	"env":            true,
	"headers":        true,
	"network":        true,
	"allowed":        true,
	"version":        true,
}

// buildSchemaMCPConfig extracts only the fields defined in mcp_config_schema.json
// from a full tool config map. Tool-specific fields that are not part of the MCP
// schema (e.g. auth, proxy-args, mode, github-token) are excluded so that schema
// validation does not fail on fields unknown to the schema.
//
// If the 'type' field is absent but can be inferred from other fields (url → http,
// command/container → stdio), the inferred type is injected. This is necessary because
// the schema's if/then conditions use properties-based matching which is vacuously true
// when 'type' is absent, causing contradictory constraints to fire for valid configs
// that rely on type inference.
func buildSchemaMCPConfig(toolConfig map[string]any) map[string]any {
	result := make(map[string]any, len(mcpSchemaTopLevelFields))
	for field := range mcpSchemaTopLevelFields {
		if value, exists := toolConfig[field]; exists {
			result[field] = value
		}
	}
	// If 'type' is not present, infer it from other fields so the schema's
	// if/then conditions do not fire vacuously and reject valid inferred-type configs.
	//
	// Why this is necessary: the JSON Schema draft-07 `properties` keyword is
	// vacuously satisfied when the checked property is absent. This means the
	// `if {"properties": {"type": {"enum": ["stdio"]}}}` condition evaluates to
	// true even when 'type' is not in the config, causing the stdio `then` clause
	// (requiring command/container) to apply unexpectedly for HTTP-only configs.
	// Injecting the inferred type before schema validation ensures the correct
	// if/then branch fires. When inference is not possible (empty string returned),
	// the map is left without a 'type'; the schema's anyOf constraint will then
	// report a clear "missing required property" error on its own.
	if _, hasType := result["type"]; !hasType {
		if inferred := inferMCPType(result); inferred != "" {
			result["type"] = inferred
		}
	}
	return result
}

// validateMCPMountsSyntax validates that mount strings in a custom MCP server config
// follow the correct syntax required by MCP Gateway v0.1.5+.
// Expected format: "source:destination:mode" where mode is either "ro" or "rw".
func validateMCPMountsSyntax(toolName string, mountsRaw any) error {
	var mounts []string

	switch v := mountsRaw.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				mounts = append(mounts, s)
			}
		}
	case []string:
		mounts = v
	default:
		return fmt.Errorf("tool '%s' mcp configuration 'mounts' must be an array of strings.\n\nExample:\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    mounts:\n      - \"/host/path:/container/path:ro\"\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
	}

	for i, mount := range mounts {
		source, dest, mode, err := validateMountStringFormat(mount)
		if err != nil {
			if source == "" && dest == "" && mode == "" {
				return fmt.Errorf("tool '%s' mcp configuration mounts[%d] must follow 'source:destination:mode' format, got: %q.\n\nExample:\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    mounts:\n      - \"/host/path:/container/path:ro\"\n\nSee: %s", toolName, i, mount, toolName, constants.DocsToolsURL)
			}
			return fmt.Errorf("tool '%s' mcp configuration mounts[%d] mode must be 'ro' or 'rw', got: %q.\n\nExample:\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    mounts:\n      - \"/host/path:/container/path:ro\"  # read-only\n      - \"/host/path:/container/path:rw\"  # read-write\n\nSee: %s", toolName, i, mode, toolName, constants.DocsToolsURL)
		}
	}

	return nil
}
