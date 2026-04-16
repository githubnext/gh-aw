package workflow

import "github.com/github/gh-aw/pkg/logger"

var buildInputSchemaLog = logger.New("workflow:build_input_schema")

// buildInputSchema converts GitHub Actions input definitions (workflow_dispatch,
// workflow_call, or dispatch_repository inputs) into JSON Schema properties and
// a required field list suitable for MCP tool inputSchema.
//
// descriptionFn is called to produce the fallback description when an input
// definition does not include its own "description" field.
//
// Supported input types: string (default), number, boolean, choice, environment.
// Choice inputs with options are mapped to a string enum. Unknown types default
// to string.
func buildInputSchema(inputs map[string]any, descriptionFn func(inputName string) string) (properties map[string]any, required []string) {
	buildInputSchemaLog.Printf("Building input schema for %d inputs", len(inputs))
	properties = make(map[string]any)
	required = []string{}

	for inputName, inputDef := range inputs {
		inputDefMap, ok := inputDef.(map[string]any)
		if !ok {
			buildInputSchemaLog.Printf("Skipping input %q: expected map, got %T", inputName, inputDef)
			continue
		}

		inputType := "string"
		inputDescription := descriptionFn(inputName)
		inputRequired := false

		if desc, ok := inputDefMap["description"].(string); ok && desc != "" {
			inputDescription = desc
		}

		if req, ok := inputDefMap["required"].(bool); ok {
			inputRequired = req
		}

		// Map GitHub Actions input types to JSON Schema types.
		if typeStr, ok := inputDefMap["type"].(string); ok {
			switch typeStr {
			case "number":
				inputType = "number"
			case "boolean":
				inputType = "boolean"
			case "choice":
				inputType = "string"
				if options, ok := inputDefMap["options"].([]any); ok && len(options) > 0 {
					prop := map[string]any{
						"type":        inputType,
						"description": inputDescription,
						"enum":        options,
					}
					if defaultVal, ok := inputDefMap["default"]; ok {
						prop["default"] = defaultVal
					}
					properties[inputName] = prop
					if inputRequired {
						required = append(required, inputName)
					}
					continue
				}
			case "environment":
				inputType = "string"
			}
		}

		prop := map[string]any{
			"type":        inputType,
			"description": inputDescription,
		}
		if defaultVal, ok := inputDefMap["default"]; ok {
			prop["default"] = defaultVal
		}
		buildInputSchemaLog.Printf("Input %q: type=%s, required=%v", inputName, inputType, inputRequired)
		properties[inputName] = prop

		if inputRequired {
			required = append(required, inputName)
		}
	}

	buildInputSchemaLog.Printf("Built input schema: %d properties, %d required", len(properties), len(required))
	return properties, required
}
