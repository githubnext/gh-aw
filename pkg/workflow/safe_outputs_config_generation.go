package workflow

import (
	"fmt"
	"sort"
)

// generateCustomJobToolDefinition creates an MCP tool definition for a custom safe-output job
// Returns a map representing the tool definition in MCP format with name, description, and inputSchema
func generateCustomJobToolDefinition(jobName string, jobConfig *SafeJobConfig) map[string]any {
	safeOutputsConfigLog.Printf("Generating tool definition for custom job: %s", jobName)

	description := jobConfig.Description
	if description == "" {
		description = fmt.Sprintf("Execute the %s custom job", jobName)
	}

	inputSchema := map[string]any{
		"type":                 "object",
		"properties":           make(map[string]any),
		"additionalProperties": false,
	}

	var requiredFields []string
	properties := inputSchema["properties"].(map[string]any)

	for inputName, inputDef := range jobConfig.Inputs {
		property := map[string]any{}

		if inputDef.Description != "" {
			property["description"] = inputDef.Description
		}

		// Convert type to JSON Schema type
		switch inputDef.Type {
		case "choice":
			// Choice inputs are strings with enum constraints
			property["type"] = "string"
			if len(inputDef.Options) > 0 {
				property["enum"] = inputDef.Options
			}
		case "boolean":
			property["type"] = "boolean"
		case "number":
			property["type"] = "number"
		default:
			// "string", empty string, or any unknown type defaults to string
			property["type"] = "string"
		}

		if inputDef.Default != nil {
			property["default"] = inputDef.Default
		}

		if inputDef.Required {
			requiredFields = append(requiredFields, inputName)
		}

		properties[inputName] = property
	}

	if len(requiredFields) > 0 {
		sort.Strings(requiredFields)
		inputSchema["required"] = requiredFields
	}

	safeOutputsConfigLog.Printf("Generated tool definition for %s with %d inputs, %d required",
		jobName, len(jobConfig.Inputs), len(requiredFields))

	return map[string]any{
		"name":        jobName,
		"description": description,
		"inputSchema": inputSchema,
	}
}
