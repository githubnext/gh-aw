package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateObjectInput_NotAnObject(t *testing.T) {
	err := validateObjectInput("config", "not-an-object", map[string]any{}, "org/repo/workflow.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be an object")
	assert.Contains(t, err.Error(), "config")
	assert.Contains(t, err.Error(), "org/repo/workflow.md")
}

func TestValidateObjectInput_NoPropertiesDeclared_AcceptsAnyObject(t *testing.T) {
	value := map[string]any{"anything": "goes", "num": 42}
	err := validateObjectInput("config", value, map[string]any{}, "import")
	assert.NoError(t, err)
}

func TestValidateObjectInput_PropertiesNotAMap_AcceptsAnyObject(t *testing.T) {
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
	assert.Error(t, err)
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
	assert.Error(t, err)
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config.count")
}

func TestValidateObjectInput_TypeMatch(t *testing.T) {
	paramDef := map[string]any{
		"properties": map[string]any{
			"count": map[string]any{"type": "number"},
			"label": map[string]any{"type": "string"},
			"flag":  map[string]any{"type": "boolean"},
		},
	}
	value := map[string]any{"count": 3, "label": "hi", "flag": true}
	err := validateObjectInput("config", value, paramDef, "import")
	assert.NoError(t, err)
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parentField.nested")
}
