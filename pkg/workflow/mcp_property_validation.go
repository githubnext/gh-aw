// This file provides focused MCP property and schema-shaping validation helpers.
//
// # MCP Property Validation
//
//   - validateStringProperty() - Validates required MCP string properties
//   - validateMCPRequirements() - Validates type-specific MCP requirements
//   - buildSchemaMCPConfig() - Projects tool config to schema-compatible MCP fields
//
// This file contains validation details used by entry points in
// mcp_config_validation.go.

package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

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
