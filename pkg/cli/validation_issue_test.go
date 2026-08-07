//go:build !integration

package cli

import (
	"encoding/json"
	"testing"
)

func TestValidationIssueJSONCompatibility(t *testing.T) {
	compileResult := ValidationResult{
		Workflow: "test.md",
		Valid:    false,
		Errors: []ValidationIssue{{
			Type:    "schema_validation",
			Message: "Unknown property",
			Line:    5,
		}},
		Warnings: []ValidationIssue{{
			Type:    "warning",
			Message: "Deprecated field",
		}},
	}

	compileJSON, err := json.Marshal(compileResult)
	if err != nil {
		t.Fatalf("marshal compile result: %v", err)
	}

	var compilePayload map[string]any
	if err := json.Unmarshal(compileJSON, &compilePayload); err != nil {
		t.Fatalf("unmarshal compile result: %v", err)
	}
	errors := compilePayload["errors"].([]any)
	compileIssue := errors[0].(map[string]any)
	if compileIssue["type"] != "schema_validation" {
		t.Fatalf("expected type to be serialized, got %#v", compileIssue["type"])
	}
	if compileIssue["message"] != "Unknown property" {
		t.Fatalf("expected message to be serialized, got %#v", compileIssue["message"])
	}
	if compileIssue["line"] != float64(5) {
		t.Fatalf("expected line to be serialized, got %#v", compileIssue["line"])
	}
	if _, ok := compileIssue["file"]; ok {
		t.Fatalf("did not expect compile issue file field when empty: %#v", compileIssue)
	}
	warnings := compilePayload["warnings"].([]any)
	compileWarning := warnings[0].(map[string]any)
	if _, ok := compileWarning["line"]; ok {
		t.Fatalf("did not expect warning line field when zero: %#v", compileWarning)
	}

	auditData := AuditData{
		Errors: []ValidationIssue{{
			Type:    "step_failure",
			Message: "Step failed",
			Line:    12,
			File:    "workflow/job/12_step.txt",
		}},
	}

	auditJSON, err := json.Marshal(auditData)
	if err != nil {
		t.Fatalf("marshal audit data: %v", err)
	}

	var auditPayload map[string]any
	if err := json.Unmarshal(auditJSON, &auditPayload); err != nil {
		t.Fatalf("unmarshal audit data: %v", err)
	}
	auditErrors := auditPayload["errors"].([]any)
	auditIssue := auditErrors[0].(map[string]any)
	if auditIssue["file"] != "workflow/job/12_step.txt" {
		t.Fatalf("expected file to be serialized for audit issue, got %#v", auditIssue["file"])
	}
	if auditIssue["line"] != float64(12) {
		t.Fatalf("expected audit line to be serialized, got %#v", auditIssue["line"])
	}
}
