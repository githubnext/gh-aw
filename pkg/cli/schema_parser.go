package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var schemaParserLog = logger.New("cli:schema_parser")

// SafeOutputTypeSchema represents a safe output type definition from the schema
type SafeOutputTypeSchema struct {
	TypeName    string              // e.g., "create_issue"
	Title       string              // e.g., "Create Issue Output"
	Description string              // Schema description
	Properties  map[string]Property // Schema properties
	Required    []string            // Required property names
}

// Property represents a property in the schema
type Property struct {
	Type        any    // string, array, object, or map for oneOf/anyOf
	Description string
	Const       string // For type discriminator
	Items       any    // For array types
	MinLength   *int   // Validation
	MinItems    *int   // Validation for arrays
	Enum        []string
}

// AgentOutputSchema represents the full agent-output.json schema
type AgentOutputSchema struct {
	Definitions map[string]TypeDefinition `json:"$defs"`
}

// TypeDefinition represents a type definition in the schema
type TypeDefinition struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required"`
}

// LoadSafeOutputSchema loads and parses the agent-output.json schema
func LoadSafeOutputSchema(schemaPath string) (*AgentOutputSchema, error) {
	schemaParserLog.Printf("Loading schema from %s", schemaPath)

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema AgentOutputSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	schemaParserLog.Printf("Loaded schema with %d type definitions", len(schema.Definitions))
	return &schema, nil
}

// GetSafeOutputTypes extracts all safe output type definitions from the schema
// Returns only output types (those ending in "Output"), excluding the SafeOutput union type
func GetSafeOutputTypes(schema *AgentOutputSchema) []SafeOutputTypeSchema {
	var types []SafeOutputTypeSchema

	for defName, def := range schema.Definitions {
		// Skip non-output types and the SafeOutput union type
		if !strings.HasSuffix(defName, "Output") || defName == "SafeOutput" {
			continue
		}

		// Extract the type name from the type discriminator in properties
		typeName := extractTypeNameFromDefinition(def)
		if typeName == "" {
			schemaParserLog.Printf("Warning: Could not extract type name from %s", defName)
			continue
		}

		// Parse properties
		properties := parseProperties(def.Properties)

		types = append(types, SafeOutputTypeSchema{
			TypeName:    typeName,
			Title:       def.Title,
			Description: def.Description,
			Properties:  properties,
			Required:    def.Required,
		})
	}

	schemaParserLog.Printf("Extracted %d safe output types", len(types))
	return types
}

// extractTypeNameFromDefinition extracts the type name from the "type" property's "const" value
func extractTypeNameFromDefinition(def TypeDefinition) string {
	if def.Properties == nil {
		return ""
	}

	typeProperty, exists := def.Properties["type"]
	if !exists {
		return ""
	}

	// Type property should have a "const" field with the type name
	typePropMap, ok := typeProperty.(map[string]interface{})
	if !ok {
		return ""
	}

	constValue, exists := typePropMap["const"]
	if !exists {
		return ""
	}

	constStr, ok := constValue.(string)
	if !ok {
		return ""
	}

	return constStr
}

// parseProperties converts schema properties to our Property type
func parseProperties(props map[string]interface{}) map[string]Property {
	result := make(map[string]Property)

	for name, propData := range props {
		propMap, ok := propData.(map[string]interface{})
		if !ok {
			continue
		}

		prop := Property{}

		// Extract type
		if typeVal, exists := propMap["type"]; exists {
			prop.Type = typeVal
		}

		// Extract const (for type discriminator)
		if constVal, exists := propMap["const"]; exists {
			if constStr, ok := constVal.(string); ok {
				prop.Const = constStr
			}
		}

		// Extract description
		if desc, exists := propMap["description"]; exists {
			if descStr, ok := desc.(string); ok {
				prop.Description = descStr
			}
		}

		// Extract minLength
		if minLen, exists := propMap["minLength"]; exists {
			if minLenFloat, ok := minLen.(float64); ok {
				minLenInt := int(minLenFloat)
				prop.MinLength = &minLenInt
			}
		}

		// Extract minItems
		if minItems, exists := propMap["minItems"]; exists {
			if minItemsFloat, ok := minItems.(float64); ok {
				minItemsInt := int(minItemsFloat)
				prop.MinItems = &minItemsInt
			}
		}

		// Extract enum values
		if enumVal, exists := propMap["enum"]; exists {
			if enumArray, ok := enumVal.([]interface{}); ok {
				for _, item := range enumArray {
					if itemStr, ok := item.(string); ok {
						prop.Enum = append(prop.Enum, itemStr)
					}
				}
			}
		}

		// Extract items (for array types)
		if items, exists := propMap["items"]; exists {
			prop.Items = items
		}

		result[name] = prop
	}

	return result
}

// GetJavaScriptFilename returns the expected JavaScript filename for a safe output type
// e.g., "create_issue" -> "create_issue.cjs"
func GetJavaScriptFilename(typeName string) string {
	return typeName + ".cjs"
}

// ShouldGenerateCustomAction determines if a type should have a custom action generated
// Currently we generate custom actions for simpler types that don't require complex dependencies
func ShouldGenerateCustomAction(typeName string) bool {
	// List of types that should have custom actions
	customActionTypes := map[string]bool{
		"noop":                true,
		"minimize_comment":    true,
		"close_issue":         true,
		"close_pull_request":  true,
		"close_discussion":    true,
		"add_comment":         true,
		"create_issue":        true,
		"add_labels":          true,
		"create_discussion":   true,
		"update_issue":        true,
		"update_pull_request": true,
	}

	return customActionTypes[typeName]
}
