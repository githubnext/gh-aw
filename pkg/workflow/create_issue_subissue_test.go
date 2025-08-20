package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateIssueSubissueFeature tests that the create_issue.js script includes subissue functionality
func TestCreateIssueSubissueFeature(t *testing.T) {
	// Test that the script contains the subissue detection logic
	if !strings.Contains(createIssueScript, "context.payload?.issue?.number") {
		t.Error("Expected create_issue.js to check for parent issue context")
	}

	// Test that the script modifies the body when in issue context
	if !strings.Contains(createIssueScript, "Related to #${parentIssueNumber}") {
		t.Error("Expected create_issue.js to add parent issue reference to body")
	}

	// Test that the script creates a comment on the parent issue
	if !strings.Contains(createIssueScript, "github.rest.issues.createComment") {
		t.Error("Expected create_issue.js to create comment on parent issue")
	}

	// Test that the script has proper error handling for comment creation
	if !strings.Contains(createIssueScript, "Warning: Could not add comment to parent issue") {
		t.Error("Expected create_issue.js to have error handling for parent issue comment")
	}

	// Test console logging for debugging
	if !strings.Contains(createIssueScript, "Detected issue context, parent issue") {
		t.Error("Expected create_issue.js to log when issue context is detected")
	}
}

// TestCreateIssueWorkflowCompilation tests that workflows with output.issue still compile correctly
func TestCreateIssueWorkflowCompilationWithSubissue(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "subissue-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test Subissue Feature  
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: claude
output:
  issue:
    title-prefix: "[test] "
    labels: [automation, test]
---

# Test Workflow

This is a test workflow that should create an issue with subissue functionality.
Write output to ${{ env.GITHUB_AW_OUTPUT }}.`

	testFile := filepath.Join(tmpDir, "test-subissue.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow with output.issue: %v", err)
	}

	// Read the generated lock file to verify content
	lockFile := filepath.Join(tmpDir, "test-subissue.lock.yml")
	lockContentBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContent := string(lockContentBytes)

	// Verify the compiled workflow includes the subissue functionality
	if !strings.Contains(lockContent, "context.payload?.issue?.number") {
		t.Error("Expected compiled workflow to include subissue detection")
	}

	if !strings.Contains(lockContent, "Created related issue: #${issue.number}") {
		t.Error("Expected compiled workflow to include parent issue comment")
	}

	// Verify it still has the standard create_issue job structure
	if !strings.Contains(lockContent, "create_issue:") {
		t.Error("Expected create_issue job to be present")
	}

	if !strings.Contains(lockContent, "permissions:\n      contents: read\n      issues: write") {
		t.Error("Expected correct permissions in create_issue job")
	}
}
