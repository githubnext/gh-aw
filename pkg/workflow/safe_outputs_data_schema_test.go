//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
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

	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
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

func TestValidateSafeOutputsDataSchemaFile(t *testing.T) {
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "data-schema.json")
	err := os.WriteFile(schemaPath, []byte(`{
  "type": "object",
  "properties": {
    "verdict": { "type": "string", "enum": ["APPROVE", "REJECT"] },
    "score": { "type": "integer" }
  },
  "required": ["verdict"],
  "additionalProperties": false
}`), 0o600)
	require.NoError(t, err)

	cfg := &SafeOutputsConfig{
		Data: "data-schema.json",
	}

	err = validateSafeOutputsDataSchema(cfg, filepath.Join(tempDir, "workflow.md"))
	require.NoError(t, err)
	assert.True(t, cfg.DataEnabled)
	require.NotNil(t, cfg.NormalizedDataSchema)
	assert.Equal(t, "object", cfg.NormalizedDataSchema["type"])
	assert.Equal(t, false, cfg.NormalizedDataSchema["additionalProperties"])
	assert.Equal(t, []string{"score", "verdict"}, cfg.NormalizedDataSchema["required"])
	properties, ok := cfg.NormalizedDataSchema["properties"].(map[string]any)
	require.True(t, ok)
	score, ok := properties["score"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", score["type"])
}

func TestValidateSafeOutputsDataSchemaMutuallyExclusiveSources(t *testing.T) {
	cfg := &SafeOutputsConfig{
		Data:           true,
		DataSchema:     map[string]any{"verdict": "string"},
		DataSchemaFile: "data-schema.json",
	}

	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined")
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

	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
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

	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be false for OpenAI Codex structured outputs compatibility")
}

func TestValidateSafeOutputsDataSchemaDisabledByDefault(t *testing.T) {
	cfg := &SafeOutputsConfig{}
	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
	require.NoError(t, err)
	assert.False(t, cfg.DataEnabled)
	assert.Nil(t, cfg.NormalizedDataSchema)
}

func TestValidateSafeOutputsDataSchemaBooleanTrueAllowsAnyObject(t *testing.T) {
	cfg := &SafeOutputsConfig{Data: true}
	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
	require.NoError(t, err)
	assert.True(t, cfg.DataEnabled)
	assert.Nil(t, cfg.NormalizedDataSchema)
}
