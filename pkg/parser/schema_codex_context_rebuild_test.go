//go:build !integration

package parser

import "testing"

func TestMainWorkflowSchema_MaxContextRebuild(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		value any
		valid bool
	}{
		{name: "minimum", value: 1, valid: true},
		{name: "integer", value: 50, valid: true},
		{name: "fractional factor", value: 29.5, valid: true},
		{name: "zero", value: 0},
		{name: "negative", value: -1},
		{name: "unreachable factor", value: 0.5},
		{name: "numeric string", value: "50"},
		{name: "expression", value: "${{ inputs.max-context-rebuild }}"},
		{name: "boolean", value: true},
		{name: "null", value: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			frontmatter := map[string]any{
				"on": "workflow_dispatch",
				"engine": map[string]any{
					"id":                  "codex",
					"max-context-rebuild": test.value,
				},
			}
			err := validateWithSchema(frontmatter, mainWorkflowSchema, "main workflow file")
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, got schema error: %v", test.valid, err)
			}
		})
	}
}

func TestMainWorkflowSchema_DeclarativeMaxContextRebuild(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		capability any
		env        any
		valid      bool
	}{
		{name: "supported", capability: true, env: "REBUILD_FACTOR_LIMIT", valid: true},
		{name: "string capability", capability: "true", env: "REBUILD_FACTOR_LIMIT"},
		{name: "invalid env name", capability: true, env: "REBUILD-FACTOR-LIMIT"},
		{name: "empty env name", capability: true, env: ""},
		{name: "numeric env name", capability: true, env: 50},
	} {
		t.Run(test.name, func(t *testing.T) {
			frontmatter := map[string]any{
				"on": "workflow_dispatch",
				"engine": map[string]any{
					"id":           "rebuild-test",
					"display-name": "Rebuild Test",
					"behaviors": map[string]any{
						"capabilities": map[string]any{"max-context-rebuild": test.capability},
						"execution":    map[string]any{"command-name": "rebuild-cli", "max-context-rebuild-env-var": test.env},
					},
				},
			}
			err := validateWithSchema(frontmatter, mainWorkflowSchema, "main workflow file")
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, got schema error: %v", test.valid, err)
			}
		})
	}
}
