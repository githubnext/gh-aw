//go:build !integration

package console

import (
	"strings"
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

// TestSpec_PublicAPI_FormatNumber validates the documented behavior of FormatNumber.
// Specification: "Formats a large integer as a compact human-readable string using SI suffixes (k, M, B)."
func TestSpec_PublicAPI_FormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		// From spec: FormatNumber(0) // "0"
		{name: "zero", input: 0, expected: "0"},
		// From spec: FormatNumber(999) // "999"
		{name: "below 1000", input: 999, expected: "999"},
		// From spec: FormatNumber(1500) // "1.50k"
		{name: "1500 as 1.50k", input: 1500, expected: "1.50k"},
		// From spec: FormatNumber(1_200_000) // "1.20M"
		{name: "1.2M as 1.20M", input: 1_200_000, expected: "1.20M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.input)
			assert.Equal(t, tt.expected, result,
				"FormatNumber(%d) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_ToRelativePath validates the documented behavior of ToRelativePath.
// Specification: "Converts an absolute path to a path relative to the current working directory.
// If the relative path would require traversing parent directories (..), the original absolute
// path is returned unchanged."
func TestSpec_PublicAPI_ToRelativePath(t *testing.T) {
	t.Run("absolute path outside cwd returns absolute path unchanged", func(t *testing.T) {
		// /etc/hosts is outside any likely cwd, so relative path would contain ..
		result := ToRelativePath("/etc/hosts")
		assert.Equal(t, "/etc/hosts", result,
			"path requiring .. traversal should return the original absolute path as documented")
	})

	t.Run("non-absolute path returned unchanged", func(t *testing.T) {
		result := ToRelativePath("relative/path.md")
		assert.Equal(t, "relative/path.md", result,
			"non-absolute path should be returned unchanged")
	})
}

// TestSpec_PublicAPI_FormatErrorWithSuggestions validates the documented behavior.
// Specification: "Formats an error message followed by a bulleted list of actionable suggestions.
// Returns an empty suggestions block when suggestions is nil or empty."
func TestSpec_PublicAPI_FormatErrorWithSuggestions(t *testing.T) {
	t.Run("with suggestions includes bulleted list", func(t *testing.T) {
		result := FormatErrorWithSuggestions(
			"Unknown engine 'myengine'",
			[]string{
				"Valid engines are: copilot, claude, codex",
				"Check your workflow frontmatter",
			},
		)
		assert.Contains(t, result, "Valid engines are: copilot, claude, codex",
			"formatted result should include each suggestion as documented")
		assert.Contains(t, result, "Check your workflow frontmatter",
			"formatted result should include each suggestion as documented")
	})

	t.Run("with nil suggestions returns non-empty (message only)", func(t *testing.T) {
		result := FormatErrorWithSuggestions("Some error", nil)
		assert.NotEmpty(t, result,
			"result should be non-empty even with nil suggestions")
	})

	t.Run("with empty suggestions returns non-empty (message only)", func(t *testing.T) {
		result := FormatErrorWithSuggestions("Some error", []string{})
		assert.NotEmpty(t, result,
			"result should be non-empty even with empty suggestions slice")
	})
}

// TestSpec_PublicAPI_IsAccessibleMode validates the documented behavior.
// Specification: "Returns true when the terminal is in accessibility mode based on environment variables:
// ACCESSIBLE is set, TERM is 'dumb', NO_COLOR is set."
func TestSpec_PublicAPI_IsAccessibleMode(t *testing.T) {
	t.Run("ACCESSIBLE env var triggers accessible mode", func(t *testing.T) {
		t.Setenv("ACCESSIBLE", "1")
		assert.True(t, IsAccessibleMode(),
			"IsAccessibleMode should return true when ACCESSIBLE env var is set")
	})

	t.Run("TERM=dumb triggers accessible mode", func(t *testing.T) {
		t.Setenv("ACCESSIBLE", "")
		t.Setenv("NO_COLOR", "")
		t.Setenv("TERM", "dumb")
		assert.True(t, IsAccessibleMode(),
			"IsAccessibleMode should return true when TERM=dumb as documented")
	})

	t.Run("NO_COLOR env var triggers accessible mode", func(t *testing.T) {
		t.Setenv("ACCESSIBLE", "")
		t.Setenv("TERM", "xterm")
		t.Setenv("NO_COLOR", "1")
		assert.True(t, IsAccessibleMode(),
			"IsAccessibleMode should return true when NO_COLOR env var is set")
	})
}

// TestSpec_PublicAPI_RenderTable validates the documented RenderTable function.
// Specification: "Renders a formatted table with optional title and total row."
func TestSpec_PublicAPI_RenderTable(t *testing.T) {
	t.Run("renders headers and rows", func(t *testing.T) {
		config := TableConfig{
			Headers: []string{"Name", "Status", "Duration"},
			Rows: [][]string{
				{"build", "success", "1m30s"},
				{"test", "failure", "45s"},
			},
		}
		result := RenderTable(config)
		assert.NotEmpty(t, result, "RenderTable should return non-empty output")
		assert.Contains(t, result, "build", "RenderTable output should include row data")
		assert.Contains(t, result, "success", "RenderTable output should include row data")
	})

	t.Run("renders optional title", func(t *testing.T) {
		config := TableConfig{
			Headers: []string{"Name", "Status"},
			Rows:    [][]string{{"build", "success"}},
			Title:   "Job Results",
		}
		result := RenderTable(config)
		assert.Contains(t, result, "Job Results",
			"RenderTable should include the optional title when set")
	})
}

// TestSpec_PublicAPI_RenderTitleBox validates the documented RenderTitleBox function.
// Specification: "Returns a rounded-border box containing title, padded to at least width characters."
func TestSpec_PublicAPI_RenderTitleBox(t *testing.T) {
	result := RenderTitleBox("Audit Report", 60)
	require.NotEmpty(t, result, "RenderTitleBox should return non-empty []string")
	var sb strings.Builder
	for _, line := range result {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	assert.Contains(t, sb.String(), "Audit Report",
		"RenderTitleBox output should contain the provided title")
}
