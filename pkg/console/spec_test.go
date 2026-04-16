//go:build !integration

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_PublicAPI_FormatFileSize validates the documented byte formatting
// behavior of FormatFileSize as described in the package README.md.
//
// Specification: "Formats a byte count as a human-readable string with
// appropriate unit suffix."
func TestSpec_PublicAPI_FormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{
			name:     "zero bytes documented as '0 B'",
			size:     0,
			expected: "0 B",
		},
		{
			name:     "1500 bytes documented as '1.5 KB'",
			size:     1500,
			expected: "1.5 KB",
		},
		{
			name:     "2.1 million bytes documented as '2.0 MB'",
			size:     2_100_000,
			expected: "2.0 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileSize(tt.size)
			assert.Equal(t, tt.expected, result,
				"FormatFileSize(%d) should match documented output", tt.size)
		})
	}
}

// TestSpec_Types_CompilerError validates that CompilerError has the documented
// fields and structure as described in the package README.md.
//
// Specification:
//
//	type CompilerError struct {
//	    Position ErrorPosition // Source file position
//	    Type     string        // "error", "warning", "info"
//	    Message  string
//	    Context  []string      // Source lines shown around the error
//	    Hint     string        // Optional actionable fix suggestion
//	}
func TestSpec_Types_CompilerError(t *testing.T) {
	err := CompilerError{
		Position: ErrorPosition{File: "workflow.md", Line: 12, Column: 5},
		Type:     "error",
		Message:  "unknown engine: 'myengine'",
		Context:  []string{"engine: myengine"},
		Hint:     "Valid engines are: copilot, claude, codex, gemini",
	}

	assert.Equal(t, "workflow.md", err.Position.File, "ErrorPosition.File should be accessible")
	assert.Equal(t, 12, err.Position.Line, "ErrorPosition.Line should be accessible")
	assert.Equal(t, 5, err.Position.Column, "ErrorPosition.Column should be accessible")
	assert.Equal(t, "error", err.Type, "CompilerError.Type should be accessible")
	assert.Equal(t, "unknown engine: 'myengine'", err.Message, "CompilerError.Message should be accessible")
	require.Len(t, err.Context, 1, "CompilerError.Context should hold context lines")
	assert.Equal(t, "engine: myengine", err.Context[0], "CompilerError.Context[0] should match")
	assert.Equal(t, "Valid engines are: copilot, claude, codex, gemini", err.Hint, "CompilerError.Hint should be accessible")
}

// TestSpec_Types_CompilerError_DocumentedTypes validates that CompilerError.Type
// accepts the documented values as described in the package README.md.
//
// Specification: Type string // "error", "warning", "info"
func TestSpec_Types_CompilerError_DocumentedTypes(t *testing.T) {
	documentedTypes := []string{"error", "warning", "info"}
	for _, errType := range documentedTypes {
		t.Run("type_"+errType, func(t *testing.T) {
			err := CompilerError{Type: errType, Message: "test"}
			assert.Equal(t, errType, err.Type,
				"CompilerError.Type should accept documented value %q", errType)
		})
	}
}

// TestSpec_Types_TableConfig validates the documented TableConfig struct fields
// as described in the package README.md.
//
// Specification:
//
//	type TableConfig struct {
//	    Headers   []string
//	    Rows      [][]string
//	    Title     string   // Optional table title
//	    ShowTotal bool     // Display a total row
//	    TotalRow  []string // Content for the total row
//	}
func TestSpec_Types_TableConfig(t *testing.T) {
	config := TableConfig{
		Headers:   []string{"Name", "Status", "Duration"},
		Rows:      [][]string{{"build", "success", "1m30s"}},
		Title:     "Job Results",
		ShowTotal: true,
		TotalRow:  []string{"Total", "", "1m30s"},
	}

	assert.Equal(t, []string{"Name", "Status", "Duration"}, config.Headers,
		"TableConfig.Headers should be settable")
	require.Len(t, config.Rows, 1, "TableConfig.Rows should hold row data")
	assert.Equal(t, "Job Results", config.Title, "TableConfig.Title should be settable")
	assert.True(t, config.ShowTotal, "TableConfig.ShowTotal should be settable")
	assert.Equal(t, []string{"Total", "", "1m30s"}, config.TotalRow,
		"TableConfig.TotalRow should be settable")
}

// TestSpec_Types_FormField validates the documented FormField struct and its
// Type values as described in the package README.md.
//
// Specification: Type string // "input", "password", "confirm", "select"
func TestSpec_Types_FormField(t *testing.T) {
	documentedTypes := []string{"input", "password", "confirm", "select"}

	for _, fieldType := range documentedTypes {
		t.Run("type_"+fieldType, func(t *testing.T) {
			field := FormField{
				Type:        fieldType,
				Title:       "Test Field",
				Description: "A test description",
				Placeholder: "placeholder",
			}
			assert.Equal(t, fieldType, field.Type,
				"FormField.Type should accept documented value %q", fieldType)
			assert.Equal(t, "Test Field", field.Title,
				"FormField.Title should be accessible")
			assert.Equal(t, "A test description", field.Description,
				"FormField.Description should be accessible")
			assert.Equal(t, "placeholder", field.Placeholder,
				"FormField.Placeholder should be accessible")
		})
	}
}

// TestSpec_Types_SelectOption validates the documented SelectOption struct
// as described in the package README.md.
//
// Specification:
//
//	type SelectOption struct {
//	    Label string
//	    Value string
//	}
func TestSpec_Types_SelectOption(t *testing.T) {
	opt := SelectOption{
		Label: "My Option",
		Value: "my-option",
	}
	assert.Equal(t, "My Option", opt.Label, "SelectOption.Label should be accessible")
	assert.Equal(t, "my-option", opt.Value, "SelectOption.Value should be accessible")
}

// TestSpec_Types_TreeNode validates the documented TreeNode struct
// as described in the package README.md.
//
// Specification:
//
//	type TreeNode struct {
//	    Value    string
//	    Children []TreeNode
//	}
func TestSpec_Types_TreeNode(t *testing.T) {
	node := TreeNode{
		Value: "root",
		Children: []TreeNode{
			{Value: "child1", Children: nil},
			{Value: "child2", Children: []TreeNode{{Value: "grandchild"}}},
		},
	}
	assert.Equal(t, "root", node.Value, "TreeNode.Value should be accessible")
	require.Len(t, node.Children, 2, "TreeNode.Children should support multiple children")
	assert.Equal(t, "child1", node.Children[0].Value, "Nested TreeNode.Value should be accessible")
	assert.Len(t, node.Children[1].Children, 1,
		"TreeNode.Children should support recursive nesting")
}

// TestSpec_PublicAPI_NewListItem validates the documented NewListItem constructor
// as described in the package README.md.
//
// Specification: "An item in an interactive list with title, description, and
// an internal value. Create with NewListItem(title, description, value string)."
func TestSpec_PublicAPI_NewListItem(t *testing.T) {
	item := NewListItem("My Title", "My Description", "my-value")
	assert.Equal(t, "My Title", item.title, "NewListItem should set title")
	assert.Equal(t, "My Description", item.description, "NewListItem should set description")
}

// TestSpec_DesignDecision_RenderStruct_SkipTag validates the documented
// console:"-" struct tag behavior of RenderStruct as described in the README.md.
//
// Specification: `"-"` — Always skips the field
func TestSpec_DesignDecision_RenderStruct_SkipTag(t *testing.T) {
	type TestData struct {
		Visible  string `console:"header:Visible"`
		Internal string `console:"-"`
	}

	data := TestData{
		Visible:  "shown",
		Internal: "hidden",
	}

	result := RenderStruct(data)
	assert.Contains(t, result, "shown",
		"fields without '-' tag should appear in rendered output")
	assert.NotContains(t, result, "hidden",
		"fields tagged with '-' must not appear in rendered output")
}

// TestSpec_DesignDecision_RenderStruct_OmitEmptyTag validates the documented
// omitempty struct tag behavior of RenderStruct as described in the README.md.
//
// Specification: `"omitempty"` — Skips the field if it has a zero value
func TestSpec_DesignDecision_RenderStruct_OmitEmptyTag(t *testing.T) {
	type TestData struct {
		Name     string `console:"header:Name"`
		Duration string `console:"header:Duration,omitempty"`
	}

	t.Run("zero value omitted", func(t *testing.T) {
		data := TestData{Name: "test", Duration: ""}
		result := RenderStruct(data)
		assert.Contains(t, result, "test",
			"non-omitempty field should appear in rendered output")
		assert.NotContains(t, result, "Duration",
			"omitempty field with zero value should not appear in rendered output")
	})

	t.Run("non-zero value included", func(t *testing.T) {
		data := TestData{Name: "test", Duration: "5m30s"}
		result := RenderStruct(data)
		assert.Contains(t, result, "5m30s",
			"omitempty field with non-zero value should appear in rendered output")
	})
}
