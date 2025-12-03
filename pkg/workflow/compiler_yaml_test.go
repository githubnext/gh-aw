package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestCompileWorkflowWithInvalidYAML(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "invalid-yaml-test")

	tests := []struct {
		name                string
		content             string
		expectedErrorLine   int
		expectedErrorColumn int
		expectedMessagePart string
		description         string
	}{
		{
			name: "unclosed_bracket_in_array",
			content: `---
on: push
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    allowed: [list_issues
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with unclosed bracket.`,
			expectedErrorLine:   10, // Error detected at 'engine: claude' line
			expectedErrorColumn: 1,
			expectedMessagePart: "',' or ']' must be specified",
			description:         "unclosed bracket in array should be detected",
		},
		{
			name: "invalid_mapping_context",
			content: `---
on: push
permissions:
  contents: read
  issues: write
  pull-requests: read
invalid: yaml: syntax
  more: bad
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with bad mapping.`,
			expectedErrorLine:   7,
			expectedErrorColumn: 10, // Updated to match new YAML library error reporting
			expectedMessagePart: "mapping value is not allowed in this context",
			description:         "invalid mapping context should be detected",
		},
		{
			name: "bad_indentation",
			content: `---
on: push
permissions:
contents: read
  issues: write
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with bad indentation.`,
			expectedErrorLine:   4, // Updated to match new YAML library error reporting
			expectedErrorColumn: 11,
			expectedMessagePart: "mapping value is not allowed in this context", // Updated error message
			description:         "bad indentation should be detected",
		},
		{
			name: "unclosed_quote",
			content: `---
on: push
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    allowed: ["list_issues]
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with unclosed quote.`,
			expectedErrorLine:   9,
			expectedErrorColumn: 15, // Updated to match new YAML library error reporting
			expectedMessagePart: "could not find end character of double-quoted text",
			description:         "unclosed quote should be detected",
		},
		{
			name: "duplicate_keys",
			content: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
permissions:
  issues: write
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with duplicate keys.`,
			expectedErrorLine:   7,
			expectedErrorColumn: 1,
			expectedMessagePart: "mapping key \"permissions\" already defined",
			description:         "duplicate keys should be detected",
		},
		{
			name: "invalid_boolean_value",
			content: `---
on: push
permissions:
  contents: read
  issues: yes_please
  pull-requests: read
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with non-boolean value for permissions.`,
			expectedErrorLine:   3,                                              // The permissions field is on line 3
			expectedErrorColumn: 13,                                             // After "permissions:"
			expectedMessagePart: "value must be one of 'read', 'write', 'none'", // Schema validation catches this
			description:         "invalid boolean values should trigger schema validation error",
		},
		{
			name: "missing_colon_in_mapping",
			content: `---
on: push
permissions
  contents: read
  issues: write
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with missing colon.`,
			expectedErrorLine:   3,
			expectedErrorColumn: 1,
			expectedMessagePart: "unexpected key name",
			description:         "missing colon in mapping should be detected",
		},
		{
			name: "invalid_array_syntax_missing_comma",
			content: `---
on: push
tools:
  github:
    allowed: ["list_issues" "create_issue"]
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with missing comma in array.`,
			expectedErrorLine:   5,
			expectedErrorColumn: 29, // Updated to match new YAML library error reporting
			expectedMessagePart: "',' or ']' must be specified",
			description:         "missing comma in array should be detected",
		},
		{
			name:                "mixed_tabs_and_spaces",
			content:             "---\non: push\npermissions:\n  contents: read\n\tissues: write\nengine: claude\n---\n\n# Test Workflow\n\nInvalid YAML with mixed tabs and spaces.",
			expectedErrorLine:   5,
			expectedErrorColumn: 1,
			expectedMessagePart: "found character '\t' that cannot start any token",
			description:         "mixed tabs and spaces should be detected",
		},
		{
			name: "invalid_number_format",
			content: `---
on: push
timeout-minutes: 05.5
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with invalid number format.`,
			expectedErrorLine:   3,                          // The timeout-minutes field is on line 3
			expectedErrorColumn: 17,                         // After "timeout-minutes: "
			expectedMessagePart: "got number, want integer", // Schema validation catches this
			description:         "invalid number format should trigger schema validation error",
		},
		{
			name: "invalid_nested_structure",
			content: `---
on: push
tools:
  github: {
    allowed: ["list_issues"]
  }
  claude: [
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with malformed nested structure.`,
			expectedErrorLine:   7,
			expectedErrorColumn: 11, // Updated to match new YAML library error reporting
			expectedMessagePart: "sequence end token ']' not found",
			description:         "invalid nested structure should be detected",
		},
		{
			name: "unclosed_flow_mapping",
			content: `---
on: push
permissions: {contents: read, issues: write
engine: claude
strict: false
---

# Test Workflow

Invalid YAML with unclosed flow mapping.`,
			expectedErrorLine:   4,
			expectedErrorColumn: 1,
			expectedMessagePart: "',' or '}' must be specified",
			description:         "unclosed flow mapping should be detected",
		},
		{
			name: "yaml_error_with_column_information_support",
			content: `---
on: push
message: "invalid escape sequence \x in middle"
engine: claude
strict: false
---

# Test Workflow

YAML error that demonstrates column position handling.`,
			expectedErrorLine:   3, // The message field is on line 3 of the frontmatter (line 4 of file)
			expectedErrorColumn: 1, // Schema validation error
			expectedMessagePart: "Unknown property: message",
			description:         "yaml error should be extracted with column information when available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, fmt.Sprintf("%s.md", tt.name))
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Create compiler
			compiler := NewCompiler(false, "", "test")

			// Attempt compilation - should fail with proper error formatting
			err := compiler.CompileWorkflow(testFile)
			if err == nil {
				t.Errorf("%s: expected compilation to fail due to invalid YAML", tt.description)
				return
			}

			errorStr := err.Error()

			// Verify error contains file:line:column: format
			// The error should contain the filename (relative or absolute) with :line:column:
			expectedPattern := fmt.Sprintf("%s.md:%d:%d:", tt.name, tt.expectedErrorLine, tt.expectedErrorColumn)
			if !strings.Contains(errorStr, expectedPattern) {
				t.Errorf("%s: error should contain '%s', got: %s", tt.description, expectedPattern, errorStr)
			}

			// Verify error contains "error:" type indicator
			if !strings.Contains(errorStr, "error:") {
				t.Errorf("%s: error should contain 'error:' type indicator, got: %s", tt.description, errorStr)
			}

			// Verify error contains the expected YAML error message part
			if !strings.Contains(errorStr, tt.expectedMessagePart) {
				t.Errorf("%s: error should contain '%s', got: %s", tt.description, tt.expectedMessagePart, errorStr)
			}

			// For YAML parsing errors, verify error contains context lines
			if strings.Contains(errorStr, "frontmatter parsing failed") {
				// Verify error contains context lines (should show surrounding code)
				if !strings.Contains(errorStr, "|") {
					t.Errorf("%s: error should contain context lines with '|' markers, got: %s", tt.description, errorStr)
				}
			}
		})
	}
}

// TestCommentOutProcessedFieldsInOnSection tests the commentOutProcessedFieldsInOnSection function directly
