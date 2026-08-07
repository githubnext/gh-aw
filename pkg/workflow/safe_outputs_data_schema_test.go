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

func TestValidateSafeOutputsDataSchemaAllowsJSONStringSchema(t *testing.T) {
	cfg := &SafeOutputsConfig{
		Data: `{"type":"object","properties":{"verdict":{"type":"string"}},"required":["verdict"],"additionalProperties":false}`,
	}
	err := validateSafeOutputsDataSchema(cfg)
	require.NoError(t, err)
	assert.True(t, cfg.DataEnabled)
	require.NotNil(t, cfg.NormalizedDataSchema)
	assert.Equal(t, "object", cfg.NormalizedDataSchema["type"])
}

func TestValidateSafeOutputsDataSchemaRejectsInvalidStringSyntax(t *testing.T) {
	cfg := &SafeOutputsConfig{Data: "not json and not expression"}
	err := validateSafeOutputsDataSchema(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a GitHub Actions expression or JSON object schema")
}

func TestSimplifyDataSchemaNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		raw            any
		path           string
		allowShorthand bool
		wantErr        string
		check          func(t *testing.T, result map[string]any)
	}{
		{
			name:           "string shorthand supported type produces type-only schema",
			raw:            "string",
			path:           "root",
			allowShorthand: true,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, map[string]any{"type": "string"}, result)
			},
		},
		{
			name:           "string shorthand disallowed returns error",
			raw:            "string",
			path:           "root",
			allowShorthand: false,
			wantErr:        "root: string shorthand is not allowed here",
		},
		{
			name:           "string shorthand unsupported type returns error",
			raw:            "unsupported",
			path:           "root",
			allowShorthand: true,
			wantErr:        `root: unsupported type "unsupported"`,
		},
		{
			name:           "non-object non-string raw returns error",
			raw:            42,
			path:           "root",
			allowShorthand: true,
			wantErr:        "root: expected an object schema",
		},
		{
			name:           "empty map with shorthand disallowed returns error",
			raw:            map[string]any{},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root: expected JSON schema keywords or shorthand properties",
		},
		{
			name: "shorthand object properties expands to object schema",
			raw: map[string]any{
				"verdict": "string",
				"score":   "number",
			},
			path:           "root",
			allowShorthand: true,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "object", result["type"])
				assert.Equal(t, false, result["additionalProperties"])
				assert.Equal(t, []string{"score", "verdict"}, result["required"])
				properties, ok := result["properties"].(map[string]any)
				require.True(t, ok)
				assert.Len(t, properties, 2)
			},
		},
		{
			name: "unsupported keyword returns error",
			raw: map[string]any{
				"type":    "string",
				"bananas": "yellow",
			},
			path:           "root",
			allowShorthand: true,
			wantErr:        `root: unsupported keyword "bananas"`,
		},
		{
			name: "unsupported type value returns error",
			raw: map[string]any{
				"type": "widget",
			},
			path:           "root",
			allowShorthand: true,
			wantErr:        `root.type: unsupported type "widget"`,
		},
		{
			name: "inferred object type from properties without explicit type",
			raw: map[string]any{
				"properties": map[string]any{
					"a": "string",
				},
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "object", result["type"])
			},
		},
		{
			name: "inferred array type from items without explicit type",
			raw: map[string]any{
				"items": "string",
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "array", result["type"])
				items, ok := result["items"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "string", items["type"])
			},
		},
		{
			name: "description must be a string",
			raw: map[string]any{
				"type":        "string",
				"description": 123,
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.description: must be a string",
		},
		{
			name: "description passed through when a string",
			raw: map[string]any{
				"type":        "string",
				"description": "a verdict field",
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "a verdict field", result["description"])
			},
		},
		{
			name: "enum must be a non-empty array",
			raw: map[string]any{
				"type": "string",
				"enum": []any{},
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.enum: must be a non-empty array",
		},
		{
			name: "enum wrong type returns error",
			raw: map[string]any{
				"type": "string",
				"enum": "not-an-array",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.enum: must be a non-empty array",
		},
		{
			name: "enum with non-scalar item returns error",
			raw: map[string]any{
				"type": "string",
				"enum": []any{"ok", []any{"nested"}},
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.enum[1]: must be a scalar value",
		},
		{
			name: "enum with valid scalar values passes through",
			raw: map[string]any{
				"type": "string",
				"enum": []any{"a", 1, 1.5, true, int64(2)},
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, []any{"a", 1, 1.5, true, int64(2)}, result["enum"])
			},
		},
		{
			name: "object without properties returns error",
			raw: map[string]any{
				"type": "object",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.properties: is required for object schemas",
		},
		{
			name: "object with non-map properties returns error",
			raw: map[string]any{
				"type":       "object",
				"properties": "nope",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.properties: must be an object",
		},
		{
			name: "object property schema error propagates",
			raw: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"bad": 42,
				},
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.properties.bad: expected an object schema",
		},
		{
			name: "object required must be an array",
			raw: map[string]any{
				"type":       "object",
				"properties": map[string]any{"a": "string"},
				"required":   "a",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.required: must be an array of strings",
		},
		{
			name: "object required item must be a non-empty string",
			raw: map[string]any{
				"type":       "object",
				"properties": map[string]any{"a": "string"},
				"required":   []any{"  "},
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.required[0]: must be a non-empty string",
		},
		{
			name: "object required item non-string type returns error",
			raw: map[string]any{
				"type":       "object",
				"properties": map[string]any{"a": "string"},
				"required":   []any{42},
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.required[0]: must be a non-empty string",
		},
		{
			name: "object required unknown property returns error",
			raw: map[string]any{
				"type":       "object",
				"properties": map[string]any{"a": "string"},
				"required":   []any{"missing"},
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        `root.required[0]: unknown property "missing"`,
		},
		{
			name: "object all properties forced required and sorted",
			raw: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"zeta":  "string",
					"alpha": "number",
				},
				"required": []any{"zeta"},
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, []string{"alpha", "zeta"}, result["required"])
			},
		},
		{
			name: "object additionalProperties must be boolean",
			raw: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"a": "string"},
				"additionalProperties": "yes",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.additionalProperties: must be boolean",
		},
		{
			name: "object additionalProperties true rejected",
			raw: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"a": "string"},
				"additionalProperties": true,
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.additionalProperties: must be false for OpenAI Codex structured outputs compatibility",
		},
		{
			name: "object additionalProperties false explicit is kept",
			raw: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"a": "string"},
				"additionalProperties": false,
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, false, result["additionalProperties"])
			},
		},
		{
			name: "object additionalProperties defaults to false when absent",
			raw: map[string]any{
				"type":       "object",
				"properties": map[string]any{"a": "string"},
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, false, result["additionalProperties"])
			},
		},
		{
			name: "array without items returns error",
			raw: map[string]any{
				"type": "array",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        "root.items: is required for array schemas",
		},
		{
			name: "array items error propagates",
			raw: map[string]any{
				"type":  "array",
				"items": "notatype",
			},
			path:           "root",
			allowShorthand: false,
			wantErr:        `root.items: unsupported type "notatype"`,
		},
		{
			name: "array with valid items normalizes recursively",
			raw: map[string]any{
				"type":  "array",
				"items": "integer",
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				items, ok := result["items"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "integer", items["type"])
			},
		},
		{
			name: "string keeps minLength maxLength and pattern",
			raw: map[string]any{
				"type":      "string",
				"minLength": 1,
				"maxLength": 10,
				"pattern":   "^[a-z]+$",
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, 1, result["minLength"])
				assert.Equal(t, 10, result["maxLength"])
				assert.Equal(t, "^[a-z]+$", result["pattern"])
			},
		},
		{
			name: "string without extra keywords has only type",
			raw: map[string]any{
				"type": "string",
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, map[string]any{"type": "string"}, result)
			},
		},
		{
			name: "number keeps minimum and maximum",
			raw: map[string]any{
				"type":    "number",
				"minimum": 0,
				"maximum": 100,
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, 0, result["minimum"])
				assert.Equal(t, 100, result["maximum"])
			},
		},
		{
			name: "integer keeps minimum and maximum",
			raw: map[string]any{
				"type":    "integer",
				"minimum": -5,
				"maximum": 5,
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, -5, result["minimum"])
				assert.Equal(t, 5, result["maximum"])
			},
		},
		{
			name: "boolean type with no extra keywords",
			raw: map[string]any{
				"type": "boolean",
			},
			path:           "root",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, map[string]any{"type": "boolean"}, result)
			},
		},
		{
			name: "unicode description and property names pass through",
			raw: map[string]any{
				"properties": map[string]any{
					"日本語": "string",
				},
			},
			path:           "根",
			allowShorthand: false,
			check: func(t *testing.T, result map[string]any) {
				properties, ok := result["properties"].(map[string]any)
				require.True(t, ok)
				_, exists := properties["日本語"]
				assert.True(t, exists)
				assert.Equal(t, []string{"日本語"}, result["required"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := simplifyDataSchemaNode(tt.raw, tt.path, tt.allowShorthand)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
