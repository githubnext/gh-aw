package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWebSearchValidationForCopilot tests that when a Copilot workflow uses web-search,
// compilation fails with an appropriate error message
func TestWebSearchValidationForCopilot(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that uses web-search with Copilot engine (which doesn't support web-search)
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
tools:
  web-search:
---

# Test Workflow

Search the web for information.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow - should fail
	err := compiler.CompileWorkflow(workflowPath)
	if err == nil {
		t.Fatal("Expected compilation to fail for Copilot engine with web-search tool, but it succeeded")
	}

	// Check that the error message mentions web-search and copilot
	errMsg := err.Error()
	if !strings.Contains(errMsg, "web-search") {
		t.Errorf("Expected error message to mention 'web-search', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "copilot") {
		t.Errorf("Expected error message to mention 'copilot', got: %s", errMsg)
	}
}

// TestWebSearchValidationForClaude tests that when a Claude workflow uses web-search,
// compilation succeeds (because Claude has native support)
func TestWebSearchValidationForClaude(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that uses web-search with Claude engine (which supports web-search)
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
tools:
  web-search:
---

# Test Workflow

Search the web for information.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow - should succeed
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Expected compilation to succeed for Claude engine with web-search tool, but got error: %v", err)
	}

	// Verify the lock file was created
	lockFile := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}

	// Read and verify the lock file contains web-search configuration
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)
	if !strings.Contains(lockStr, "WebSearch") {
		t.Errorf("Expected Claude workflow to have WebSearch in allowed tools, but it didn't")
	}
}

// TestWebSearchValidationForCodex tests that when a Codex workflow uses web-search,
// compilation succeeds (because Codex has native support)
func TestWebSearchValidationForCodex(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that uses web-search with Codex engine (which supports web-search)
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: codex
tools:
  web-search:
---

# Test Workflow

Search the web for information.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow - should succeed
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Expected compilation to succeed for Codex engine with web-search tool, but got error: %v", err)
	}

	// Verify the lock file was created
	lockFile := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}
}

// TestNoWebSearchNoValidation tests that when a workflow doesn't use web-search,
// compilation succeeds regardless of engine support
func TestNoWebSearchNoValidation(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that doesn't use web-search with Copilot engine
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
tools:
  github:
---

# Test Workflow

Do something without web search.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow - should succeed
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Expected compilation to succeed for workflow without web-search, but got error: %v", err)
	}
}
