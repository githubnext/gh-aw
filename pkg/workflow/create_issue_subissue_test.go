package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestCreateIssueSubissueFeature tests that the create_issue.js script includes subissue functionality
func TestCreateIssueSubissueFeature(t *testing.T) {
	script := getCreateIssueScript()
	// Test that the script contains the subissue detection logic
	if !strings.Contains(script, "context.payload?.issue?.number") {
		t.Error("Expected create_issue.js to check for parent issue context")
	}

	// Test that the script modifies the body when in issue context
	if !strings.Contains(script, "Related to #${effectiveParentIssueNumber}") {
		t.Error("Expected create_issue.js to add parent issue reference to body")
	}

	// Test that the script supports explicit parent field
	if !strings.Contains(script, "createIssueItem.parent") {
		t.Error("Expected create_issue.js to support explicit parent field")
	}

	// Test that the script uses effectiveParentIssueNumber
	if !strings.Contains(script, "effectiveParentIssueNumber") {
		t.Error("Expected create_issue.js to use effectiveParentIssueNumber variable")
	}

	// Test that the script includes GraphQL sub-issue linking
	if !strings.Contains(script, "addSubIssue") {
		t.Error("Expected create_issue.js to include addSubIssue GraphQL mutation")
	}

	// Test that the script calls github.graphql for sub-issue linking
	if !strings.Contains(script, "github.graphql(addSubIssueMutation") {
		t.Error("Expected create_issue.js to call github.graphql for sub-issue linking")
	}

	// Test that the script fetches node IDs before linking
	if !strings.Contains(script, "getIssueNodeIdQuery") {
		t.Error("Expected create_issue.js to fetch issue node IDs before linking")
	}

	// Test that the script creates a comment on the parent issue
	if !strings.Contains(script, "github.rest.issues.createComment") {
		t.Error("Expected create_issue.js to create comment on parent issue")
	}

	// Test that the script has proper error handling for sub-issue linking
	if !strings.Contains(script, "Warning: Could not link sub-issue to parent") {
		t.Error("Expected create_issue.js to have error handling for sub-issue linking")
	}

	// Test console logging for debugging
	if !strings.Contains(script, "Detected issue context, parent issue") {
		t.Error("Expected create_issue.js to log when issue context is detected")
	}

	// Test that it logs successful sub-issue linking
	if !strings.Contains(script, "Successfully linked issue #") {
		t.Error("Expected create_issue.js to log successful sub-issue linking")
	}
}

// TestCreateIssueWorkflowCompilation tests that workflows with safe-outputs.create-issue still compile correctly
func TestCreateIssueWorkflowCompilationWithSubissue(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "subissue-test")

	testContent := `---
name: Test Subissue Feature  
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
strict: false
safe-outputs:
  create-issue:
    title-prefix: "[test] "
    labels: [automation, test]
---

# Test Workflow

This is a test workflow that should create an issue with subissue functionality.
Write output to ${{ env.GH_AW_SAFE_OUTPUTS }}.`

	testFile := filepath.Join(tmpDir, "test-subissue.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	compiler.SetActionMode(ActionModeDev) // Use dev mode with local action paths
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow with output.create-issue: %v", err)
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

	// Verify GraphQL sub-issue linking code is present
	if !strings.Contains(lockContent, "addSubIssue") {
		t.Error("Expected compiled workflow to include addSubIssue GraphQL mutation")
	}

	if !strings.Contains(lockContent, "github.graphql(addSubIssueMutation") {
		t.Error("Expected compiled workflow to call github.graphql for sub-issue linking")
	}

	if !strings.Contains(lockContent, "Successfully linked issue #") {
		t.Error("Expected compiled workflow to log successful sub-issue linking")
	}

	// Verify it still has the standard safe_outputs job structure
	if !strings.Contains(lockContent, "safe_outputs:") {
		t.Error("Expected safe_outputs job to be present")
	}

	if !strings.Contains(lockContent, "permissions:\n      contents: read\n      issues: write") {
		t.Error("Expected correct permissions in safe_outputs job")
	}
}
