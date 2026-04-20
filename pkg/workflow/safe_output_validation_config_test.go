//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"
)

func TestGetValidationConfigJSON(t *testing.T) {
	// Test with nil (all types)
	jsonStr, err := GetValidationConfigJSON(nil)
	if err != nil {
		t.Fatalf("GetValidationConfigJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]TypeValidationConfig
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse validation config JSON: %v", err)
	}

	// Verify all expected types are present
	expectedTypes := []string{
		"create_issue",
		"create_agent_session",
		"add_comment",
		"create_pull_request",
		"add_labels",
		"add_reviewer",
		"assign_milestone",
		"assign_to_agent",
		"assign_to_user",
		"update_issue",
		"update_pull_request",
		"push_to_pull_request_branch",
		"create_pull_request_review_comment",
		"submit_pull_request_review",
		"create_discussion",
		"close_discussion",
		"close_issue",
		"close_pull_request",
		"missing_tool",
		"update_release",
		"upload_asset",
		"noop",
		"create_code_scanning_alert",
		"link_sub_issue",
		"update_discussion",
		"remove_labels",
		"unassign_from_user",
		"hide_comment",
		"missing_data",
		"autofix_code_scanning_alert",
		"mark_pull_request_as_ready_for_review",
		"report_incomplete",
	}

	for _, typeName := range expectedTypes {
		if _, ok := parsed[typeName]; !ok {
			t.Errorf("Expected type %q not found in validation config", typeName)
		}
	}

	// Verify JSON is indented (contains newlines)
	if !containsNewline(jsonStr) {
		t.Error("Expected indented JSON output with newlines")
	}
}

func TestGetValidationConfigJSONFiltered(t *testing.T) {
	// Test with filtered types
	enabledTypes := []string{"create_issue", "add_comment"}
	jsonStr, err := GetValidationConfigJSON(enabledTypes)
	if err != nil {
		t.Fatalf("GetValidationConfigJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]TypeValidationConfig
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse validation config JSON: %v", err)
	}

	// Verify only enabled types are present
	if len(parsed) != 2 {
		t.Errorf("Expected 2 types, got %d", len(parsed))
	}

	if _, ok := parsed["create_issue"]; !ok {
		t.Error("Expected create_issue to be present")
	}
	if _, ok := parsed["add_comment"]; !ok {
		t.Error("Expected add_comment to be present")
	}

	// Verify other types are NOT present
	if _, ok := parsed["create_discussion"]; ok {
		t.Error("Did not expect create_discussion to be present")
	}
}

func TestGetValidationConfigJSONEmpty(t *testing.T) {
	// Test with empty slice (should return all types, same as nil)
	jsonStr, err := GetValidationConfigJSON([]string{})
	if err != nil {
		t.Fatalf("GetValidationConfigJSON() error = %v", err)
	}

	var parsed map[string]TypeValidationConfig
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse validation config JSON: %v", err)
	}

	// Empty slice should return all types
	if len(parsed) != len(ValidationConfig) {
		t.Errorf("Expected %d types with empty slice, got %d", len(ValidationConfig), len(parsed))
	}
}

func containsNewline(s string) bool {
	for _, r := range s {
		if r == '\n' {
			return true
		}
	}
	return false
}

func TestFieldValidationMarshaling(t *testing.T) {
	// Test that FieldValidation marshals correctly with omitempty
	field := FieldValidation{
		Required:  true,
		Type:      "string",
		MaxLength: 128,
		Sanitize:  true,
	}

	data, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("Failed to marshal FieldValidation: %v", err)
	}

	// Verify omitempty works - should not include false/zero values
	jsonStr := string(data)
	if jsonStr == "" {
		t.Error("Empty JSON output")
	}

	// Parse it back
	var parsed FieldValidation
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal FieldValidation: %v", err)
	}

	if parsed.Required != field.Required {
		t.Errorf("Required mismatch: got %v, want %v", parsed.Required, field.Required)
	}
	if parsed.Type != field.Type {
		t.Errorf("Type mismatch: got %v, want %v", parsed.Type, field.Type)
	}
	if parsed.MaxLength != field.MaxLength {
		t.Errorf("MaxLength mismatch: got %v, want %v", parsed.MaxLength, field.MaxLength)
	}
}

func TestUpdateDiscussionValidationConfig(t *testing.T) {
	// Verify update_discussion accepts label-only updates (regression test for
	// https://github.com/github/gh-aw/issues/24979 where label-only updates were
	// rejected with "requires at least one of: 'title', 'body' fields").
	config, ok := ValidationConfig["update_discussion"]
	if !ok {
		t.Fatal("update_discussion not found in ValidationConfig")
	}

	// customValidation must include labels so label-only messages pass
	if config.CustomValidation != "requiresOneOf:title,body,labels" {
		t.Errorf("update_discussion customValidation = %q, want %q", config.CustomValidation, "requiresOneOf:title,body,labels")
	}

	// labels field must be defined so label values are validated
	if _, ok := config.Fields["labels"]; !ok {
		t.Error("update_discussion Fields is missing the 'labels' field")
	}
}

func TestUpdatePullRequestValidationConfig(t *testing.T) {
	config, ok := ValidationConfig["update_pull_request"]
	if !ok {
		t.Fatal("update_pull_request not found in ValidationConfig")
	}

	if config.CustomValidation != "requiresOneOf:title,body,update_branch" {
		t.Errorf("update_pull_request customValidation = %q, want %q", config.CustomValidation, "requiresOneOf:title,body,update_branch")
	}

	if _, ok := config.Fields["update_branch"]; !ok {
		t.Error("update_pull_request Fields is missing the 'update_branch' field")
	}
}

func TestValidationConfigConsistency(t *testing.T) {
	// Verify that all types with customValidation have valid validation rules
	validCustomValidations := map[string]bool{
		"requiresOneOf:status,title,body":        true,
		"requiresOneOf:title,body":               true,
		"requiresOneOf:title,body,update_branch": true,
		"requiresOneOf:title,body,labels":        true,
		"requiresOneOf:issue_number,pull_number": true,
		"requiresOneOf:reviewers,team_reviewers": true,
		"startLineLessOrEqualLine":               true,
		"parentAndSubDifferent":                  true,
	}

	for typeName, config := range ValidationConfig {
		if config.CustomValidation != "" {
			if !validCustomValidations[config.CustomValidation] {
				t.Errorf("Type %q has unknown customValidation: %q", typeName, config.CustomValidation)
			}
		}

		// Verify all types have at least one field
		if len(config.Fields) == 0 {
			t.Errorf("Type %q has no fields defined", typeName)
		}

		// Verify defaultMax is positive
		if config.DefaultMax <= 0 {
			t.Errorf("Type %q has invalid defaultMax: %d", typeName, config.DefaultMax)
		}
	}
}

func TestCreatePullRequestBaseValidationMaxLength(t *testing.T) {
	config, ok := ValidationConfig["create_pull_request"]
	if !ok {
		t.Fatal("create_pull_request not found in ValidationConfig")
	}

	baseField, ok := config.Fields["base"]
	if !ok {
		t.Fatal("base field not found in create_pull_request validation config")
	}

	if baseField.MaxLength != 128 {
		t.Errorf("base field MaxLength = %d, want 128", baseField.MaxLength)
	}
}
