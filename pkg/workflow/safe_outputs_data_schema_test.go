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
		DataSchema: map[string]any{
			"verdict":         "string",
			"criteria_passed": "number",
		},
	}

	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
	require.NoError(t, err)
	require.NotNil(t, cfg.NormalizedDataSchema)
	assert.Equal(t, "object", cfg.NormalizedDataSchema["type"])
	assert.Equal(t, false, cfg.NormalizedDataSchema["additionalProperties"])
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
		DataSchemaFile: "data-schema.json",
	}

	err = validateSafeOutputsDataSchema(cfg, filepath.Join(tempDir, "workflow.md"))
	require.NoError(t, err)
	require.NotNil(t, cfg.NormalizedDataSchema)
	assert.Equal(t, "object", cfg.NormalizedDataSchema["type"])
	properties, ok := cfg.NormalizedDataSchema["properties"].(map[string]any)
	require.True(t, ok)
	score, ok := properties["score"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", score["type"])
}

func TestValidateSafeOutputsDataSchemaMutuallyExclusiveSources(t *testing.T) {
	cfg := &SafeOutputsConfig{
		DataSchema:     map[string]any{"verdict": "string"},
		DataSchemaFile: "data-schema.json",
	}

	err := validateSafeOutputsDataSchema(cfg, "/tmp/workflow.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestValidateSafeOutputsDataSchemaRejectsUnsupportedKeyword(t *testing.T) {
	cfg := &SafeOutputsConfig{
		DataSchema: map[string]any{
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
