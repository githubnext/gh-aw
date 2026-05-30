//go:build !integration

package workflow

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateStringProperty_ReturnsValidationErrorWithYAMLSuggestion(t *testing.T) {
	err := validateStringProperty("test-server", "url", nil, false)
	if err == nil {
		t.Fatal("expected missing property error")
	}

	var validationErr *WorkflowValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected WorkflowValidationError, got %T", err)
	}
	if validationErr.Suggestion == "" {
		t.Fatal("expected non-empty suggestion")
	}
	if !strings.Contains(validationErr.Suggestion, "tools:") || !strings.Contains(validationErr.Suggestion, "url:") {
		t.Fatalf("expected YAML suggestion, got: %s", validationErr.Suggestion)
	}
}

func TestValidateMCPRequirements_ReturnsValidationErrorWithYAMLSuggestion(t *testing.T) {
	err := validateMCPRequirements("test-server", map[string]any{"type": "stdio"}, map[string]any{"type": "stdio"})
	if err == nil {
		t.Fatal("expected stdio missing command/container error")
	}

	var validationErr *WorkflowValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected WorkflowValidationError, got %T", err)
	}
	if validationErr.Suggestion == "" {
		t.Fatal("expected non-empty suggestion")
	}
	if !strings.Contains(validationErr.Suggestion, "tools:") || !strings.Contains(validationErr.Suggestion, "command:") {
		t.Fatalf("expected YAML suggestion, got: %s", validationErr.Suggestion)
	}
}
