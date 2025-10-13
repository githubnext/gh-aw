package cli

import (
	"testing"
)

func TestGenerateOutputSchema(t *testing.T) {
	t.Run("generates schema for simple struct", func(t *testing.T) {
		type SimpleOutput struct {
			Name  string `json:"name" jsonschema:"Name of the item"`
			Count int    `json:"count" jsonschema:"Number of items"`
		}

		schema, err := GenerateOutputSchema[SimpleOutput]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that schema has type object
		if schema.Type != "object" {
			t.Errorf("Expected schema type to be 'object', got '%s'", schema.Type)
		}

		// Check that properties are defined
		if schema.Properties == nil {
			t.Fatal("Expected schema properties to be defined")
		}

		// Check that name property exists
		if _, ok := schema.Properties["name"]; !ok {
			t.Error("Expected 'name' property to be defined")
		}

		// Check that count property exists
		if _, ok := schema.Properties["count"]; !ok {
			t.Error("Expected 'count' property to be defined")
		}
	})

	t.Run("generates schema for struct with optional fields", func(t *testing.T) {
		type OutputWithOptional struct {
			Required string  `json:"required" jsonschema:"Required field"`
			Optional *string `json:"optional,omitempty" jsonschema:"Optional field"`
		}

		schema, err := GenerateOutputSchema[OutputWithOptional]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that properties are defined
		if schema.Properties == nil {
			t.Fatal("Expected schema properties to be defined")
		}

		// Check that both fields exist
		if _, ok := schema.Properties["required"]; !ok {
			t.Error("Expected 'required' property to be defined")
		}
		if _, ok := schema.Properties["optional"]; !ok {
			t.Error("Expected 'optional' property to be defined")
		}
	})

	t.Run("generates schema for nested struct", func(t *testing.T) {
		type NestedData struct {
			Value int `json:"value" jsonschema:"Nested value"`
		}

		type OutputWithNested struct {
			Name   string     `json:"name" jsonschema:"Name"`
			Nested NestedData `json:"nested" jsonschema:"Nested data"`
		}

		schema, err := GenerateOutputSchema[OutputWithNested]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that nested property exists
		nestedProp, ok := schema.Properties["nested"]
		if !ok {
			t.Fatal("Expected 'nested' property to be defined")
		}

		// Check that nested property has object type
		if nestedProp.Type != "object" {
			t.Errorf("Expected nested type to be 'object', got '%s'", nestedProp.Type)
		}

		// Check that nested properties are defined
		if nestedProp.Properties == nil {
			t.Fatal("Expected nested properties to be defined")
		}

		if _, ok := nestedProp.Properties["value"]; !ok {
			t.Error("Expected nested 'value' property to be defined")
		}
	})

	t.Run("generates schema for slice field", func(t *testing.T) {
		type OutputWithSlice struct {
			Items []string `json:"items" jsonschema:"List of items"`
		}

		schema, err := GenerateOutputSchema[OutputWithSlice]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that items property exists
		itemsProp, ok := schema.Properties["items"]
		if !ok {
			t.Fatal("Expected 'items' property to be defined")
		}

		// Check that items is an array type
		if itemsProp.Type != "array" {
			t.Errorf("Expected items type to be 'array', got '%s'", itemsProp.Type)
		}

		// Check that items has an items schema
		if itemsProp.Items == nil {
			t.Fatal("Expected items to have an items schema")
		}

		// Check that the items schema is for strings
		if itemsProp.Items.Type != "string" {
			t.Errorf("Expected items schema type to be 'string', got '%s'", itemsProp.Items.Type)
		}
	})

	t.Run("generates schema for WorkflowStatus", func(t *testing.T) {
		schema, err := GenerateOutputSchema[WorkflowStatus]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed for WorkflowStatus: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that all expected properties exist
		expectedProps := []string{"workflow", "engine_id", "compiled", "status", "time_remaining"}
		for _, prop := range expectedProps {
			if _, ok := schema.Properties[prop]; !ok {
				t.Errorf("Expected '%s' property to be defined", prop)
			}
		}
	})

	t.Run("generates schema for LogsData", func(t *testing.T) {
		schema, err := GenerateOutputSchema[LogsData]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed for LogsData: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that expected top-level properties exist
		expectedProps := []string{"summary", "runs", "logs_location"}
		for _, prop := range expectedProps {
			if _, ok := schema.Properties[prop]; !ok {
				t.Errorf("Expected '%s' property to be defined", prop)
			}
		}
	})

	t.Run("generates schema for AuditData", func(t *testing.T) {
		schema, err := GenerateOutputSchema[AuditData]()
		if err != nil {
			t.Fatalf("GenerateOutputSchema failed for AuditData: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected schema to be non-nil")
		}

		// Check that expected top-level properties exist
		expectedProps := []string{"overview", "metrics", "downloaded_files"}
		for _, prop := range expectedProps {
			if _, ok := schema.Properties[prop]; !ok {
				t.Errorf("Expected '%s' property to be defined", prop)
			}
		}
	})
}
