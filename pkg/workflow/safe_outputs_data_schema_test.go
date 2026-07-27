//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSafeOutputsDataSchemaInlineShorthand(t *testing.T) {
	cfg := &SafeOutputsConfig{
		Data: map[string]any{
			"verdict":         "string",
			"criteria_passed": "number",
		},
	}

	err := validateSafeOutputsDataSchema(cfg)
	require.NoError(t, err)
	assert.True(t, cfg.DataEnabled)
	require.NotNil(t, cfg.NormalizedDataSchema)
	assert.Equal(t, "object", cfg.NormalizedDataSchema["type"])
	assert.Equal(t, false, cfg.NormalizedDataSchema["additionalProperties"])
	assert.Equal(t, []string{"criteria_passed", "verdict"}, cfg.NormalizedDataSchema["required"])
	properties, ok := cfg.NormalizedDataSchema["properties"].(map[string]any)
	require.True(t, ok)
	verdict, ok := properties["verdict"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", verdict["type"])
}

func TestValidateSafeOutputsDataSchemaRejectsInvalidDataType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data any
	}{
		{name: "string path", data: "data-schema.json"},
		{name: "numeric", data: 42},
		{name: "array", data: []any{"string"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &SafeOutputsConfig{Data: tc.data}

			err := validateSafeOutputsDataSchema(cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "safe-outputs.data")
		})
	}
}

func TestValidateSafeOutputsDataSchemaRejectsUnsupportedKeyword(t *testing.T) {
	cfg := &SafeOutputsConfig{
		Data: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"verdict": map[string]any{
					"type": "string",
					"$ref": "#/definitions/other",
				},
			},
		},
	}

	err := validateSafeOutputsDataSchema(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported keyword")
}

func TestValidateSafeOutputsDataSchemaRejectsAdditionalPropertiesTrue(t *testing.T) {
	cfg := &SafeOutputsConfig{
		Data: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"verdict": "string",
			},
			"additionalProperties": true,
		},
	}

	err := validateSafeOutputsDataSchema(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be false for OpenAI Codex structured outputs compatibility")
}

func TestValidateSafeOutputsDataSchemaDisabledByDefault(t *testing.T) {
	cfg := &SafeOutputsConfig{}
	err := validateSafeOutputsDataSchema(cfg)
	require.NoError(t, err)
	assert.False(t, cfg.DataEnabled)
	assert.Nil(t, cfg.NormalizedDataSchema)
}

func TestValidateSafeOutputsDataSchemaBooleanTrueAllowsAnyObject(t *testing.T) {
	cfg := &SafeOutputsConfig{Data: true}
	err := validateSafeOutputsDataSchema(cfg)
	require.NoError(t, err)
	assert.True(t, cfg.DataEnabled)
	assert.Nil(t, cfg.NormalizedDataSchema)
}

func TestValidateSafeOutputsDataSchemaAllowsExpression(t *testing.T) {
	cfg := &SafeOutputsConfig{Data: "${{ fromJSON(inputs.safe_outputs_data_schema) }}"}
	err := validateSafeOutputsDataSchema(cfg)
	require.NoError(t, err)
	assert.True(t, cfg.DataEnabled)
	assert.Nil(t, cfg.NormalizedDataSchema)
	assert.Equal(t, "${{ fromJSON(inputs.safe_outputs_data_schema) }}", cfg.DataSchemaExpression)
}
