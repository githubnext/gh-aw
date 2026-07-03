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
		prop, inputType, inputRequired, ok := buildInputSchemaProperty(inputName, inputDef, descriptionFn)
		if !ok {
			continue
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

func buildInputSchemaProperty(inputName string, inputDef any, descriptionFn func(inputName string) string) (prop map[string]any, inputType string, inputRequired bool, ok bool) {
	inputDefMap, ok := inputDef.(map[string]any)
	if !ok {
		buildInputSchemaLog.Printf("Skipping input %q: expected map, got %T", inputName, inputDef)
		return nil, "", false, false
	}

	inputType = "string"
	inputDescription, inputRequired := getInputSchemaMetadata(inputName, inputDefMap, descriptionFn)
	if typeStr, ok := inputDefMap["type"].(string); ok {
		switch typeStr {
		case "number":
			inputType = "number"
		case "boolean":
			inputType = "boolean"
		case "choice":
			if options, ok := inputDefMap["options"].([]any); ok && len(options) > 0 {
				prop = newInputSchemaProperty(inputType, inputDescription, inputDefMap)
				prop["enum"] = options
				return prop, inputType, inputRequired, true
			}
		}
	}

	return newInputSchemaProperty(inputType, inputDescription, inputDefMap), inputType, inputRequired, true
}

func getInputSchemaMetadata(inputName string, inputDefMap map[string]any, descriptionFn func(inputName string) string) (string, bool) {
	inputDescription := descriptionFn(inputName)
	if desc, ok := inputDefMap["description"].(string); ok && desc != "" {
		inputDescription = desc
	}
	inputRequired := false
	if req, ok := inputDefMap["required"].(bool); ok {
		inputRequired = req
	}
	return inputDescription, inputRequired
}

func newInputSchemaProperty(inputType, inputDescription string, inputDefMap map[string]any) map[string]any {
	prop := map[string]any{
		"type":        inputType,
		"description": inputDescription,
	}
	if defaultVal, ok := inputDefMap["default"]; ok {
		prop["default"] = defaultVal
	}
	return prop
}
