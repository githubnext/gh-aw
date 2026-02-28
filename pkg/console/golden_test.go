//go:build !integration

package console

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

// TestGolden_TableRendering tests table rendering with different configurations
func TestGolden_TableRendering(t *testing.T) {
	tests := []struct {
		name   string
		config TableConfig
	}{
		{
			name: "simple_table",
			config: TableConfig{
				Headers: []string{"Name", "Status", "Duration"},
				Rows: [][]string{
					{"test-1", "success", "1.2s"},
					{"test-2", "failed", "0.5s"},
					{"test-3", "pending", "0.0s"},
				},
			},
		},
		{
			name: "table_with_title",
			config: TableConfig{
				Title:   "Workflow Results",
				Headers: []string{"ID", "Name", "Status"},
				Rows: [][]string{
					{"123", "workflow-1", "completed"},
					{"456", "workflow-2", "in_progress"},
				},
			},
		},
		{
			name: "table_with_total",
			config: TableConfig{
				Headers: []string{"Run", "Duration", "Cost"},
				Rows: [][]string{
					{"123", "5m", "$0.50"},
					{"456", "3m", "$0.30"},
				},
				ShowTotal: true,
				TotalRow:  []string{"TOTAL", "8m", "$0.80"},
			},
		},
		{
			name: "wide_table",
			config: TableConfig{
				Headers: []string{"ID", "Workflow", "Status", "Duration", "Engine", "Model", "Cost"},
				Rows: [][]string{
					{"12345", "test-workflow", "completed", "5m 30s", "copilot", "gpt-4", "$0.75"},
					{"12346", "build-project", "failed", "2m 15s", "claude", "opus", "$0.45"},
					{"12347", "deploy-staging", "in_progress", "8m 42s", "copilot", "gpt-4", "$1.20"},
				},
			},
		},
		{
			name: "empty_table",
			config: TableConfig{
				Headers: []string{},
				Rows:    [][]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Force non-TTY mode for consistent output
			origStdout := os.Stdout
			defer func() { os.Stdout = origStdout }()

			output := RenderTable(tt.config)
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// TestGolden_BoxRendering tests box rendering with various content
func TestGolden_BoxRendering(t *testing.T) {
	tests := []struct {
		name  string
		title string
		width int
	}{
		{
			name:  "narrow_box",
			title: "Test",
			width: 30,
		},
		{
			name:  "medium_box",
			title: "Trial Execution Plan",
			width: 60,
		},
		{
			name:  "wide_box",
			title: "GitHub Agentic Workflows Compilation Report",
			width: 100,
		},
		{
			name:  "box_with_emoji",
			title: "⚠️ WARNING: Critical Issue",
			width: 50,
		},
		{
			name:  "very_narrow_box",
			title: "X",
			width: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test RenderTitleBox which returns []string
			output := RenderTitleBox(tt.title, tt.width)
			var fullOutput strings.Builder
			for _, line := range output {
				fullOutput.WriteString(line + "\n")
			}
			golden.RequireEqual(t, []byte(fullOutput.String()))
		})
	}
}

// TestGolden_LayoutBoxRendering tests layout box rendering (returns string)

// TestGolden_ErrorFormatting tests error formatting with context
func TestGolden_ErrorFormatting(t *testing.T) {
	tests := []struct {
		name string
		err  CompilerError
	}{
		{
			name: "basic_error",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "test.md",
					Line:   5,
					Column: 10,
				},
				Type:    "error",
				Message: "invalid syntax: missing colon after key",
			},
		},
		{
			name: "warning_with_hint",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "workflow.md",
					Line:   2,
					Column: 1,
				},
				Type:    "warning",
				Message: "deprecated field 'mcp_servers' detected",
				Hint:    "use 'tools' field instead",
			},
		},
		{
			name: "error_with_context",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "test.md",
					Line:   3,
					Column: 5,
				},
				Type:    "error",
				Message: "missing colon in YAML mapping",
				Context: []string{
					"tools:",
					"  github",
					"    allowed: [list_issues]",
				},
			},
		},
		{
			name: "error_multiline_context",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "workflow.md",
					Line:   10,
					Column: 15,
				},
				Type:    "error",
				Message: "invalid MCP server configuration",
				Context: []string{
					"---",
					"engine: copilot",
					"tools:",
					"  github:",
					"    mode: remote",
					"    toolsets: [invalid_toolset]",
					"---",
				},
			},
		},
		{
			name: "info_message",
			err: CompilerError{
				Position: ErrorPosition{
					File:   "example.md",
					Line:   1,
					Column: 1,
				},
				Type:    "info",
				Message: "workflow compiled successfully",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatError(tt.err)
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// TestGolden_ErrorWithSuggestions tests error formatting with suggestions
func TestGolden_ErrorWithSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		suggestions []string
	}{
		{
			name:    "error_with_multiple_suggestions",
			message: "workflow 'test' not found",
			suggestions: []string{
				"Run 'gh aw status' to see all available workflows",
				"Create a new workflow with 'gh aw new test'",
				"Check for typos in the workflow name",
			},
		},
		{
			name:        "error_no_suggestions",
			message:     "workflow 'test' not found",
			suggestions: []string{},
		},
		{
			name:    "error_single_suggestion",
			message: "file not found: workflow.md",
			suggestions: []string{
				"Check the file path and try again",
			},
		},
		{
			name:    "compilation_error_with_suggestions",
			message: "invalid YAML syntax in frontmatter",
			suggestions: []string{
				"Ensure all keys have values",
				"Check for missing colons after keys",
				"Verify proper indentation (use spaces, not tabs)",
				"Run 'gh aw validate workflow.md' for detailed errors",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatErrorWithSuggestions(tt.message, tt.suggestions)
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// TestGolden_MessageFormatting tests various message formatting functions
func TestGolden_MessageFormatting(t *testing.T) {
	tests := []struct {
		name    string
		message string
		format  func(string) string
	}{
		{
			name:    "success_message",
			message: "Compilation completed successfully",
			format:  FormatSuccessMessage,
		},
		{
			name:    "info_message",
			message: "Processing workflow file: test.md",
			format:  FormatInfoMessage,
		},
		{
			name:    "warning_message",
			message: "Deprecated syntax detected",
			format:  FormatWarningMessage,
		},
		{
			name:    "error_message",
			message: "Failed to compile workflow",
			format:  FormatErrorMessage,
		},
		{
			name:    "command_message",
			message: "gh aw compile workflow.md",
			format:  FormatCommandMessage,
		},
		{
			name:    "progress_message",
			message: "Compiling workflow (step 2/5)...",
			format:  FormatProgressMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.format(tt.message)
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// TestGolden_LayoutComposition tests composing multiple layout elements

// TestGolden_LayoutEmphasisBox tests emphasis boxes with different colors

// TestGolden_InfoSection tests info section rendering
func TestGolden_InfoSection(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "single_line",
			content: "Workflow: test-workflow",
		},
		{
			name:    "multiple_lines",
			content: "Line 1\nLine 2\nLine 3",
		},
		{
			name:    "with_special_chars",
			content: "Path: /tmp/gh-aw/workflows\nStatus: ✓ Active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderInfoSection(tt.content)
			var fullOutput strings.Builder
			for _, line := range output {
				fullOutput.WriteString(line + "\n")
			}
			golden.RequireEqual(t, []byte(fullOutput.String()))
		})
	}
}
