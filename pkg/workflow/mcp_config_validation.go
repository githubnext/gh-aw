// This file contains MCP (Model Context Protocol) validation entry points.
//
// # MCP Validation Entry Points
//
//   - ValidateMCPConfigs() - Orchestrates MCP validation for merged tools map entries
//   - ValidateToolsSection() - Validates built-in tool names in tools:
//   - getRawMCPConfig() - Extracts raw MCP config fields and checks unknown properties
//   - inferMCPType() - Infers MCP type (http/stdio) from present fields
//
// Detailed property and mount validation helpers are intentionally split into:
//   - mcp_property_validation.go
//   - mcp_mount_validation.go

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
	"comment-memory":    true,
	"repo-memory":       true,
	"bash":              true,
	"edit":              true,
	"web-fetch":         true,
	"web-search":        true,
	"safety-prompt":     true,
	"timeout":           true,
	"startup-timeout":   true,
	"mount-as-clis":     true,
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
