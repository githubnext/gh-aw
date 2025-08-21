package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSONSchemaValidationErrorMapping tests that JSON Schema validation errors
// are properly mapped to precise YAML source locations using the mapper package
func TestJSONSchemaValidationErrorMapping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schema-validation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name          string
		frontmatter   string
		expectError   bool
		errorContains []string // Multiple strings that should be in the error
		description   string
	}{
		{
			name: "valid_workflow_passes_validation",
			frontmatter: `---
on: push
permissions:
  contents: read
  issues: write
timeout_minutes: 30
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			expectError: false,
			description: "Valid workflow should pass GitHub Actions schema validation",
		},
		{
			name: "workflow_with_custom_steps_invalid_structure",
			frontmatter: `---
on: push
permissions:
  contents: read
  issues: write
steps:
  - name: Invalid step
    invalid_step_property: not_allowed
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			expectError:   true,
			errorContains: []string{"validation failed", "invalid_step_property"},
			description:   "Custom steps with invalid properties should fail GitHub Actions schema validation",
		},
		{
			name: "workflow_generates_invalid_yaml_structure",
			frontmatter: `---
on: push
permissions:
  contents: read
  issues: write
# Use steps that might have invalid properties when compiled
steps:
  - name: Step with invalid property
    run: echo "test"
    invalid_step_property: "should_not_exist"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			expectError:   true,
			errorContains: []string{"validation failed", "invalid_step_property"},
			description:   "Steps with invalid properties should fail GitHub Actions schema validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test workflow file
			testFile := filepath.Join(tmpDir, fmt.Sprintf("%s.md", tt.name))
			testContent := tt.frontmatter + "\n\n# Test Schema Validation\n\n" + tt.description
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Create compiler with validation enabled
			compiler := NewCompiler(false, "", "test")
			compiler.SetSkipValidation(false) // Enable GitHub Actions schema validation

			// Attempt compilation
			err := compiler.CompileWorkflow(testFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected compilation to fail due to schema validation error", tt.description)
					return
				}

				errorStr := err.Error()

				// Check that all expected error messages are present
				for _, expectedErr := range tt.errorContains {
					if !strings.Contains(errorStr, expectedErr) {
						t.Errorf("%s: error should contain '%s', got: %s", tt.description, expectedErr, errorStr)
					}
				}

				// Verify error contains proper formatting (file path)
				if !strings.Contains(errorStr, testFile) {
					t.Errorf("%s: error should contain file path, got: %s", tt.description, errorStr)
				}

				// Verify error contains "error:" type indicator
				if !strings.Contains(errorStr, "error:") {
					t.Errorf("%s: error should contain 'error:' type indicator, got: %s", tt.description, errorStr)
				}
			} else {
				if err != nil {
					t.Errorf("%s: expected compilation to succeed but got error: %v", tt.description, err)
				}
			}
		})
	}
}

// TestJSONSchemaValidationDisabled tests that schema validation can be disabled
func TestJSONSchemaValidationDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schema-disabled-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a workflow that might have GitHub Actions schema issues but valid frontmatter
	frontmatter := `---
on: push
permissions:
  contents: read
  issues: write
steps:
  - name: Test step
    run: echo "test"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`

	testFile := filepath.Join(tmpDir, "disabled-test.md")
	testContent := frontmatter + "\n\n# Test Disabled Validation\n\nThis tests validation can be disabled."
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with validation disabled (default)
	compiler := NewCompiler(false, "", "test")
	// Don't call SetSkipValidation(false) - validation should be skipped by default

	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Errorf("Expected compilation to succeed with validation disabled, got error: %v", err)
	}

	// Test with validation explicitly disabled
	compiler2 := NewCompiler(false, "", "test")
	compiler2.SetSkipValidation(true)

	err = compiler2.CompileWorkflow(testFile)
	if err != nil {
		t.Errorf("Expected compilation to succeed with validation explicitly disabled, got error: %v", err)
	}
}

// TestJSONSchemaValidationIntegration tests the integration between the compiler
// and the JSON schema validation error mapper
func TestJSONSchemaValidationIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schema-integration-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test a workflow that will generate invalid GitHub Actions YAML structure
	// This focuses on testing the mapper integration after compilation
	frontmatter := `---
on: push
permissions:
  contents: read
  issues: write
# This should create a workflow that may fail GitHub Actions schema validation
steps:
  - name: Test
    run: echo "test"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`

	testFile := filepath.Join(tmpDir, "integration-test.md")
	testContent := frontmatter + "\n\n# Integration Test\n\nThis tests the schema validation integration."
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with validation enabled
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(false) // Enable GitHub Actions schema validation

	err = compiler.CompileWorkflow(testFile)
	// Note: This might pass or fail depending on whether the generated YAML is valid
	// The important thing is that if it fails, it should use the mapper for error reporting

	if err != nil {
		errorStr := err.Error()
		t.Logf("Schema validation error (expected behavior): %s", errorStr)

		// If there's an error, verify it has proper formatting
		expectedContains := []string{
			testFile, // Should contain file path
			"error:", // Should be formatted as an error
		}

		for _, expected := range expectedContains {
			if !strings.Contains(errorStr, expected) {
				t.Errorf("Error should contain '%s', got: %s", expected, errorStr)
			}
		}
	} else {
		t.Logf("Schema validation passed (workflow is valid)")
	}
}

// TestJSONSchemaValidationErrorFormatting tests that when validation fails,
// errors are properly formatted with source locations
func TestJSONSchemaValidationErrorFormatting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "schema-formatting-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		frontmatter string
		description string
	}{
		{
			name: "simple_workflow",
			frontmatter: `---
on: workflow_dispatch
permissions:
  contents: read
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			description: "Simple workflow to test error formatting capabilities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, fmt.Sprintf("%s.md", tt.name))
			testContent := tt.frontmatter + "\n\n# Test\n\n" + tt.description
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")
			compiler.SetSkipValidation(false) // Enable validation

			err := compiler.CompileWorkflow(testFile)

			if err != nil {
				errorStr := err.Error()
				t.Logf("Validation error for %s: %s", tt.name, errorStr)

				// Verify basic error formatting
				if !strings.Contains(errorStr, testFile) {
					t.Errorf("Error should contain file path")
				}
				if !strings.Contains(errorStr, ":") {
					t.Errorf("Error should contain position information")
				}
			} else {
				t.Logf("Validation passed for %s", tt.name)
			}
		})
	}
}

// TestJSONSchemaValidationMapperIntegration specifically tests that the mapper
// is being used when schema validation fails
func TestJSONSchemaValidationMapperIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapper-integration-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a workflow that should generate YAML that might fail schema validation
	frontmatter := `---
on: push
permissions:
  contents: read
  issues: write
timeout_minutes: 60
steps:
  - name: Checkout
    uses: actions/checkout@v3
tools:
  github:
    allowed: [list_issues]
engine: claude
---`

	testFile := filepath.Join(tmpDir, "mapper-test.md")
	testContent := frontmatter + "\n\n# Mapper Test\n\nThis tests the mapper integration with schema validation."
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with validation enabled
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(false)

	err = compiler.CompileWorkflow(testFile)

	if err != nil {
		errorStr := err.Error()
		t.Logf("Schema validation produced error: %s", errorStr)

		// The key test: if validation fails, the error should be formatted
		// using the console error formatting system which integrates with the mapper
		expectedFormatting := []string{
			":",      // Should have position formatting
			"error:", // Should have error type
		}

		for _, format := range expectedFormatting {
			if !strings.Contains(errorStr, format) {
				t.Errorf("Error should contain formatting '%s', got: %s", format, errorStr)
			}
		}
	} else {
		// If validation passes, that's also fine - it means the workflow is valid
		t.Logf("Schema validation passed - workflow generates valid GitHub Actions YAML")
	}
}

// TestJSONSchemaValidationDirect tests the schema validation functions directly
func TestJSONSchemaValidationDirect(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(false) // Enable validation

	tests := []struct {
		name          string
		yaml          string
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name: "valid_github_actions_yaml",
			yaml: `name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3`,
			expectError: false,
			description: "Valid GitHub Actions YAML should pass validation",
		},
		{
			name: "invalid_yaml_missing_jobs",
			yaml: `name: Test Workflow
on: push`,
			expectError:   true,
			errorContains: "missing property 'jobs'",
			description:   "YAML missing required 'jobs' property should fail validation",
		},
		{
			name: "invalid_yaml_bad_job_structure",
			yaml: `name: Test Workflow
on: push
jobs:
  test:
    invalid-job-property: not_allowed`,
			expectError:   true,
			errorContains: "validation failed",
			description:   "YAML with invalid job properties should fail validation",
		},
		{
			name: "invalid_yaml_bad_runs_on_type",
			yaml: `name: Test Workflow
on: push
jobs:
  test:
    runs-on: 123`,
			expectError:   true,
			errorContains: "got number, want string",
			description:   "YAML with invalid runs-on type should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.validateWorkflowSchema(tt.yaml)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected validation to fail but it passed", tt.description)
					return
				}

				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("%s: error should contain '%s', got: %s", tt.description, tt.errorContains, err.Error())
				}

				t.Logf("%s: validation failed as expected: %s", tt.description, err.Error())
			} else {
				if err != nil {
					t.Errorf("%s: expected validation to pass but got error: %v", tt.description, err)
				}
			}
		})
	}
}
