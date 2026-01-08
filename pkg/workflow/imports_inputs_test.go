package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestImportWithInputs tests that imports with inputs correctly substitute values
func TestImportWithInputs(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-import-inputs-*")

	// Create shared workflow file with inputs
	sharedPath := filepath.Join(tempDir, "shared", "data-fetch.md")
	sharedDir := filepath.Dir(sharedPath)
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
inputs:
  count:
    description: Number of items to fetch
    type: number
    default: 100
  category:
    description: Category to filter
    type: string
    default: "general"
---

# Data Fetch Instructions

Fetch ${{ github.aw.inputs.count }} items from the ${{ github.aw.inputs.category }} category.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports shared with inputs
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - path: shared/data-fetch.md
    inputs:
      count: 50
      category: "technology"
---

# Test Workflow

This workflow tests import with inputs.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(lockFileContent)

	// Extract the first prompt heredoc section
	// The prompt is written as: cat << 'PROMPT_EOF' > "$GH_AW_PROMPT"
	// and ends with: PROMPT_EOF
	promptSection := ""
	startMarker := "cat << 'PROMPT_EOF'"
	endMarker := "\n          PROMPT_EOF" // Note the indentation
	if idx := strings.Index(lockContent, startMarker); idx != -1 {
		contentStart := idx + len(startMarker)
		// Find the end marker after the start
		if endIdx := strings.Index(lockContent[contentStart:], endMarker); endIdx > 0 {
			promptSection = lockContent[contentStart : contentStart+endIdx]
		}
	}

	// The prompt should contain the substituted values
	if promptSection != "" {
		// Check substituted values are present
		if !strings.Contains(promptSection, "50") {
			t.Errorf("Expected prompt section to contain substituted count value '50', got: %s", promptSection)
		}
		if !strings.Contains(promptSection, "technology") {
			t.Errorf("Expected prompt section to contain substituted category value 'technology', got: %s", promptSection)
		}

		// Check that the agentics.inputs expressions are NOT in the prompt (they should be substituted)
		if strings.Contains(promptSection, "github.aw.inputs.count") {
			t.Error("Prompt section should not contain unsubstituted github.aw.inputs.count expression")
		}
		if strings.Contains(promptSection, "github.aw.inputs.category") {
			t.Error("Prompt section should not contain unsubstituted github.aw.inputs.category expression")
		}
	} else {
		t.Error("Could not find prompt heredoc section in lock file")
	}
}

// TestImportWithInputsStringFormat tests that string import format still works
func TestImportWithInputsStringFormat(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-import-string-*")

	// Create shared workflow file (no inputs needed for this test)
	sharedPath := filepath.Join(tempDir, "shared", "simple.md")
	sharedDir := filepath.Dir(sharedPath)
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
tools:
  bash:
    - "echo *"
---

# Simple Shared Instructions

Do something simple.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports using string format
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - shared/simple.md
---

# Test Workflow

This workflow tests that string imports still work.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(lockFileContent)

	// Verify that the shared content is included
	if !strings.Contains(lockContent, "Simple Shared Instructions") {
		t.Error("Expected lock file to contain content from shared workflow")
	}
}

// TestImportInputsExpressionValidation tests that github.aw.inputs expressions are allowed
func TestImportInputsExpressionValidation(t *testing.T) {
	// This test just verifies the expression is allowed in the markdown content
	content := "Process ${{ github.aw.inputs.limit }} items."
	err := workflow.ValidateExpressionSafetyPublic(content)
	if err != nil {
		t.Errorf("Expression validation should allow github.aw.inputs.* expressions: %v", err)
	}
}
