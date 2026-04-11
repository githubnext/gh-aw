//go:build !integration

package workflow

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultDescFn is a simple description function used in tests.
func defaultDescFn(inputName string) string {
	return fmt.Sprintf("Input parameter '%s'", inputName)
}

// TestBuildInputSchemaStringType tests that the default type is string.
func TestBuildInputSchemaStringType(t *testing.T) {
	inputs := map[string]any{
		"message": map[string]any{
			"type":        "string",
			"description": "A message",
		},
	}

	properties, required := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["message"].(map[string]any)
	require.True(t, ok, "message property should exist")
	assert.Equal(t, "string", prop["type"], "type should be string")
	assert.Equal(t, "A message", prop["description"], "description should match")
	assert.Empty(t, required, "required should be empty")
}

// TestBuildInputSchemaNumberType tests number type mapping.
func TestBuildInputSchemaNumberType(t *testing.T) {
	inputs := map[string]any{
		"count": map[string]any{
			"type":        "number",
			"description": "A count",
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["count"].(map[string]any)
	require.True(t, ok, "count property should exist")
	assert.Equal(t, "number", prop["type"], "type should be number")
}

// TestBuildInputSchemaBooleanType tests boolean type mapping.
func TestBuildInputSchemaBooleanType(t *testing.T) {
	inputs := map[string]any{
		"dry_run": map[string]any{
			"type": "boolean",
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["dry_run"].(map[string]any)
	require.True(t, ok, "dry_run property should exist")
	assert.Equal(t, "boolean", prop["type"], "type should be boolean")
}

// TestBuildInputSchemaChoiceType tests choice type with enum options.
func TestBuildInputSchemaChoiceType(t *testing.T) {
	inputs := map[string]any{
		"environment": map[string]any{
			"type":        "choice",
			"description": "Target environment",
			"options":     []any{"staging", "production"},
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["environment"].(map[string]any)
	require.True(t, ok, "environment property should exist")
	assert.Equal(t, "string", prop["type"], "choice maps to string")
	assert.Equal(t, []any{"staging", "production"}, prop["enum"], "enum values should match")
}

// TestBuildInputSchemaChoiceWithDefault tests that choice type preserves default.
func TestBuildInputSchemaChoiceWithDefault(t *testing.T) {
	inputs := map[string]any{
		"environment": map[string]any{
			"type":        "choice",
			"description": "Target environment",
			"options":     []any{"staging", "production"},
			"default":     "staging",
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["environment"].(map[string]any)
	require.True(t, ok, "environment property should exist")
	assert.Equal(t, "string", prop["type"], "choice maps to string")
	assert.Equal(t, []any{"staging", "production"}, prop["enum"], "enum values should match")
	assert.Equal(t, "staging", prop["default"], "default should be preserved for choice")
}

// TestBuildInputSchemaEnvironmentType tests environment type mapping.
func TestBuildInputSchemaEnvironmentType(t *testing.T) {
	inputs := map[string]any{
		"deploy_env": map[string]any{
			"type":        "environment",
			"description": "Deployment environment",
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["deploy_env"].(map[string]any)
	require.True(t, ok, "deploy_env property should exist")
	assert.Equal(t, "string", prop["type"], "environment maps to string")
}

// TestBuildInputSchemaDefaultPropagation tests that default values are propagated.
func TestBuildInputSchemaDefaultPropagation(t *testing.T) {
	inputs := map[string]any{
		"version": map[string]any{
			"type":        "string",
			"description": "Version",
			"default":     "latest",
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["version"].(map[string]any)
	require.True(t, ok, "version property should exist")
	assert.Equal(t, "latest", prop["default"], "default value should be set")
}

// TestBuildInputSchemaRequiredInputs tests required field tracking.
func TestBuildInputSchemaRequiredInputs(t *testing.T) {
	inputs := map[string]any{
		"name": map[string]any{
			"type":     "string",
			"required": true,
		},
		"optional": map[string]any{
			"type":     "string",
			"required": false,
		},
	}

	_, required := buildInputSchema(inputs, defaultDescFn)

	assert.Contains(t, required, "name", "name should be required")
	assert.NotContains(t, required, "optional", "optional should not be required")
}

// TestBuildInputSchemaRequiredSorting tests that required list order is deterministic
// across multiple runs (callers typically sort the result).
func TestBuildInputSchemaRequiredSorting(t *testing.T) {
	inputs := map[string]any{
		"z_param": map[string]any{"type": "string", "required": true},
		"a_param": map[string]any{"type": "string", "required": true},
		"m_param": map[string]any{"type": "string", "required": true},
	}

	for i := range 10 {
		_, required := buildInputSchema(inputs, defaultDescFn)
		sort.Strings(required)
		assert.Equal(t, []string{"a_param", "m_param", "z_param"}, required,
			"required should be sortable to deterministic order (iteration %d)", i)
	}
}

// TestBuildInputSchemaSkipsInvalidDefs tests that non-map input definitions are skipped.
func TestBuildInputSchemaSkipsInvalidDefs(t *testing.T) {
	inputs := map[string]any{
		"valid": map[string]any{
			"type":        "string",
			"description": "Valid input",
		},
		"invalid_string": "not a map",
		"invalid_number": 42,
		"invalid_nil":    nil,
	}

	properties, required := buildInputSchema(inputs, defaultDescFn)

	assert.Len(t, properties, 1, "Only valid input should produce a property")
	assert.Contains(t, properties, "valid", "valid input should be present")
	assert.Empty(t, required, "no required inputs")
}

// TestBuildInputSchemaEmptyInputs tests that empty inputs produce empty results.
func TestBuildInputSchemaEmptyInputs(t *testing.T) {
	properties, required := buildInputSchema(make(map[string]any), defaultDescFn)

	assert.Empty(t, properties, "properties should be empty")
	assert.Empty(t, required, "required should be empty")
}

// TestBuildInputSchemaNilInputs tests that nil inputs produce empty results.
func TestBuildInputSchemaNilInputs(t *testing.T) {
	properties, required := buildInputSchema(nil, defaultDescFn)

	assert.Empty(t, properties, "properties should be empty")
	assert.Empty(t, required, "required should be empty")
}

// TestBuildInputSchemaFallbackDescription tests the descriptionFn callback.
func TestBuildInputSchemaFallbackDescription(t *testing.T) {
	inputs := map[string]any{
		"param": map[string]any{
			"type": "string",
			// No description provided - should use fallback
		},
	}

	descFn := func(inputName string) string {
		return fmt.Sprintf("Custom fallback for '%s'", inputName)
	}

	properties, _ := buildInputSchema(inputs, descFn)

	prop, ok := properties["param"].(map[string]any)
	require.True(t, ok, "param property should exist")
	assert.Equal(t, "Custom fallback for 'param'", prop["description"], "should use fallback description")
}

// TestBuildInputSchemaChoiceWithoutOptions tests choice type without options falls back to string.
func TestBuildInputSchemaChoiceWithoutOptions(t *testing.T) {
	inputs := map[string]any{
		"env": map[string]any{
			"type":        "choice",
			"description": "Environment",
			// No options - should fall through to regular string property
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["env"].(map[string]any)
	require.True(t, ok, "env property should exist")
	assert.Equal(t, "string", prop["type"], "choice without options maps to string")
	_, hasEnum := prop["enum"]
	assert.False(t, hasEnum, "should not have enum when no options")
}

// TestBuildInputSchemaUnknownType tests that unknown type defaults to string.
func TestBuildInputSchemaUnknownType(t *testing.T) {
	inputs := map[string]any{
		"param": map[string]any{
			"type":        "unknown_type",
			"description": "Some param",
		},
	}

	properties, _ := buildInputSchema(inputs, defaultDescFn)

	prop, ok := properties["param"].(map[string]any)
	require.True(t, ok, "param property should exist")
	assert.Equal(t, "string", prop["type"], "unknown type should default to string")
}
