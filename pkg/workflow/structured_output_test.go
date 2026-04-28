//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractStructuredOutputConfig tests the compile-time parsing and validation of
// the structured-output frontmatter section.
func TestExtractStructuredOutputConfig(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantNil     bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "no structured-output field",
			frontmatter: map[string]any{},
			wantNil:     true,
		},
		{
			name:        "structured-output is nil",
			frontmatter: map[string]any{"structured-output": nil},
			wantNil:     true,
		},
		{
			name: "valid inline schema - object type",
			frontmatter: map[string]any{
				"structured-output": map[string]any{
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"decision": map[string]any{"type": "string"},
						},
						"required": []any{"decision"},
					},
				},
			},
			wantNil: false,
		},
		{
			name: "schema with enum constraint",
			frontmatter: map[string]any{
				"structured-output": map[string]any{
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"decision": map[string]any{
								"type": "string",
								"enum": []any{"APPROVE", "REQUEST_CHANGES", "ESCALATE"},
							},
							"confidence": map[string]any{
								"type":    "number",
								"minimum": 0,
								"maximum": 1,
							},
						},
						"required":             []any{"decision", "confidence"},
						"additionalProperties": false,
					},
				},
			},
			wantNil: false,
		},
		{
			name: "missing both schema and schema-file",
			frontmatter: map[string]any{
				"structured-output": map[string]any{},
			},
			wantErr:     true,
			errContains: "requires either 'schema'",
		},
		{
			name: "both schema and schema-file specified",
			frontmatter: map[string]any{
				"structured-output": map[string]any{
					"schema":      map[string]any{"type": "object"},
					"schema-file": ".github/schemas/output.json",
				},
			},
			wantErr:     true,
			errContains: "cannot specify both",
		},
		{
			name: "structured-output is not an object",
			frontmatter: map[string]any{
				"structured-output": "invalid",
			},
			wantErr:     true,
			errContains: "must be an object",
		},
		{
			name: "schema is not an object",
			frontmatter: map[string]any{
				"structured-output": map[string]any{
					"schema": "not-an-object",
				},
			},
			wantErr:     true,
			errContains: "must be a JSON Schema object",
		},
		{
			name: "schema-file is empty string",
			frontmatter: map[string]any{
				"structured-output": map[string]any{
					"schema-file": "",
				},
			},
			wantErr:     true,
			errContains: "must be a non-empty string path",
		},
		{
			name: "schema-file references non-existent file",
			frontmatter: map[string]any{
				"structured-output": map[string]any{
					"schema-file": "nonexistent-schema.json",
				},
			},
			wantErr:     true,
			errContains: "cannot read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := extractStructuredOutputConfig(tt.frontmatter, "/tmp")

			if tt.wantErr {
				require.Error(t, err, "expected an error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "error message should contain expected text")
				}
				return
			}

			require.NoError(t, err, "expected no error")
			if tt.wantNil {
				assert.Nil(t, config, "expected nil config")
			} else {
				require.NotNil(t, config, "expected non-nil config")
				assert.NotEmpty(t, config.GetResolvedSchemaJSON(), "expected resolved schema JSON to be set")
			}
		})
	}
}

// TestHasStructuredOutput tests the HasStructuredOutput helper function.
func TestHasStructuredOutput(t *testing.T) {
	assert.False(t, HasStructuredOutput(nil), "nil WorkflowData should return false")

	data := &WorkflowData{}
	assert.False(t, HasStructuredOutput(data), "WorkflowData without StructuredOutputConfig should return false")

	data.StructuredOutputConfig = &StructuredOutputConfig{
		resolvedSchemaJSON: `{"type":"object"}`,
	}
	assert.True(t, HasStructuredOutput(data), "WorkflowData with StructuredOutputConfig should return true")
}

// TestApplyStructuredOutputEnvToMap tests the environment variable injection.
func TestApplyStructuredOutputEnvToMap(t *testing.T) {
	t.Run("no-op when structured output not configured", func(t *testing.T) {
		env := make(map[string]string)
		applyStructuredOutputEnvToMap(env, &WorkflowData{})
		assert.Empty(t, env, "env should be unchanged when no structured output configured")
	})

	t.Run("no-op for detection runs", func(t *testing.T) {
		env := make(map[string]string)
		data := &WorkflowData{
			StructuredOutputConfig: &StructuredOutputConfig{resolvedSchemaJSON: `{"type":"object"}`},
			IsDetectionRun:         true,
		}
		applyStructuredOutputEnvToMap(env, data)
		assert.Empty(t, env, "env should be unchanged for detection runs")
	})

	t.Run("sets env vars for configured workflow", func(t *testing.T) {
		env := make(map[string]string)
		data := &WorkflowData{
			StructuredOutputConfig: &StructuredOutputConfig{resolvedSchemaJSON: `{"type":"object"}`},
		}
		applyStructuredOutputEnvToMap(env, data)
		assert.Equal(t, StructuredOutputSchemaPath, env["GH_AW_STRUCTURED_OUTPUT_SCHEMA"],
			"GH_AW_STRUCTURED_OUTPUT_SCHEMA should point to the schema file")
		assert.Equal(t, StructuredOutputFilePath, env["GH_AW_STRUCTURED_OUTPUT_FILE"],
			"GH_AW_STRUCTURED_OUTPUT_FILE should point to the output file")
	})
}

// TestGenerateStructuredOutputSchemaStep tests the pre-agent schema write step generation.
func TestGenerateStructuredOutputSchemaStep(t *testing.T) {
	t.Run("returns empty for unconfigured workflow", func(t *testing.T) {
		step := generateStructuredOutputSchemaStep(&WorkflowData{})
		assert.Empty(t, step, "step should be empty when structured output is not configured")
	})

	t.Run("generates step for configured workflow", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"decision": map[string]any{"type": "string"},
			},
		}
		config, err := extractStructuredOutputConfig(
			map[string]any{"structured-output": map[string]any{"schema": schema}},
			"/tmp",
		)
		require.NoError(t, err)

		data := &WorkflowData{StructuredOutputConfig: config}
		step := generateStructuredOutputSchemaStep(data)

		assert.NotEmpty(t, step, "step should not be empty for configured workflow")
		assert.Contains(t, step, "Set up structured output schema", "step should have correct name")
		assert.Contains(t, step, StructuredOutputSchemaPath, "step should reference schema path")
		assert.Contains(t, step, "mkdir -p /tmp/gh-aw", "step should create directory")
		assert.Contains(t, step, "printf", "step should use printf to write the schema")
	})

	t.Run("schema JSON is embedded in step", func(t *testing.T) {
		schema := map[string]any{"type": "string"}
		config, err := extractStructuredOutputConfig(
			map[string]any{"structured-output": map[string]any{"schema": schema}},
			"/tmp",
		)
		require.NoError(t, err)

		data := &WorkflowData{StructuredOutputConfig: config}
		step := generateStructuredOutputSchemaStep(data)

		// The step should contain "type" from the schema
		assert.Contains(t, step, "type", "schema content should appear in step")
	})
}

// TestGenerateStructuredOutputValidationStep tests the post-agent validation step generation.
func TestGenerateStructuredOutputValidationStep(t *testing.T) {
	t.Run("returns empty for unconfigured workflow", func(t *testing.T) {
		step := generateStructuredOutputValidationStep(&WorkflowData{}, nil)
		assert.Empty(t, step, "step should be empty when structured output is not configured")
	})

	t.Run("generates validation step", func(t *testing.T) {
		schema := map[string]any{"type": "object"}
		config, err := extractStructuredOutputConfig(
			map[string]any{"structured-output": map[string]any{"schema": schema}},
			"/tmp",
		)
		require.NoError(t, err)

		data := &WorkflowData{StructuredOutputConfig: config}
		step := generateStructuredOutputValidationStep(data, nil)

		assert.NotEmpty(t, step, "step should not be empty for configured workflow")
		assert.Contains(t, step, "Validate structured output", "step should have correct name")
		assert.Contains(t, step, "if: always()", "validation step should always run")
		assert.Contains(t, step, "id: validate_structured_output", "step should have correct id")
		assert.Contains(t, step, "actions/github-script", "step should use github-script")
		assert.Contains(t, step, StructuredOutputFilePath, "step should reference output file path")
		assert.Contains(t, step, "structured_output", "step should set structured_output output")
	})

	t.Run("uses custom action pin function", func(t *testing.T) {
		schema := map[string]any{"type": "object"}
		config, err := extractStructuredOutputConfig(
			map[string]any{"structured-output": map[string]any{"schema": schema}},
			"/tmp",
		)
		require.NoError(t, err)

		data := &WorkflowData{StructuredOutputConfig: config}
		customPin := func(repo string, _ *WorkflowData) string {
			return repo + "@abc123"
		}
		step := generateStructuredOutputValidationStep(data, customPin)

		assert.Contains(t, step, "actions/github-script@abc123", "step should use custom action pin")
	})
}

// TestShellEscapeSingleQuote tests the single-quote escaping helper.
func TestShellEscapeSingleQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"no quotes", "no quotes"},
		{"it's a test", `it'\''s a test`},
		{"a'b'c", `a'\''b'\''c`},
		{`{"key": "value"}`, `{"key": "value"}`},
		{`{"key": "it's here"}`, `{"key": "it'\''s here"}`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := shellEscapeSingleQuote(tt.input)
			assert.Equal(t, tt.expected, result, "single quote escaping should be correct")
		})
	}
}

// TestValidateJSONSchema tests the JSON Schema validation function.
func TestValidateJSONSchema(t *testing.T) {
	t.Run("valid minimal schema", func(t *testing.T) {
		schema := map[string]any{"type": "object"}
		err := validateJSONSchema(schema)
		assert.NoError(t, err, "minimal object schema should be valid")
	})

	t.Run("valid schema with properties", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "integer"},
			},
			"required": []any{"name"},
		}
		err := validateJSONSchema(schema)
		assert.NoError(t, err, "schema with properties should be valid")
	})

	t.Run("valid schema with enum", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{
					"type": "string",
					"enum": []any{"PENDING", "ACTIVE", "CLOSED"},
				},
			},
		}
		err := validateJSONSchema(schema)
		assert.NoError(t, err, "schema with enum should be valid")
	})

	t.Run("empty schema is valid", func(t *testing.T) {
		schema := map[string]any{}
		err := validateJSONSchema(schema)
		assert.NoError(t, err, "empty schema should be valid (matches anything)")
	})
}

// TestStructuredOutputSchemaWriteEscaping tests that schemas with special characters
// are correctly escaped when embedded in the shell step.
func TestStructuredOutputSchemaWriteEscaping(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "A message with 'single quotes' inside",
			},
		},
	}
	config, err := extractStructuredOutputConfig(
		map[string]any{"structured-output": map[string]any{"schema": schema}},
		"/tmp",
	)
	require.NoError(t, err, "schema with single quotes in description should parse")

	data := &WorkflowData{StructuredOutputConfig: config}
	step := generateStructuredOutputSchemaStep(data)

	// Verify the generated step is well-formed YAML (starts correctly)
	assert.True(t, strings.HasPrefix(step, "      - name:"), "step should start with YAML list item")
	// The escaped single quotes should appear as '\'' in the generated step
	assert.Contains(t, step, `'\''`, "single quotes in schema should be escaped as '\\''")
}
