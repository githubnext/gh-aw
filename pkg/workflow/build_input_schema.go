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
				if prop, isChoice := buildChoiceInputProperty(inputDefMap, inputDescription); isChoice {
					properties[inputName] = prop
					if inputRequired {
						required = append(required, inputName)
					}
					continue
				}
				inputType = "string"
			case "environment":
				inputType = "string"
			}
		}

		prop := buildInputProperty(inputType, inputDescription, inputDefMap)
		buildInputSchemaLog.Printf("Input %q: type=%s, required=%v", inputName, inputType, inputRequired)
		properties[inputName] = prop

		if inputRequired {
			required = append(required, inputName)
		}
	}

	buildInputSchemaLog.Printf("Built input schema: %d properties, %d required", len(properties), len(required))
	return properties, required
}

// buildChoiceInputProperty builds a JSON Schema property for a "choice" input type.
// It returns (property, true) when options are available, or (nil, false) otherwise.
func buildChoiceInputProperty(inputDefMap map[string]any, inputDescription string) (map[string]any, bool) {
	options, ok := inputDefMap["options"].([]any)
	if !ok || len(options) == 0 {
		return nil, false
	}
	prop := map[string]any{
		"type":        "string",
		"description": inputDescription,
		"enum":        options,
	}
	if defaultVal, ok := inputDefMap["default"]; ok {
		prop["default"] = defaultVal
	}
	return prop, true
}

// buildInputProperty builds a JSON Schema property map for a scalar input (string,
// number, or boolean). Use buildChoiceInputProperty for "choice" inputs instead.
func buildInputProperty(inputType, inputDescription string, inputDefMap map[string]any) map[string]any {
	prop := map[string]any{
		"type":        inputType,
		"description": inputDescription,
	}
	if defaultVal, ok := inputDefMap["default"]; ok {
		prop["default"] = defaultVal
	}
	return prop
}
