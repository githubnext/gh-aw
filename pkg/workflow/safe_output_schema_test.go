package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSafeOutputSchema(t *testing.T) {
	// Use the actual schema file from the repository
	schemaPath := filepath.Join("..", "..", "schemas", "agent-output.json")

	schema, err := LoadSafeOutputSchema(schemaPath)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	if len(schema.Definitions) == 0 {
		t.Error("Expected schema to have definitions")
	}

	// Verify some known types exist
	expectedTypes := []string{
		"CreateIssueOutput",
		"AddCommentOutput",
		"NoOpOutput",
		"CloseIssueOutput",
	}

	for _, typeName := range expectedTypes {
		if _, exists := schema.Definitions[typeName]; !exists {
			t.Errorf("Expected schema to contain definition for %s", typeName)
		}
	}
}

func TestGetSafeOutputTypes(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "agent-output.json")

	schema, err := LoadSafeOutputSchema(schemaPath)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	types := GetSafeOutputTypes(schema)

	if len(types) == 0 {
		t.Error("Expected to extract some safe output types")
	}

	// Verify we got the expected number of output types
	// Should be all *Output types except SafeOutput itself
	if len(types) < 20 {
		t.Errorf("Expected at least 20 safe output types, got %d", len(types))
	}

	// Verify that SafeOutput union type is not included
	for _, typeSchema := range types {
		if typeSchema.TypeName == "" {
			t.Error("Found type with empty TypeName")
		}
		// Check that type name was correctly extracted
		if typeSchema.Title == "Safe Output Item" {
			t.Error("SafeOutput union type should not be included in output types")
		}
	}

	// Verify known types have correct type names
	typeNamesFound := make(map[string]bool)
	for _, typeSchema := range types {
		typeNamesFound[typeSchema.TypeName] = true
	}

	expectedTypeNames := []string{
		"create_issue",
		"add_comment",
		"noop",
		"close_issue",
		"minimize_comment",
	}

	for _, expectedName := range expectedTypeNames {
		if !typeNamesFound[expectedName] {
			t.Errorf("Expected to find type name %s", expectedName)
		}
	}
}

func TestGetJavaScriptFilename(t *testing.T) {
	tests := []struct {
		typeName string
		expected string
	}{
		{"create_issue", "create_issue.cjs"},
		{"add_comment", "add_comment.cjs"},
		{"noop", "noop.cjs"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := GetJavaScriptFilename(tt.typeName)
			if result != tt.expected {
				t.Errorf("GetJavaScriptFilename(%q) = %q, want %q", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestShouldGenerateCustomAction(t *testing.T) {
	tests := []struct {
		typeName string
		expected bool
	}{
		// Types that should have custom actions
		{"noop", true},
		{"create_issue", true},
		{"add_comment", true},
		{"close_issue", true},
		{"minimize_comment", true},
		{"close_pull_request", true},
		{"close_discussion", true},
		{"add_labels", true},
		{"create_discussion", true},
		{"update_issue", true},
		{"update_pull_request", true},

		// Types that should not have custom actions (more complex ones)
		{"create_pull_request", false},
		{"push_to_pull_request_branch", false},
		{"create_code_scanning_alert", false},
		{"update_project", false},
		{"missing_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := ShouldGenerateCustomAction(tt.typeName)
			if result != tt.expected {
				t.Errorf("ShouldGenerateCustomAction(%q) = %v, want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestSchemaTypeExtraction(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "agent-output.json")

	schema, err := LoadSafeOutputSchema(schemaPath)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	types := GetSafeOutputTypes(schema)

	// Find create_issue type to verify its properties
	var createIssueType *SafeOutputTypeSchema
	for i := range types {
		if types[i].TypeName == "create_issue" {
			createIssueType = &types[i]
			break
		}
	}

	if createIssueType == nil {
		t.Fatal("Expected to find create_issue type")
	}

	// Verify title and description were extracted
	if createIssueType.Title == "" {
		t.Error("Expected create_issue to have a title")
	}

	if createIssueType.Description == "" {
		t.Error("Expected create_issue to have a description")
	}

	// Verify properties were extracted
	if len(createIssueType.Properties) == 0 {
		t.Error("Expected create_issue to have properties")
	}

	// Verify required fields were extracted
	if len(createIssueType.Required) == 0 {
		t.Error("Expected create_issue to have required fields")
	}

	// Check for specific properties
	expectedProps := []string{"type", "title", "body"}
	for _, propName := range expectedProps {
		if _, exists := createIssueType.Properties[propName]; !exists {
			t.Errorf("Expected create_issue to have property %s", propName)
		}
	}
}

func TestSchemaFileExists(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "agent-output.json")

	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Fatal("Schema file does not exist at expected location")
	}
}

func TestExtractTypeNameFromDefinition(t *testing.T) {
	// Test with a simple type definition
	def := TypeDefinition{
		Title:       "Create Issue Output",
		Description: "Output for creating a GitHub issue",
		Type:        "object",
		Properties: map[string]interface{}{
			"type": map[string]interface{}{
				"const": "create_issue",
			},
			"title": map[string]interface{}{
				"type": "string",
			},
		},
		Required: []string{"type", "title"},
	}

	typeName := extractTypeNameFromDefinition(def)
	if typeName != "create_issue" {
		t.Errorf("extractTypeNameFromDefinition() = %q, want %q", typeName, "create_issue")
	}
}

func TestExtractTypeNameFromDefinition_NoType(t *testing.T) {
	// Test with definition that has no type property
	def := TypeDefinition{
		Title:      "Test Output",
		Properties: map[string]interface{}{},
	}

	typeName := extractTypeNameFromDefinition(def)
	if typeName != "" {
		t.Errorf("extractTypeNameFromDefinition() = %q, want empty string", typeName)
	}
}

func TestParseProperties(t *testing.T) {
	props := map[string]interface{}{
		"title": map[string]interface{}{
			"type":        "string",
			"description": "Title of the issue",
			"minLength":   float64(1),
		},
		"labels": map[string]interface{}{
			"type":        "array",
			"description": "Optional labels",
			"items": map[string]interface{}{
				"type": "string",
			},
		},
		"status": map[string]interface{}{
			"type":        "string",
			"description": "Status of the issue",
			"enum":        []interface{}{"open", "closed"},
		},
	}

	result := parseProperties(props)

	if len(result) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(result))
	}

	// Check title property
	if titleProp, exists := result["title"]; exists {
		if titleProp.Description != "Title of the issue" {
			t.Errorf("Expected description 'Title of the issue', got %q", titleProp.Description)
		}
		if titleProp.MinLength == nil || *titleProp.MinLength != 1 {
			t.Error("Expected minLength to be 1")
		}
	} else {
		t.Error("Expected title property to exist")
	}

	// Check labels property
	if labelsProp, exists := result["labels"]; exists {
		if labelsProp.Description != "Optional labels" {
			t.Errorf("Expected description 'Optional labels', got %q", labelsProp.Description)
		}
		if labelsProp.Items == nil {
			t.Error("Expected items to be set for array type")
		}
	} else {
		t.Error("Expected labels property to exist")
	}

	// Check status property with enum
	if statusProp, exists := result["status"]; exists {
		if len(statusProp.Enum) != 2 {
			t.Errorf("Expected 2 enum values, got %d", len(statusProp.Enum))
		}
	} else {
		t.Error("Expected status property to exist")
	}
}
