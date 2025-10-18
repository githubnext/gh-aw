package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateIssueBackwardCompatibility ensures existing workflows without assignees still compile correctly
func TestCreateIssueBackwardCompatibility(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "backward-compat-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with an existing workflow format (no assignees)
	testContent := `---
name: Legacy Workflow Format
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
    title-prefix: "[legacy] "
    labels: [automation]
    max: 2
---

# Legacy Workflow

This workflow uses the old format without assignees and should continue to work.
`

	testFile := filepath.Join(tmpDir, "legacy-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Legacy workflow should compile without errors: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "legacy-workflow.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that create_issue job exists
	if !strings.Contains(compiledStr, "create_issue:") {
		t.Error("Expected create_issue job in compiled workflow")
	}

	// Verify that JavaScript step is present
	if !strings.Contains(compiledStr, "Create Output Issue") {
		t.Error("Expected Create Output Issue step in compiled workflow")
	}

	// Verify that no assignee steps are present
	if strings.Contains(compiledStr, "Assign issue to") {
		t.Error("Did not expect assignee steps in legacy workflow")
	}

	// Verify that outputs are still set correctly
	if !strings.Contains(compiledStr, "issue_number: ${{ steps.create_issue.outputs.issue_number }}") {
		t.Error("Expected issue_number output in compiled workflow")
	}
	if !strings.Contains(compiledStr, "issue_url: ${{ steps.create_issue.outputs.issue_url }}") {
		t.Error("Expected issue_url output in compiled workflow")
	}
}

// TestCreateIssueMinimalConfiguration ensures minimal configuration still works
func TestCreateIssueMinimalConfiguration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "minimal-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with minimal configuration (just enabling create-issue)
	testContent := `---
name: Minimal Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
---

# Minimal Workflow

Create an issue with minimal configuration.
`

	testFile := filepath.Join(tmpDir, "minimal-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Minimal workflow should compile without errors: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "minimal-workflow.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that create_issue job exists
	if !strings.Contains(compiledStr, "create_issue:") {
		t.Error("Expected create_issue job in compiled workflow")
	}

	// Verify that no assignee steps are present
	if strings.Contains(compiledStr, "Assign issue to") {
		t.Error("Did not expect assignee steps in minimal workflow")
	}

	// Verify basic job structure
	if !strings.Contains(compiledStr, "permissions:") {
		t.Error("Expected permissions section in create_issue job")
	}
	if !strings.Contains(compiledStr, "issues: write") {
		t.Error("Expected issues: write permission in create_issue job")
	}
}
