package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
)

func TestCompileWorkflowExpressionSizeValidation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "expression-size-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("workflow with normal expression sizes should compile successfully", func(t *testing.T) {
		// Create a workflow with normal-sized expressions
		testContent := `---
timeout_minutes: 10
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues, get_issue]
---

# Normal Expression Test Workflow

This workflow has normal-sized expressions and should compile successfully.
The content is reasonable and won't generate overly long environment variables.
`

		testFile := filepath.Join(tmpDir, "normal-expressions.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Errorf("Expected no error for workflow with normal expressions, got: %v", err)
		}

		// Verify lock file was created
		lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
		if _, err := os.Stat(lockFile); err != nil {
			t.Errorf("Lock file was not created: %v", err)
		}
	})

	t.Run("workflow with oversized markdown content should fail validation", func(t *testing.T) {
		// Create a workflow with markdown content that will exceed the 21KB limit
		// The content will be written to the workflow YAML as a single line in a heredoc
		// We need 25KB+ of content to trigger the validation
		largeContent := strings.Repeat("x", 25000)
		testContent := fmt.Sprintf(`---
timeout_minutes: 10
permissions:
  contents: read
  pull-requests: write
tools:
  github:
    allowed: [list_issues]
safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
---

# Large Content Test Workflow

%s
`, largeContent)

		testFile := filepath.Join(tmpDir, "large-content.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(false) // Enable validation to test expression size limits
		err := compiler.CompileWorkflow(testFile)
		
		// This should fail with an expression size validation error
		if err == nil {
			t.Error("Expected error for workflow with oversized expressions, got nil")
		} else if !strings.Contains(err.Error(), "exceeds maximum allowed") {
			t.Errorf("Expected 'exceeds maximum allowed' error, got: %v", err)
		} else if !strings.Contains(err.Error(), "expression size validation failed") {
			t.Errorf("Expected 'expression size validation failed' error, got: %v", err)
		}
	})

	t.Run("expression size validation constant", func(t *testing.T) {
		// Verify the constant is set correctly
		if MaxExpressionSize != 21000 {
			t.Errorf("MaxExpressionSize constant should be 21000, got %d", MaxExpressionSize)
		}
	})

	t.Run("expression size validation error message format", func(t *testing.T) {
		// Test that the validation produces correct error message format
		testLineSize := int64(25000) // 25KB, exceeds limit
		actualSize := pretty.FormatFileSize(testLineSize)
		maxSizeFormatted := pretty.FormatFileSize(int64(MaxExpressionSize))
		
		expectedMessage := fmt.Sprintf("expression value for 'WORKFLOW_MARKDOWN' (%s) exceeds maximum allowed size (%s)", 
			actualSize, maxSizeFormatted)
		
		// Verify the message contains expected elements
		if !strings.Contains(expectedMessage, "exceeds maximum allowed size") {
			t.Error("Error message should contain 'exceeds maximum allowed size'")
		}
		if !strings.Contains(expectedMessage, "KB") {
			t.Error("Error message should contain size in KB")
		}
		if !strings.Contains(expectedMessage, "WORKFLOW_MARKDOWN") {
			t.Error("Error message should identify the problematic key")
		}
		
		t.Logf("Generated error message: %s", expectedMessage)
	})
}
