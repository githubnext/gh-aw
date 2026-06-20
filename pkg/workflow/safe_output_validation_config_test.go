//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"
)

func TestGetValidationConfigJSON(t *testing.T) {
	// Test with nil (all types)
	jsonStr, err := GetValidationConfigJSON(nil, nil)
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
	jsonStr, err := GetValidationConfigJSON(enabledTypes, nil)
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
	jsonStr, err := GetValidationConfigJSON([]string{}, nil)
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

func TestGetValidationConfigJSONWithMentions(t *testing.T) {
	mentions := map[string]any{
		"enabled":              true,
		"allowedCollaborators": false,
		"allowContext":         false,
		"allowed":              []string{"copilot", "github-actions"},
		"max":                  5,
	}

	jsonStr, err := GetValidationConfigJSON([]string{"add_comment"}, mentions)
	if err != nil {
		t.Fatalf("GetValidationConfigJSON() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse validation config JSON: %v", err)
	}

	if _, ok := parsed["add_comment"]; !ok {
		t.Error("Expected add_comment type entry to be present")
	}

	raw, ok := parsed["mentions"]
	if !ok {
		t.Fatal("Expected top-level mentions key to be present in validation JSON")
	}
	mentionsParsed, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("Expected mentions to be an object, got %T", raw)
	}

	allowed, ok := mentionsParsed["allowed"].([]any)
	if !ok {
		t.Fatalf("Expected mentions.allowed to be an array, got %T", mentionsParsed["allowed"])
	}
	if len(allowed) != 2 || allowed[0] != "copilot" || allowed[1] != "github-actions" {
		t.Errorf("Unexpected mentions.allowed contents: %v", allowed)
	}
	if enabled, _ := mentionsParsed["enabled"].(bool); !enabled {
		t.Errorf("Expected mentions.enabled to be true, got %v", mentionsParsed["enabled"])
	}
	if allowTeam, _ := mentionsParsed["allowedCollaborators"].(bool); allowTeam {
		t.Errorf("Expected mentions.allowedCollaborators to be false, got %v", mentionsParsed["allowedCollaborators"])
	}

	// A second call without mentions must not include the key (cache safety).
	plainJSON, err := GetValidationConfigJSON([]string{"add_comment"}, nil)
	if err != nil {
		t.Fatalf("GetValidationConfigJSON() error = %v", err)
	}
	var plainParsed map[string]any
	if err := json.Unmarshal([]byte(plainJSON), &plainParsed); err != nil {
		t.Fatalf("Failed to parse plain validation config JSON: %v", err)
	}
	if _, ok := plainParsed["mentions"]; ok {
		t.Error("Did not expect mentions key in JSON produced without mentions argument")
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
		"requiresOneOf:status,title,body":                true,
		"requiresOneOf:title,body":                       true,
		"requiresOneOf:title,body,update_branch":         true,
		"requiresOneOf:title,body,labels":                true,
		"requiresOneOf:issue_number,pull_number":         true,
		"requiresOneOf:milestone_number,milestone_title": true,
		"requiresOneOf:field_name,field_node_id":         true,
		"requiresOneOf:reviewers,team_reviewers":         true,
		"startLineLessOrEqualLine":                       true,
		"parentAndSubDifferent":                          true,
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

func TestCreateDiscussionBodyMinLength(t *testing.T) {
	config, ok := ValidationConfig["create_discussion"]
	if !ok {
		t.Fatal("create_discussion not found in ValidationConfig")
	}

	bodyField, ok := config.Fields["body"]
	if !ok {
		t.Fatal("body field not found in create_discussion validation config")
	}

	if bodyField.MinLength != MinDiscussionBodyLength {
		t.Errorf("create_discussion body MinLength = %d, want %d", bodyField.MinLength, MinDiscussionBodyLength)
	}
}

func TestCreateIssueBodyMinLength(t *testing.T) {
	config, ok := ValidationConfig["create_issue"]
	if !ok {
		t.Fatal("create_issue not found in ValidationConfig")
	}

	bodyField, ok := config.Fields["body"]
	if !ok {
		t.Fatal("body field not found in create_issue validation config")
	}

	if bodyField.MinLength != MinIssueBodyLength {
		t.Errorf("create_issue body MinLength = %d, want %d", bodyField.MinLength, MinIssueBodyLength)
	}
}

func TestFieldValidationMinLengthMarshaling(t *testing.T) {
	field := FieldValidation{
		Required:  true,
		Type:      "string",
		MaxLength: 65000,
		MinLength: 64,
		Sanitize:  true,
	}

	data, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("Failed to marshal FieldValidation with MinLength: %v", err)
	}

	var parsed FieldValidation
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal FieldValidation with MinLength: %v", err)
	}

	if parsed.MinLength != field.MinLength {
		t.Errorf("MinLength mismatch: got %v, want %v", parsed.MinLength, field.MinLength)
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

func TestAssignMilestoneValidationConfig(t *testing.T) {
	config, ok := ValidationConfig["assign_milestone"]
	if !ok {
		t.Fatal("assign_milestone not found in ValidationConfig")
	}

	if config.CustomValidation != "requiresOneOf:milestone_number,milestone_title" {
		t.Errorf("assign_milestone customValidation = %q, want %q", config.CustomValidation, "requiresOneOf:milestone_number,milestone_title")
	}

	if _, ok := config.Fields["milestone_number"]; !ok {
		t.Error("assign_milestone Fields is missing the 'milestone_number' field")
	}
	if _, ok := config.Fields["milestone_title"]; !ok {
		t.Error("assign_milestone Fields is missing the 'milestone_title' field")
	}
}

func TestAssignMilestoneValidationConfigJSON(t *testing.T) {
	jsonStr, err := GetValidationConfigJSON([]string{"assign_milestone"}, nil)
	if err != nil {
		t.Fatalf("GetValidationConfigJSON() error = %v", err)
	}

	var parsed map[string]TypeValidationConfig
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse validation config JSON: %v", err)
	}

	cfg, ok := parsed["assign_milestone"]
	if !ok {
		t.Fatal("assign_milestone not found in serialized validation config")
	}
	if cfg.CustomValidation != "requiresOneOf:milestone_number,milestone_title" {
		t.Errorf("assign_milestone customValidation in JSON = %q, want %q", cfg.CustomValidation, "requiresOneOf:milestone_number,milestone_title")
	}
}

func TestBuildValidationConfigCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"nil input", nil, ""},
		{"empty input", []string{}, ""},
		{"single type", []string{"comment"}, "comment"},
		{"already sorted", []string{"comment", "issue"}, "comment,issue"},
		{"reverse order produces same key", []string{"issue", "comment"}, "comment,issue"},
		{"multiple types sorted", []string{"z_type", "a_type", "m_type"}, "a_type,m_type,z_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildValidationConfigCacheKey(tc.input)
			if got != tc.expected {
				t.Errorf("buildValidationConfigCacheKey(%v) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
