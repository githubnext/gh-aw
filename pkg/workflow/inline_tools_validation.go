// Package workflow provides inline tools validation functions for agentic workflow compilation.
//
// This file contains domain-specific validation functions for inline tool configuration:
//   - validateInlineTools() - Validates inline tools can only be used in SDK mode
//   - validateInlineToolDefinition() - Validates individual inline tool definitions
//
// Inline tools allow defining custom tools directly in the workflow frontmatter
// with JavaScript/TypeScript implementation. They are only supported in SDK mode
// to provide proper sandboxing and execution context.
package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var inlineToolsValidationLog = logger.New("workflow:inline_tools_validation")

// validateInlineTools validates that inline tools are only used in SDK mode
// and that all inline tool definitions are valid.
func validateInlineTools(workflowData *WorkflowData) error {
	if workflowData == nil {
		return nil
	}

	// Check if inline tools are present
	if workflowData.ParsedTools == nil || len(workflowData.ParsedTools.Inline) == 0 {
		inlineToolsValidationLog.Print("No inline tools defined, skipping validation")
		return nil
	}

	inlineTools := workflowData.ParsedTools.Inline
	inlineToolsValidationLog.Printf("Validating %d inline tools", len(inlineTools))

	// Inline tools are currently only supported in SDK mode (future implementation)
	// For now, we reject them with a clear error message
	// TODO: Update this validation once SDK mode is implemented
	var engine string
	if workflowData.EngineConfig != nil {
		engine = workflowData.EngineConfig.ID
	}
	if engine == "" {
		engine = "copilot" // default engine
	}

	// Check if this is SDK mode (currently not implemented, so we reject all inline tools)
	// SDK mode would be indicated by a specific engine configuration like "copilot-sdk" or mode field
	// For now, we reject inline tools in all cases with a helpful message
	return fmt.Errorf(
		"inline tools are not yet supported - they will be available in SDK mode (coming soon)\n"+
			"Found %d inline tool(s): %s\n"+
			"For now, please use MCP servers instead by defining tools in the 'tools:' section",
		len(inlineTools),
		getInlineToolNames(inlineTools),
	)
}

// validateInlineToolDefinition validates a single inline tool definition
// Checks that required fields are present and valid.
func validateInlineToolDefinition(tool InlineToolConfig, index int) error {
	inlineToolsValidationLog.Printf("Validating inline tool at index %d: name=%s", index, tool.Name)

	// Validate required field: name
	if tool.Name == "" {
		return fmt.Errorf("inline tool at index %d is missing required field 'name'", index)
	}

	// Validate name format (must be a valid identifier)
	if !isValidToolName(tool.Name) {
		return fmt.Errorf(
			"inline tool at index %d has invalid name '%s': "+
				"tool names must start with a letter and contain only letters, numbers, underscores, and hyphens",
			index, tool.Name,
		)
	}

	// Validate required field: description
	if tool.Description == "" {
		return fmt.Errorf("inline tool '%s' at index %d is missing required field 'description'", tool.Name, index)
	}

	// Validate that description is meaningful (at least 10 characters)
	if len(strings.TrimSpace(tool.Description)) < 10 {
		return fmt.Errorf(
			"inline tool '%s' at index %d has invalid description: "+
				"description must be at least 10 characters to be meaningful",
			tool.Name, index,
		)
	}

	// Validate parameters schema if present
	if tool.Parameters != nil {
		if err := validateParametersSchema(tool.Parameters, tool.Name, index); err != nil {
			return err
		}
	}

	// Implementation is optional (can be provided at runtime in SDK mode)
	// But if provided, it should not be empty
	if tool.Implementation != "" {
		impl := strings.TrimSpace(tool.Implementation)
		if len(impl) == 0 {
			return fmt.Errorf(
				"inline tool '%s' at index %d has empty implementation: "+
					"either provide implementation code or omit the field",
				tool.Name, index,
			)
		}
	}

	inlineToolsValidationLog.Printf("Validated inline tool '%s': description=%d chars, has_params=%v, has_impl=%v",
		tool.Name, len(tool.Description), tool.Parameters != nil, tool.Implementation != "")

	return nil
}

// validateParametersSchema validates that the parameters field contains a valid JSON Schema
func validateParametersSchema(params map[string]any, toolName string, index int) error {
	// Check for required JSON Schema fields
	schemaType, hasType := params["type"]
	if !hasType {
		return fmt.Errorf(
			"inline tool '%s' at index %d has invalid parameters schema: "+
				"JSON Schema must have a 'type' field (e.g., 'type: object')",
			toolName, index,
		)
	}

	// Validate type is a string
	typeStr, ok := schemaType.(string)
	if !ok {
		return fmt.Errorf(
			"inline tool '%s' at index %d has invalid parameters schema: "+
				"'type' field must be a string, got %T",
			toolName, index, schemaType,
		)
	}

	// Validate type is a valid JSON Schema type
	validTypes := []string{"object", "array", "string", "number", "integer", "boolean", "null"}
	isValidType := false
	for _, validType := range validTypes {
		if typeStr == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return fmt.Errorf(
			"inline tool '%s' at index %d has invalid parameters schema: "+
				"'type' must be one of %v, got '%s'",
			toolName, index, validTypes, typeStr,
		)
	}

	// If type is object, check for properties (common pattern)
	if typeStr == "object" {
		if properties, hasProperties := params["properties"]; hasProperties {
			// Validate properties is a map
			if _, ok := properties.(map[string]any); !ok {
				return fmt.Errorf(
					"inline tool '%s' at index %d has invalid parameters schema: "+
						"'properties' field must be an object, got %T",
					toolName, index, properties,
				)
			}
		}

		// If required field exists, validate it's an array
		if required, hasRequired := params["required"]; hasRequired {
			if _, ok := required.([]any); !ok {
				// Also accept []string for convenience
				if _, ok := required.([]string); !ok {
					return fmt.Errorf(
						"inline tool '%s' at index %d has invalid parameters schema: "+
							"'required' field must be an array, got %T",
						toolName, index, required,
					)
				}
			}
		}
	}

	inlineToolsValidationLog.Printf("Validated parameters schema for tool '%s': type=%s", toolName, typeStr)
	return nil
}

// isValidToolName checks if a tool name is a valid identifier
// Tool names must start with a letter and contain only letters, numbers, underscores, and hyphens
func isValidToolName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// First character must be a letter
	firstChar := rune(name[0])
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')) {
		return false
	}

	// Remaining characters must be letters, numbers, underscores, or hyphens
	for _, char := range name[1:] {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' ||
			char == '-') {
			return false
		}
	}

	return true
}

// getInlineToolNames returns a comma-separated list of inline tool names
func getInlineToolNames(tools []InlineToolConfig) string {
	if len(tools) == 0 {
		return ""
	}

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool.Name != "" {
			names = append(names, tool.Name)
		}
	}

	return strings.Join(names, ", ")
}

// checkInlineToolNameUniqueness validates that all inline tool names are unique
// and don't conflict with built-in tool names
func checkInlineToolNameUniqueness(tools []InlineToolConfig, allTools *ToolsConfig) error {
	if len(tools) == 0 {
		return nil
	}

	// Track seen tool names
	seenNames := make(map[string]int)

	// Check for duplicates within inline tools
	for i, tool := range tools {
		if tool.Name == "" {
			continue // Will be caught by validateInlineToolDefinition
		}

		if prevIndex, exists := seenNames[tool.Name]; exists {
			return fmt.Errorf(
				"duplicate inline tool name '%s' at indices %d and %d: "+
					"each tool must have a unique name",
				tool.Name, prevIndex, i,
			)
		}
		seenNames[tool.Name] = i
	}

	// Check for conflicts with built-in tools
	if allTools == nil {
		return nil
	}

	builtinTools := []string{
		"github", "bash", "web-fetch", "web-search", "edit",
		"playwright", "serena", "agentic-workflows",
		"cache-memory", "repo-memory", "timeout", "startup-timeout",
	}

	for _, tool := range tools {
		for _, builtin := range builtinTools {
			if tool.Name == builtin {
				return fmt.Errorf(
					"inline tool name '%s' conflicts with built-in tool: "+
						"choose a different name that doesn't conflict with built-in tools",
					tool.Name,
				)
			}
		}

		// Check for conflicts with custom MCP tools
		if allTools.Custom != nil {
			if _, exists := allTools.Custom[tool.Name]; exists {
				return fmt.Errorf(
					"inline tool name '%s' conflicts with custom MCP tool: "+
						"each tool must have a unique name across all tool types",
					tool.Name,
				)
			}
		}
	}

	inlineToolsValidationLog.Printf("Validated uniqueness of %d inline tool names", len(tools))
	return nil
}
