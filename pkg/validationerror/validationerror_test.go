//go:build !integration

package validationerror_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/validationerror"
)

// exampleError is a minimal concrete type embedding validationerror.Payload,
// mirroring how parser.ValidationError and workflow.WorkflowValidationError
// each embed the shared payload.
type exampleError struct {
	validationerror.Payload
}

func (e *exampleError) Error() string {
	return validationerror.Format(fmt.Sprintf("Validation failed for field '%s'", e.Field), e.Payload, false)
}

func TestFormat_IncludesValueReasonAndSuggestion(t *testing.T) {
	msg := validationerror.Format("Validation failed for field 'skills'", validationerror.Payload{
		Field:      "skills",
		Value:      "docs",
		Reason:     "duplicate name already defined",
		Suggestion: "Rename one of the duplicate skills or remove the extra `docs` definition.",
	}, false)

	for _, want := range []string{
		"Validation failed for field 'skills'",
		"Value: docs",
		"Reason: duplicate name already defined",
		"Suggestion: Rename one of the duplicate skills",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected formatted message to contain %q, got: %s", want, msg)
		}
	}
}

func TestFormat_TruncatesValueWhenRequested(t *testing.T) {
	longValue := strings.Repeat("x", 200)
	msg := validationerror.Format("header", validationerror.Payload{Value: longValue, Reason: "reason"}, true)

	if strings.Contains(msg, longValue) {
		t.Errorf("expected long value to be truncated, got: %s", msg)
	}
}

func TestFormat_OmitsEmptyValueAndSuggestion(t *testing.T) {
	msg := validationerror.Format("header", validationerror.Payload{Reason: "reason only"}, false)

	if strings.Contains(msg, "Value:") {
		t.Errorf("expected no Value section for empty value, got: %s", msg)
	}
	if strings.Contains(msg, "Suggestion:") {
		t.Errorf("expected no Suggestion section for empty suggestion, got: %s", msg)
	}
}

func TestValidationError_ErrorsAsWorksAcrossConcreteTypes(t *testing.T) {
	err := error(&exampleError{Payload: validationerror.Payload{
		Field:      "skills",
		Value:      "docs",
		Reason:     "duplicate name already defined",
		Suggestion: "Rename one of the duplicate skills.",
	}})

	var ve validationerror.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected errors.As to match validationerror.ValidationError interface")
	}

	if ve.ValidationField() != "skills" {
		t.Errorf("expected field 'skills', got %q", ve.ValidationField())
	}
	if ve.ValidationValue() != "docs" {
		t.Errorf("expected value 'docs', got %q", ve.ValidationValue())
	}
	if ve.ValidationReason() != "duplicate name already defined" {
		t.Errorf("expected reason 'duplicate name already defined', got %q", ve.ValidationReason())
	}
	if ve.ValidationSuggestion() == "" {
		t.Error("expected non-empty suggestion")
	}
}
