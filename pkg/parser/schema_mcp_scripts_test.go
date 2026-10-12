//go:build !integration

package parser

import "testing"

func TestValidateMCPScriptArrayItems(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{"string items", map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, false},
		{"number items", map[string]any{"type": "array", "items": map[string]any{"type": "number"}}, false},
		{"boolean items", map[string]any{"type": "array", "items": map[string]any{"type": "boolean"}}, false},
		{"array items", map[string]any{"type": "array", "items": map[string]any{"type": "array"}}, false},
		{"object items", map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, false},
		{"unconstrained array", map[string]any{"type": "array"}, false},
		{"default string", map[string]any{}, false},
		{"nonarray", map[string]any{"type": "string", "items": map[string]any{"type": "string"}}, true},
		{"missing array type", map[string]any{"items": map[string]any{"type": "string"}}, true},
		{"missing item type", map[string]any{"type": "array", "items": map[string]any{}}, true},
		{"invalid item type", map[string]any{"type": "array", "items": map[string]any{"type": "invalid"}}, true},
		{"nonstring item type", map[string]any{"type": "array", "items": map[string]any{"type": 1}}, true},
		{"nonobject items", map[string]any{"type": "array", "items": "string"}, true},
		{"unsupported item constraint", map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []any{"test"}}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tools := map[string]any{
				"example": map[string]any{
					"description": "Array input",
					"inputs":      map[string]any{"args": test.input},
					"script":      "return args;",
				},
			}
			for _, validator := range []struct {
				name     string
				validate func(map[string]any, string) error
				base     map[string]any
			}{
				{"workflow", ValidateMainWorkflowFrontmatterWithSchemaAndLocation, map[string]any{"on": "push", "engine": "copilot"}},
				{"import", ValidateIncludedFileFrontmatterWithSchemaAndLocation, map[string]any{}},
			} {
				t.Run(validator.name, func(t *testing.T) {
					validator.base["mcp-scripts"] = tools
					err := validator.validate(validator.base, "workflow.md")
					if (err != nil) != test.wantErr {
						t.Fatalf("validation error = %v, wantErr %t", err, test.wantErr)
					}
				})
			}
		})
	}
}
