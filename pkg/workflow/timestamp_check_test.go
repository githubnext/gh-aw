package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWorkflowTimestampCheckUsesJavaScript verifies that the workflow timestamp check uses JavaScript instead of inline bash
func TestWorkflowTimestampCheckUsesJavaScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workflow-timestamp-js-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	workflowContent := `---
on:
  workflow_dispatch:
engine: claude
---

# Test Workflow

This is a test workflow to verify timestamp checking uses JavaScript.
`
	workflowFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = compiler.CompileWorkflow(workflowFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify the "Check workflow file timestamps" step exists
	if !strings.Contains(lockContentStr, "Check workflow file timestamps") {
		t.Error("Expected 'Check workflow file timestamps' step to be present")
	}

	// Verify it uses actions/github-script instead of inline bash
	timestampCheckIdx := strings.Index(lockContentStr, "Check workflow file timestamps")
	if timestampCheckIdx == -1 {
		t.Fatal("Could not find timestamp check step")
	}

	// Extract a section after the timestamp check (next ~500 chars should contain the uses directive)
	sectionAfterTimestampCheck := lockContentStr[timestampCheckIdx:]
	if len(sectionAfterTimestampCheck) > 500 {
		sectionAfterTimestampCheck = sectionAfterTimestampCheck[:500]
	}

	// Verify it uses actions/github-script@
	if !strings.Contains(sectionAfterTimestampCheck, "uses: actions/github-script@") {
		t.Error("Expected timestamp check to use actions/github-script")
	}

	// Verify it does NOT use inline bash (run: |)
	// Find the next step boundary or job boundary to limit the search
	nextStepOrJobIdx := strings.Index(sectionAfterTimestampCheck, "- name:")
	if nextStepOrJobIdx == -1 {
		nextStepOrJobIdx = strings.Index(sectionAfterTimestampCheck, "  agent:")
	}
	if nextStepOrJobIdx == -1 {
		nextStepOrJobIdx = len(sectionAfterTimestampCheck)
	}
	timestampCheckSection := sectionAfterTimestampCheck[:nextStepOrJobIdx]

	// The step should NOT contain "run: |" (which would indicate inline bash)
	if strings.Contains(timestampCheckSection, "run: |") {
		t.Error("Expected timestamp check to NOT use inline bash (run: |)")
	}

	// Verify the JavaScript content uses GitHub API instead of filesystem
	if !strings.Contains(lockContentStr, "GITHUB_WORKFLOW") {
		t.Error("Expected JavaScript to reference GITHUB_WORKFLOW")
	}
	if !strings.Contains(lockContentStr, "github.rest.repos.listCommits") {
		t.Error("Expected JavaScript to use GitHub REST API (github.rest.repos.listCommits)")
	}
	if !strings.Contains(lockContentStr, "context.repo.owner") {
		t.Error("Expected JavaScript to use context.repo.owner for API calls")
	}

	// Verify we're NOT using filesystem operations (since repo is not checked out)
	if strings.Contains(timestampCheckSection, "fs.statSync") {
		t.Error("Expected timestamp check to NOT use fs.statSync (should use GitHub API instead)")
	}
	if strings.Contains(timestampCheckSection, "GITHUB_WORKSPACE") {
		t.Error("Expected timestamp check to NOT reference GITHUB_WORKSPACE (should use GitHub API instead)")
	}
}
