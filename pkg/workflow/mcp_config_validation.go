// Package workflow provides MCP (Model Context Protocol) configuration validation.
//
// # MCP Configuration Validation
//
// This file validates MCP server configurations in agentic workflows.
// It ensures that MCP configurations have required fields, correct types,
// and satisfy type-specific requirements (stdio vs http).
//
// # Validation Functions
//
//   - ValidateMCPConfigs() - Validates all MCP configurations in tools section
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
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var mcpValidationLog = logger.New("workflow:mcp_config_validation")

// ValidateMCPConfigs validates all MCP configurations in the tools section using JSON schema
func ValidateMCPConfigs(tools map[string]any) error {
	mcpValidationLog.Printf("Validating MCP configurations for %d tools", len(tools))

	// List of built-in tools that have their own validation logic
	// These tools should not be validated as custom MCP servers
	builtInTools := map[string]bool{
		"github":            true,
		"playwright":        true,
		"serena":            true,
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

	for toolName, toolConfig := range tools {
		// Skip built-in tools - they have their own schema validation
		if builtInTools[toolName] {
			mcpValidationLog.Printf("Skipping MCP validation for built-in tool: %s", toolName)
			continue
		}

		if config, ok := toolConfig.(map[string]any); ok {
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

			// Validate MCP configuration requirements (before transformation)
			if err := validateMCPRequirements(toolName, mcpConfig, config); err != nil {
				return err
			}
		}
	}

	mcpValidationLog.Print("MCP configuration validation completed successfully")
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
		"entrypoint":      true,
		"entrypointArgs":  true,
		"mounts":          true,
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
			// Build list of valid fields for the error message
			validFields := []string{}
			for k := range knownToolFields {
				validFields = append(validFields, k)
			}
			sort.Strings(validFields)
			maxFields := 10
			if len(validFields) < maxFields {
				maxFields = len(validFields)
			}
			return nil, fmt.Errorf("🤔 Unknown property '%s' in your tool configuration.\n\nValid properties include: %s\n\nExample:\n  tools:\n    my-tool:\n      command: \"node server.js\"\n      args: [\"--verbose\"]", field, strings.Join(validFields[:maxFields], ", ")) // Show up to 10 to keep message reasonable
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
		return fmt.Errorf("💡 The MCP server '%s' is missing the '%s' property.\n\nExample:\n  tools:\n    %s:\n      %s: \"value\"", toolName, propertyName, toolName, propertyName)
	}
	if _, ok := value.(string); !ok {
		return fmt.Errorf("💡 The '%s' property for MCP server '%s' needs to be a string (got %T).\n\nExample:\n  tools:\n    %s:\n      %s: \"my-value\"", propertyName, toolName, value, toolName, propertyName)
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
			return fmt.Errorf("💡 The 'type' field for MCP server '%s' needs to be a string (got %T).\n\nValid types per MCP Gateway Specification:\n  • stdio - Local process communication\n  • http - HTTP-based communication\n\nNote: 'local' is accepted for backward compatibility and treated as 'stdio'.\n\nExample:\n  tools:\n    %s:\n      type: \"stdio\"\n      command: \"node server.js\"", toolName, mcpType, toolName)
		}
		typeStr = mcpType.(string)
	} else {
		// Infer type from presence of fields
		if _, hasURL := mcpConfig["url"]; hasURL {
			typeStr = "http"
		} else if _, hasCommand := mcpConfig["command"]; hasCommand {
			typeStr = "stdio"
		} else if _, hasContainer := mcpConfig["container"]; hasContainer {
			typeStr = "stdio"
		} else {
			return fmt.Errorf("💡 Unable to determine the MCP type for '%s'.\n\nMCP servers need one of these:\n  • type: \"stdio\" or \"http\"\n  • url: \"https://...\" (implies http)\n  • command: \"node ...\" (implies stdio)\n  • container: \"registry/image\" (implies stdio)\n\nExample:\n  tools:\n    %s:\n      command: \"node server.js\"\n      args: [\"--port\", \"3000\"]\n\nLearn more: https://githubnext.github.io/gh-aw/guides/mcp-servers/", toolName, toolName)
		}
	}

	// Normalize "local" to "stdio" for validation
	if typeStr == "local" {
		typeStr = "stdio"
	}

	// Validate type is one of the supported types
	if !parser.IsMCPType(typeStr) {
		return fmt.Errorf("💡 The MCP type '%s' for server '%s' isn't recognized.\n\nValid types per MCP Gateway Specification:\n  • stdio - Local process communication\n  • http - HTTP-based communication\n\nNote: 'local' is accepted for backward compatibility and treated as 'stdio'.\n\nExample:\n  tools:\n    %s:\n      type: \"stdio\"\n      command: \"node server.js\"\n\nLearn more: https://githubnext.github.io/gh-aw/guides/mcp-servers/", typeStr, toolName, toolName)
	}

	// Validate type-specific requirements
	switch typeStr {
	case "http":
		// HTTP type requires 'url' property
		url, hasURL := mcpConfig["url"]

		// HTTP type cannot use container field
		if _, hasContainer := mcpConfig["container"]; hasContainer {
			return fmt.Errorf("💡 HTTP MCP servers like '%s' can't use the 'container' field.\n\nWhy? HTTP MCP uses URL endpoints, not containers.\n\nExample:\n  tools:\n    %s:\n      type: http\n      url: \"https://api.example.com/mcp\"\n      headers:\n        Authorization: \"Bearer ${{ secrets.API_KEY }}\"", toolName, toolName)
		}

		return validateStringProperty(toolName, "url", url, hasURL)

	case "stdio":
		// stdio type requires either 'command' or 'container' property (but not both)
		command, hasCommand := mcpConfig["command"]
		container, hasContainer := mcpConfig["container"]

		if hasCommand && hasContainer {
			return fmt.Errorf("💡 MCP server '%s' has both 'command' and 'container' specified.\n\nWhy? We need to know exactly one way to start your MCP server.\n\nChoose one approach:\n\nOption 1 - Use command:\n  tools:\n    %s:\n      command: \"node server.js\"\n\nOption 2 - Use container:\n  tools:\n    %s:\n      container: \"my-registry/my-tool\"\n      version: \"latest\"", toolName, toolName, toolName)
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
			return fmt.Errorf("💡 MCP server '%s' needs a way to start.\n\nStdio MCP servers need either 'command' or 'container' specified.\n\nExample with command:\n  tools:\n    %s:\n      command: \"node server.js\"\n      args: [\"--port\", \"3000\"]\n\nOr with container:\n  tools:\n    %s:\n      container: \"my-registry/my-tool\"\n      version: \"latest\"\n\nLearn more: https://githubnext.github.io/gh-aw/guides/mcp-servers/", toolName, toolName, toolName)
		}
	}

	return nil
}
