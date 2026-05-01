//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStructuredOutput_NilConfig(t *testing.T) {
	workflowData := &WorkflowData{
		AI: "codex",
	}
	assert.NoError(t, validateStructuredOutput(workflowData), "nil structured output should not error")
}

func TestValidateStructuredOutput_CodexWithInlineSchema(t *testing.T) {
	workflowData := &WorkflowData{
		AI: "codex",
		StructuredOutput: &StructuredOutputConfig{
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"decision": map[string]any{"type": "string"},
				},
			},
		},
	}
	assert.NoError(t, validateStructuredOutput(workflowData), "codex with inline schema should not error")
}

func TestValidateStructuredOutput_CodexWithSchemaFile(t *testing.T) {
	workflowData := &WorkflowData{
		AI: "codex",
		StructuredOutput: &StructuredOutputConfig{
			SchemaFile: ".github/schemas/output.schema.json",
		},
	}
	assert.NoError(t, validateStructuredOutput(workflowData), "codex with schema-file should not error")
}

func TestValidateStructuredOutput_NonCodexEngine(t *testing.T) {
	engines := []string{"copilot", "claude", "gemini", "custom"}
	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			workflowData := &WorkflowData{
				AI: engine,
				StructuredOutput: &StructuredOutputConfig{
					Schema: map[string]any{"type": "object"},
				},
			}
			err := validateStructuredOutput(workflowData)
			require.Error(t, err, "structured-output with %s engine should error", engine)
			assert.Contains(t, err.Error(), "only supported with the codex engine", "error should mention codex engine")
			assert.Contains(t, err.Error(), engine, "error should mention current engine name")
		})
	}
}

func TestValidateStructuredOutput_MissingSchemaAndSchemaFile(t *testing.T) {
	workflowData := &WorkflowData{
		AI:               "codex",
		StructuredOutput: &StructuredOutputConfig{},
	}
	err := validateStructuredOutput(workflowData)
	require.Error(t, err, "missing both schema and schema-file should error")
	assert.Contains(t, err.Error(), "requires either", "error should describe missing requirement")
}

func TestValidateStructuredOutput_BothSchemaAndSchemaFile(t *testing.T) {
	workflowData := &WorkflowData{
		AI: "codex",
		StructuredOutput: &StructuredOutputConfig{
			Schema:     map[string]any{"type": "object"},
			SchemaFile: ".github/schemas/output.schema.json",
		},
	}
	err := validateStructuredOutput(workflowData)
	require.Error(t, err, "setting both schema and schema-file should error")
	assert.Contains(t, err.Error(), "cannot specify both", "error should describe conflicting fields")
}
