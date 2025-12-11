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

// ========================================
// convertGoPatternToJavaScript Tests
// ========================================

// TestConvertGoPatternToJavaScript tests the convertGoPatternToJavaScript method
func TestConvertGoPatternToJavaScript(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name      string
		goPattern string
		expected  string
	}{
		{
			name:      "case insensitive flag removed",
			goPattern: "(?i)error.*pattern",
			expected:  "error.*pattern",
		},
		{
			name:      "no flag to remove",
			goPattern: "error.*pattern",
			expected:  "error.*pattern",
		},
		{
			name:      "empty pattern",
			goPattern: "",
			expected:  "",
		},
		{
			name:      "flag at start only",
			goPattern: "(?i)",
			expected:  "",
		},
		{
			name:      "complex pattern with flag",
			goPattern: "(?i)^(ERROR|WARN|FATAL):.*$",
			expected:  "^(ERROR|WARN|FATAL):.*$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.convertGoPatternToJavaScript(tt.goPattern)
			if result != tt.expected {
				t.Errorf("convertGoPatternToJavaScript(%q) = %q, want %q",
					tt.goPattern, result, tt.expected)
			}
		})
	}
}

// ========================================
// convertErrorPatternsToJavaScript Tests
// ========================================

// TestConvertErrorPatternsToJavaScript tests the convertErrorPatternsToJavaScript method
func TestConvertErrorPatternsToJavaScript(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	patterns := []ErrorPattern{
		{
			Pattern:      "(?i)error",
			LevelGroup:   1,
			MessageGroup: 2,
			Description:  "Error pattern",
		},
		{
			Pattern:      "warning",
			LevelGroup:   0,
			MessageGroup: 1,
			Description:  "Warning pattern",
		},
	}

	result := compiler.convertErrorPatternsToJavaScript(patterns)

	if len(result) != len(patterns) {
		t.Errorf("Expected %d patterns, got %d", len(patterns), len(result))
	}

	// First pattern should have (?i) removed
	if result[0].Pattern != "error" {
		t.Errorf("Expected first pattern to be 'error', got %q", result[0].Pattern)
	}

	// Second pattern should remain unchanged
	if result[1].Pattern != "warning" {
		t.Errorf("Expected second pattern to be 'warning', got %q", result[1].Pattern)
	}

	// Check that other fields are preserved
	if result[0].LevelGroup != 1 {
		t.Errorf("Expected LevelGroup to be preserved")
	}
	if result[0].Description != "Error pattern" {
		t.Errorf("Expected Description to be preserved")
	}
}

// ========================================
// addCustomStepsAsIs Tests
// ========================================

// TestAddCustomStepsAsIsBasic tests the addCustomStepsAsIs method
func TestAddCustomStepsAsIsBasic(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		customSteps string
		expectedIn  []string
		expectedNot []string
	}{
		{
			name: "basic steps",
			customSteps: `steps:
  - name: Setup
    run: echo "setup"`,
			expectedIn: []string{"name: Setup", "run: echo"},
		},
		{
			name: "multiple steps",
			customSteps: `steps:
  - name: Step 1
    run: echo "1"
  - name: Step 2
    run: echo "2"`,
			expectedIn: []string{"name: Step 1", "name: Step 2"},
		},
		{
			name: "step with uses",
			customSteps: `steps:
  - name: Checkout
    uses: actions/checkout@v4`,
			expectedIn: []string{"name: Checkout", "uses: actions/checkout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			compiler.addCustomStepsAsIs(&builder, tt.customSteps)
			result := builder.String()

			for _, expected := range tt.expectedIn {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected %q in result:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.expectedNot {
				if strings.Contains(result, notExpected) {
					t.Errorf("Did not expect %q in result:\n%s", notExpected, result)
				}
			}
		})
	}
}

// ========================================
// Integration Tests for generateYAML
// ========================================

// TestGenerateYAMLBasicWorkflow tests generating YAML for a basic workflow
func TestGenerateYAMLBasicWorkflow(t *testing.T) {
	tmpDir := testutil.TempDir(t, "yaml-gen-test")

	frontmatter := `---
name: Test Workflow
on: push
permissions:
  contents: read
engine: copilot
strict: false
---

# Test Workflow

This is a test workflow.`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Check basic workflow structure
	expectedElements := []string{
		"name: \"Test Workflow\"",
		"on:",
		"push",
		"permissions:",
		"jobs:",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected %q in generated YAML", expected)
		}
	}
}

// TestGenerateYAMLWithDescription tests that description is added as comment
func TestGenerateYAMLWithDescription(t *testing.T) {
	tmpDir := testutil.TempDir(t, "yaml-desc-test")

	frontmatter := `---
name: Test Workflow
description: This workflow does important things
on: push
permissions:
  contents: read
engine: copilot
strict: false
---

# Test Workflow

Test content.`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Description should appear in comments
	if !strings.Contains(yamlStr, "# This workflow does important things") {
		t.Error("Expected description to be in comments")
	}
}

// TestGenerateYAMLAutoGeneratedDisclaimer tests that disclaimer is added
func TestGenerateYAMLAutoGeneratedDisclaimer(t *testing.T) {
	tmpDir := testutil.TempDir(t, "yaml-disclaimer-test")

	frontmatter := `---
name: Test Workflow
on: push
permissions:
  contents: read
engine: copilot
strict: false
---

# Test Workflow

Test content.`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Check for auto-generated disclaimer
	if !strings.Contains(yamlStr, "This file was automatically generated by gh-aw. DO NOT EDIT.") {
		t.Error("Expected auto-generated disclaimer")
	}
}

// TestGenerateYAMLWithEnvironment tests that environment is properly set
func TestGenerateYAMLWithEnvironment(t *testing.T) {
	tmpDir := testutil.TempDir(t, "yaml-env-test")

	frontmatter := `---
name: Test Workflow
on: push
permissions:
  contents: read
engine: copilot
strict: false
environment: production
---

# Test Workflow

Test content.`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Check for environment in output
	if !strings.Contains(yamlStr, "environment:") {
		t.Error("Expected environment in generated YAML")
	}
}

// TestGenerateYAMLWithConcurrency tests that concurrency is properly set
func TestGenerateYAMLWithConcurrency(t *testing.T) {
	tmpDir := testutil.TempDir(t, "yaml-concurrency-test")

	frontmatter := `---
name: Test Workflow
on: push
permissions:
  contents: read
engine: copilot
strict: false
concurrency:
  group: test-group
  cancel-in-progress: true
---

# Test Workflow

Test content.`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Check for concurrency in output
	if !strings.Contains(yamlStr, "concurrency:") {
		t.Error("Expected concurrency in generated YAML")
	}
}

// TestCompilerGeneratesMetadata tests that the compiler generates metadata in lock files
func TestCompilerGeneratesMetadata(t *testing.T) {
	tmpDir := testutil.TempDir(t, "metadata-test")

	workflowContent := `---
name: test-metadata-workflow
engine: copilot
on: workflow_dispatch
---

# Test Workflow

This is a test workflow for metadata generation.`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "v0.0.367-test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Check that metadata block exists
	if !strings.Contains(yamlStr, "# Metadata:") {
		t.Error("Expected metadata block in generated lock file")
	}

	// Check that all required metadata fields are present
	requiredFields := []string{
		"#   source_hash: sha256:",
		"#   compiled_at:",
		"#   gh_aw_version: v0.0.367-test",
		"#   dependencies_hash:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(yamlStr, field) {
			t.Errorf("Expected metadata field %q in lock file", field)
		}
	}

	// Verify that metadata can be extracted
	metadata, err := ExtractLockFileMetadata(lockFile)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if metadata.GhAwVersion != "v0.0.367-test" {
		t.Errorf("Expected gh_aw_version='v0.0.367-test', got %s", metadata.GhAwVersion)
	}

	if !strings.HasPrefix(metadata.SourceHash, "sha256:") {
		t.Errorf("Expected source_hash to start with 'sha256:', got %s", metadata.SourceHash)
	}

	if metadata.CompiledAt == "" {
		t.Error("Expected compiled_at to be set")
	}

	// Verify that the hash matches the source file
	sourceHash, err := ComputeSourceHash(testFile)
	if err != nil {
		t.Fatalf("Failed to compute source hash: %v", err)
	}

	if metadata.SourceHash != sourceHash {
		t.Errorf("Expected source_hash=%s, got %s", sourceHash, metadata.SourceHash)
	}
}
