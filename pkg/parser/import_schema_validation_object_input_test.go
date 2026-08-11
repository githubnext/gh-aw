//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateObjectInput_NotAnObject(t *testing.T) {
	err := validateObjectInput("config", "not-an-object", map[string]any{}, "org/repo/workflow.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be an object")
	assert.Contains(t, err.Error(), "config")
	assert.Contains(t, err.Error(), "org/repo/workflow.md")
}

func TestValidateObjectInput_NoPropertiesDeclared_AcceptsAnyObject(t *testing.T) {
	value := map[string]any{"anything": "goes", "num": 42}
	err := validateObjectInput("config", value, map[string]any{}, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_MalformedPropertiesField_FallsBackToPermissive(t *testing.T) {
	paramDef := map[string]any{"properties": "not-a-map"}
	value := map[string]any{"key": "value"}
	err := validateObjectInput("config", value, paramDef, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_UnknownSubKey(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"known": map[string]any{"type": "string"},
		},
	}
	value := map[string]any{"unknown": "value"}
	err := validateObjectInput("config", value, paramDef, "import")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property")
	assert.Contains(t, err.Error(), "unknown")
}

func TestValidateObjectInput_RequiredSubFieldMissing(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{"required": true, "type": "string"},
		},
	}
	value := map[string]any{}
	err := validateObjectInput("config", value, paramDef, "import")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required property")
	assert.Contains(t, err.Error(), "name")
}

func TestValidateObjectInput_RequiredSubFieldPresent(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{"required": true, "type": "string"},
		},
	}
	value := map[string]any{"name": "hello"}
	err := validateObjectInput("config", value, paramDef, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_OptionalSubFieldMissing_NoError(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"optional": map[string]any{"type": "string"},
		},
	}
	value := map[string]any{}
	err := validateObjectInput("config", value, paramDef, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_NoTypeDeclared_SkipsTypeValidation(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"field": map[string]any{},
		},
	}
	value := map[string]any{"field": 12345} // any type accepted since no "type" declared
	err := validateObjectInput("config", value, paramDef, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_TypeMismatch(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"count": map[string]any{"type": "number"},
		},
	}
	value := map[string]any{"count": "not-a-number"}
	err := validateObjectInput("config", value, paramDef, "import")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config.count")
}

func TestValidateObjectInput_TypeMatch(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		typeName string
		value    any
	}{
		{name: "number", field: "count", typeName: "number", value: 3},
		{name: "string", field: "label", typeName: "string", value: "hi"},
		{name: "boolean", field: "flag", typeName: "boolean", value: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramDef := map[string]any{
				"properties": map[string]any{
					tt.field: map[string]any{"type": tt.typeName},
				},
			}
			value := map[string]any{tt.field: tt.value}
			assert.NoError(t, validateObjectInput("config", value, paramDef, "import"))
		})
	}
}

func TestValidateObjectInput_PropDefNotAMap_SkipsValidation(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"weird": "not-a-map",
		},
	}
	value := map[string]any{"weird": "value"}
	err := validateObjectInput("config", value, paramDef, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_QualifiedNameInErrorMessage(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"nested": map[string]any{"type": "choice", "options": []any{"a", "b"}},
		},
	}
	value := map[string]any{"nested": "c"}
	err := validateObjectInput("parentField", value, paramDef, "import/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parentField.nested")
}

func TestValidateObjectInput_Pure(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"known": map[string]any{"type": "string"},
		},
	}
	value := map[string]any{"unknown": "value"}
	wantParamDef := map[string]any{
		"properties": map[string]any{
			"known": map[string]any{"type": "string"},
		},
	}
	wantValue := map[string]any{"unknown": "value"}

	first := validateObjectInput("config", value, paramDef, "import")
	second := validateObjectInput("config", value, paramDef, "import")

	require.Error(t, first)
	require.EqualError(t, second, first.Error())
	assert.Equal(t, wantParamDef, paramDef)
	assert.Equal(t, wantValue, value)
}
