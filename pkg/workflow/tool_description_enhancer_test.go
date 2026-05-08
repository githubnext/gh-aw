//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestEnhanceToolDescriptionCreateIssueAllowedFieldsWildcard(t *testing.T) {
	description := enhanceToolDescription("create_issue", "Create an issue.", &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			AllowedFields: []string{"*"},
		},
	})

	if !strings.Contains(description, "Any issue field is allowed.") {
		t.Fatalf("expected wildcard message in description, got: %s", description)
	}
	if strings.Contains(description, "Only these issue fields are allowed") {
		t.Fatalf("did not expect restrictive fields message for wildcard, got: %s", description)
	}
}

func TestEnhanceToolDescriptionCreateIssueAllowedFieldsList(t *testing.T) {
	description := enhanceToolDescription("create_issue", "Create an issue.", &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			AllowedFields: []string{"Priority", "Iteration"},
		},
	})

	if !strings.Contains(description, "Only these issue fields are allowed: [\"Priority\" \"Iteration\"].") {
		t.Fatalf("expected restrictive fields message in description, got: %s", description)
	}
}
