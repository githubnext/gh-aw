//go:build !integration

package parser

import "testing"

func TestCompileSchema(t *testing.T) {
	t.Parallel()

	schema, err := CompileSchema(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"],
		"additionalProperties": false
	}`, "http://example.com/schema.json")
	if err != nil {
		t.Fatalf("CompileSchema returned error: %v", err)
	}

	if err := schema.Validate(map[string]any{"name": "gh-aw"}); err != nil {
		t.Fatalf("Validate(valid) returned error: %v", err)
	}

	if err := schema.Validate(map[string]any{"name": 1}); err == nil {
		t.Fatal("Validate(invalid) returned nil error, want validation failure")
	}
}
