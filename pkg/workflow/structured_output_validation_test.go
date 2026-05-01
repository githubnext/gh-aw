//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestValidateStructuredOutput_NilConfig(t *testing.T) {
	workflowData := &WorkflowData{
		AI: "codex",
	}
	if err := validateStructuredOutput(workflowData); err != nil {
		t.Errorf("Expected no error for nil structured output, got: %v", err)
	}
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
	if err := validateStructuredOutput(workflowData); err != nil {
		t.Errorf("Expected no error for codex with inline schema, got: %v", err)
	}
}

func TestValidateStructuredOutput_CodexWithSchemaFile(t *testing.T) {
	workflowData := &WorkflowData{
		AI: "codex",
		StructuredOutput: &StructuredOutputConfig{
			SchemaFile: ".github/schemas/output.schema.json",
		},
	}
	if err := validateStructuredOutput(workflowData); err != nil {
		t.Errorf("Expected no error for codex with schema-file, got: %v", err)
	}
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
			if err == nil {
				t.Errorf("Expected error for %s engine with structured-output, got nil", engine)
				return
			}
			if !strings.Contains(err.Error(), "only supported with the codex engine") {
				t.Errorf("Expected error to mention codex engine, got: %v", err)
			}
			if !strings.Contains(err.Error(), engine) {
				t.Errorf("Expected error to mention engine name %q, got: %v", engine, err)
			}
		})
	}
}

func TestValidateStructuredOutput_MissingSchemaAndSchemaFile(t *testing.T) {
	workflowData := &WorkflowData{
		AI:               "codex",
		StructuredOutput: &StructuredOutputConfig{},
	}
	err := validateStructuredOutput(workflowData)
	if err == nil {
		t.Error("Expected error when neither schema nor schema-file is set, got nil")
	}
	if !strings.Contains(err.Error(), "requires either") {
		t.Errorf("Expected error about missing schema/schema-file, got: %v", err)
	}
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
	if err == nil {
		t.Error("Expected error when both schema and schema-file are set, got nil")
	}
	if !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("Expected error about conflicting schema/schema-file, got: %v", err)
	}
}
