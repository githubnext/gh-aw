package console

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestFormatError(t *testing.T) {
	tests := []struct {
		name     string
		err      CompilerError
		expected []string // Substrings that should be present in output
	}{
		{
			name: "basic error with position",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "test.md",
					Line:   5,
					Column: 10,
				},
				Type:    "error",
				Message: "invalid syntax",
			},
			expected: []string{
				"test.md:5:10:",
				"error:",
				"invalid syntax",
			},
		},
		{
			name: "warning with hint",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "workflow.md",
					Line:   2,
					Column: 1,
				},
				Type:    "warning",
				Message: "deprecated field",
				Hint:    "use 'new_field' instead",
			},
			expected: []string{
				"workflow.md:2:1:",
				"warning:",
				"deprecated field",
				// Hints are no longer displayed as per requirements
			},
		},
		{
			name: "error with context",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "test.md",
					Line:   3,
					Column: 5,
				},
				Type:    "error",
				Message: "missing colon",
				Context: []string{
					"tools:",
					"  github",
					"    allowed: [list_issues]",
				},
			},
			expected: []string{
				"test.md:3:5:",
				"error:",
				"missing colon",
				"2 |",
				"3 |",
				"4 |",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatError(tt.err)

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFormatErrorWithSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		suggestions []string
		expected    []string
	}{
		{
			name:    "error with suggestions",
			message: "workflow 'test' not found",
			suggestions: []string{
				"Run 'gh aw status' to see all available workflows",
				"Create a new workflow with 'gh aw new test'",
				"Check for typos in the workflow name",
			},
			expected: []string{
				"✗",
				"workflow 'test' not found",
				"Suggestions:",
				"• Run 'gh aw status' to see all available workflows",
				"• Create a new workflow with 'gh aw new test'",
				"• Check for typos in the workflow name",
			},
		},
		{
			name:        "error without suggestions",
			message:     "workflow 'test' not found",
			suggestions: []string{},
			expected: []string{
				"✗",
				"workflow 'test' not found",
			},
		},
		{
			name:    "error with single suggestion",
			message: "file not found",
			suggestions: []string{
				"Check the file path",
			},
			expected: []string{
				"✗",
				"file not found",
				"Suggestions:",
				"• Check the file path",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatErrorWithSuggestions(tt.message, tt.suggestions)

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}

			// Verify no suggestions section when empty
			if len(tt.suggestions) == 0 && strings.Contains(output, "Suggestions:") {
				t.Errorf("Expected no suggestions section for empty suggestions, got:\n%s", output)
			}
		})
	}
}

func TestFormatSuccessMessage(t *testing.T) {
	output := FormatSuccessMessage("compilation completed")
	if !strings.Contains(output, "compilation completed") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected output to contain checkmark, got: %s", output)
	}
}

func TestFormatInfoMessage(t *testing.T) {
	output := FormatInfoMessage("processing file")
	if !strings.Contains(output, "processing file") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "ℹ") {
		t.Errorf("Expected output to contain info icon, got: %s", output)
	}
}

func TestFormatWarningMessage(t *testing.T) {
	output := FormatWarningMessage("deprecated syntax")
	if !strings.Contains(output, "deprecated syntax") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "⚠") {
		t.Errorf("Expected output to contain warning icon, got: %s", output)
	}
}

func TestRenderTable(t *testing.T) {
	tests := []struct {
		name     string
		config   TableConfig
		expected []string // Substrings that should be present in output
	}{
		{
			name: "simple table",
			config: TableConfig{
				Headers: []string{"ID", "Name", "Status"},
				Rows: [][]string{
					{"1", "Test", "Active"},
					{"2", "Demo", "Inactive"},
				},
			},
			expected: []string{
				"ID",
				"Name",
				"Status",
				"Test",
				"Demo",
				"Active",
				"Inactive",
			},
		},
		{
			name: "table with title and total",
			config: TableConfig{
				Title:   "Workflow Results",
				Headers: []string{"Run", "Duration", "Cost"},
				Rows: [][]string{
					{"123", "5m", "$0.50"},
					{"456", "3m", "$0.30"},
				},
				ShowTotal: true,
				TotalRow:  []string{"TOTAL", "8m", "$0.80"},
			},
			expected: []string{
				"Workflow Results",
				"Run",
				"Duration",
				"Cost",
				"123",
				"456",
				"TOTAL",
				"8m",
				"$0.80",
			},
		},
		{
			name: "empty table",
			config: TableConfig{
				Headers: []string{},
				Rows:    [][]string{},
			},
			expected: []string{}, // Should return empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderTable(tt.config)

			if len(tt.expected) == 0 {
				if output != "" {
					t.Errorf("Expected empty output for empty table config, got: %s", output)
				}
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFormatLocationMessage(t *testing.T) {
	output := FormatLocationMessage("Downloaded to: /path/to/logs")
	if !strings.Contains(output, "Downloaded to: /path/to/logs") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "📁") {
		t.Errorf("Expected output to contain folder icon, got: %s", output)
	}
}

func TestToRelativePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedFunc func(string, string) bool // Compare function that takes result and expected pattern
	}{
		{
			name: "relative path unchanged",
			path: "test.md",
			expectedFunc: func(result, expected string) bool {
				return result == "test.md"
			},
		},
		{
			name: "nested relative path unchanged",
			path: "pkg/console/test.md",
			expectedFunc: func(result, expected string) bool {
				return result == "pkg/console/test.md"
			},
		},
		{
			name: "absolute path converted to relative",
			path: "/tmp/gh-aw/test.md",
			expectedFunc: func(result, expected string) bool {
				// Should be a relative path that doesn't start with /
				return !strings.HasPrefix(result, "/") && strings.HasSuffix(result, "test.md")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToRelativePath(tt.path)
			if !tt.expectedFunc(result, tt.path) {
				t.Errorf("ToRelativePath(%s) = %s, but validation failed", tt.path, result)
			}
		})
	}
}

func TestFormatErrorWithAbsolutePaths(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := testutil.TempDir(t, "test-*")
	tmpFile := filepath.Join(tmpDir, "test.md")

	err := CompilerError{
		Position: ErrorPosition{
			File:   tmpFile,
			Line:   5,
			Column: 10,
		},
		Type:    "error",
		Message: "invalid syntax",
	}

	output := FormatError(err)

	// The output should contain test.md and line:column information
	if !strings.Contains(output, "test.md:5:10:") {
		t.Errorf("Expected output to contain relative file path with line:column, got: %s", output)
	}

	// The output should not start with an absolute path (no leading /)
	lines := strings.Split(output, "\n")
	if strings.HasPrefix(lines[0], "/") {
		t.Errorf("Expected output to start with relative path, but found absolute path: %s", lines[0])
	}

	// Should contain error message
	if !strings.Contains(output, "invalid syntax") {
		t.Errorf("Expected output to contain error message, got: %s", output)
	}
}

func TestRenderTableAsJSON(t *testing.T) {
	tests := []struct {
		name    string
		config  TableConfig
		wantErr bool
	}{
		{
			name: "simple table",
			config: TableConfig{
				Headers: []string{"Name", "Status"},
				Rows: [][]string{
					{"workflow1", "active"},
					{"workflow2", "disabled"},
				},
			},
			wantErr: false,
		},
		{
			name: "table with spaces in headers",
			config: TableConfig{
				Headers: []string{"Workflow Name", "Agent Type", "Is Compiled"},
				Rows: [][]string{
					{"test", "copilot", "Yes"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty table",
			config: TableConfig{
				Headers: []string{},
				Rows:    [][]string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTableAsJSON(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTableAsJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Verify it's valid JSON
			if result == "" && len(tt.config.Headers) > 0 {
				t.Error("RenderTableAsJSON() returned empty string for non-empty config")
			}
			// For empty config, should return "[]"
			if len(tt.config.Headers) == 0 && result != "[]" {
				t.Errorf("RenderTableAsJSON() = %v, want []", result)
			}
		})
	}
}

func TestClearScreen(t *testing.T) {
	// ClearScreen should not panic when called
	// It only clears if stdout is a TTY, so we can't easily test the output
	// but we can verify it doesn't panic
	t.Run("clear screen does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ClearScreen() panicked: %v", r)
			}
		}()
		ClearScreen()
	})
}

func TestRenderList(t *testing.T) {
	tests := []struct {
		name       string
		items      []string
		enumerator string
		expected   []string // Substrings that should be present in output
	}{
		{
			name:       "bullet list",
			items:      []string{"Item 1", "Item 2", "Item 3"},
			enumerator: "bullet",
			expected:   []string{"Item 1", "Item 2", "Item 3"},
		},
		{
			name:       "dash list",
			items:      []string{"First", "Second", "Third"},
			enumerator: "dash",
			expected:   []string{"First", "Second", "Third"},
		},
		{
			name:       "arabic list",
			items:      []string{"Alpha", "Beta", "Gamma"},
			enumerator: "arabic",
			expected:   []string{"Alpha", "Beta", "Gamma"},
		},
		{
			name:       "empty list",
			items:      []string{},
			enumerator: "bullet",
			expected:   []string{},
		},
		{
			name:       "single item",
			items:      []string{"Only one"},
			enumerator: "bullet",
			expected:   []string{"Only one"},
		},
		{
			name:       "default to bullet when invalid enumerator",
			items:      []string{"Test"},
			enumerator: "invalid",
			expected:   []string{"Test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderList(tt.items, tt.enumerator)

			// Empty list should return empty string
			if len(tt.expected) == 0 {
				if output != "" {
					t.Errorf("Expected empty output for empty list, got: %s", output)
				}
				return
			}

			// Check all expected strings are present
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestRenderNestedList(t *testing.T) {
	tests := []struct {
		name     string
		sections map[string][]string
		expected []string // Substrings that should be present in output
	}{
		{
			name: "single section with items",
			sections: map[string][]string{
				"Fruits": {"Apple", "Banana", "Orange"},
			},
			expected: []string{"Fruits", "Apple", "Banana", "Orange"},
		},
		{
			name: "multiple sections",
			sections: map[string][]string{
				"Fruits":     {"Apple", "Banana"},
				"Vegetables": {"Carrot", "Broccoli"},
			},
			expected: []string{"Fruits", "Apple", "Banana", "Vegetables", "Carrot", "Broccoli"},
		},
		{
			name: "section with no items",
			sections: map[string][]string{
				"Empty Section": {},
			},
			expected: []string{"Empty Section"},
		},
		{
			name:     "empty sections map",
			sections: map[string][]string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderNestedList(tt.sections)

			// Empty sections should return empty string
			if len(tt.expected) == 0 {
				if output != "" {
					t.Errorf("Expected empty output for empty sections, got: %s", output)
				}
				return
			}

			// Check all expected strings are present
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}
		})
	}
}
