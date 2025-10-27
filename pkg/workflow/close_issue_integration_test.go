//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloseIssueWorkflowCompilation(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test workflow with close-issue safe output
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  close-issue:
    required-labels:
      - stale
    outcome:
      - completed
      - not_planned
    max: 3
---

# Test Close Issue Safe Output

Close stale issues using the close-issue tool.
`

	workflowPath := filepath.Join(tmpDir, "test-close-issue.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile the workflow with proper compiler initialization
	c := NewCompiler(false, "", "test")
	err = c.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	lockPath := filepath.Join(tmpDir, "test-close-issue.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify the close_issue job exists
	if !strings.Contains(lockStr, "close_issue:") {
		t.Error("Expected close_issue job in compiled workflow")
	}

	// Verify required environment variables are set
	expectedEnvVars := []string{
		"GH_AW_REQUIRED_LABELS",
		"GH_AW_ALLOWED_OUTCOMES",
		"GH_AW_AGENT_OUTPUT",
	}

	for _, envVar := range expectedEnvVars {
		if !strings.Contains(lockStr, envVar) {
			t.Errorf("Expected environment variable %q in compiled workflow", envVar)
		}
	}

	// Verify required labels are passed
	if !strings.Contains(lockStr, `GH_AW_REQUIRED_LABELS: "stale"`) {
		t.Error("Expected required labels to be set in environment")
	}

	// Verify allowed outcomes are passed
	if !strings.Contains(lockStr, `GH_AW_ALLOWED_OUTCOMES: "completed,not_planned"`) {
		t.Error("Expected allowed outcomes to be set in environment")
	}

	// Verify job permissions
	if !strings.Contains(lockStr, "issues: write") {
		t.Error("Expected issues: write permission in close_issue job")
	}

	// Verify job timeout
	if !strings.Contains(lockStr, "timeout-minutes: 10") {
		t.Error("Expected timeout-minutes: 10 in close_issue job")
	}

	// Verify job outputs
	if !strings.Contains(lockStr, "issue_number:") || !strings.Contains(lockStr, "issue_url:") {
		t.Error("Expected job outputs for issue_number and issue_url")
	}

	// Verify the close_issue job depends on the agent job
	if !strings.Contains(lockStr, "needs:") || !strings.Contains(lockStr, "- agent") {
		t.Error("Expected close_issue job to depend on agent job")
	}
}

func TestCloseIssueWithTargetConfiguration(t *testing.T) {
	tmpDir := t.TempDir()

	workflowContent := `---
on:
  issues:
    types: [labeled]
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  close-issue:
    target: "*"
    required-labels:
      - bug
      - wontfix
---

# Close Issues with Target Wildcard

Close any issue that has both "bug" and "wontfix" labels.
`

	workflowPath := filepath.Join(tmpDir, "test-close-target.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	c := NewCompiler(false, "", "test")
	err = c.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "test-close-target.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify target is set to wildcard
	if !strings.Contains(lockStr, `GH_AW_CLOSE_TARGET: "*"`) {
		t.Error("Expected target to be set to wildcard in environment")
	}

	// Verify required labels include both bug and wontfix
	if !strings.Contains(lockStr, `GH_AW_REQUIRED_LABELS: "bug,wontfix"`) {
		t.Error("Expected both 'bug' and 'wontfix' in required labels")
	}
}
