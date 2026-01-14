package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestShowUpdateSummary_DryRun tests the update summary display in dry-run mode
func TestShowUpdateSummary_DryRun(t *testing.T) {
	tests := []struct {
		name              string
		successfulUpdates []string
		failedUpdates     []updateFailure
		dryRun            bool
	}{
		{
			name:              "dry-run with successful updates",
			successfulUpdates: []string{"workflow1", "workflow2"},
			failedUpdates:     []updateFailure{},
			dryRun:            true,
		},
		{
			name:              "normal mode with successful updates",
			successfulUpdates: []string{"workflow1", "workflow2"},
			failedUpdates:     []updateFailure{},
			dryRun:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just verifies the function doesn't panic and can be called with dry-run flag
			showUpdateSummary(tt.successfulUpdates, tt.failedUpdates, tt.dryRun)
		})
	}
}

// TestUpdateActions_DryRun tests that UpdateActions respects dry-run mode
func TestUpdateActions_DryRun(t *testing.T) {
	// Create a temporary directory with an actions-lock.json
	tmpDir := testutil.TempDir(t, "test-*")
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create .github/aw directory
	awDir := filepath.Join(tmpDir, ".github", "aw")
	if err := os.MkdirAll(awDir, 0755); err != nil {
		t.Fatalf("Failed to create .github/aw directory: %v", err)
	}

	// Create an actions-lock.json
	actionsLockPath := filepath.Join(awDir, "actions-lock.json")
	actionsLock := `{
  "entries": {
    "actions/checkout@v4": {
      "repo": "actions/checkout",
      "version": "v4",
      "sha": "b4ffde65f46336ab88eb53be808477a3936bae11"
    }
  }
}`
	if err := os.WriteFile(actionsLockPath, []byte(actionsLock), 0644); err != nil {
		t.Fatalf("Failed to write actions-lock.json: %v", err)
	}

	os.Chdir(tmpDir)

	// Read the original content
	originalContent, err := os.ReadFile(actionsLockPath)
	if err != nil {
		t.Fatalf("Failed to read original actions-lock.json: %v", err)
	}

	// Run UpdateActions in dry-run mode
	err = UpdateActions(false, false, true)
	if err != nil {
		t.Logf("UpdateActions returned error (may be expected in test environment): %v", err)
	}

	// Verify the file was not modified
	afterContent, err := os.ReadFile(actionsLockPath)
	if err != nil {
		t.Fatalf("Failed to read actions-lock.json after dry-run: %v", err)
	}

	if string(originalContent) != string(afterContent) {
		t.Error("Expected file to remain unchanged in dry-run mode")
	}
}

// TestUpdateWorkflows_DryRun tests that UpdateWorkflows respects dry-run mode
func TestUpdateWorkflows_DryRun(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := testutil.TempDir(t, "test-*")
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	customWorkflowDir := filepath.Join(tmpDir, "workflows")
	if err := os.MkdirAll(customWorkflowDir, 0755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}

	// Create a workflow file with source field
	workflowContent := `---
on: push
engine: claude
source: test/repo/workflow.md@v1.0.0
---

# Test Workflow

Test content.`

	workflowPath := filepath.Join(customWorkflowDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	os.Chdir(tmpDir)

	// Read the original content
	originalContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read original workflow: %v", err)
	}

	// Run UpdateWorkflows in dry-run mode
	// This will fail because the source repository doesn't exist, but that's ok for testing
	// We're just verifying that the file is not modified
	err = UpdateWorkflows([]string{"test-workflow"}, false, false, false, "", customWorkflowDir, false, "", false, true)
	if err != nil {
		t.Logf("UpdateWorkflows returned error (expected in test environment): %v", err)
	}

	// Verify the file was not modified
	afterContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow after dry-run: %v", err)
	}

	if string(originalContent) != string(afterContent) {
		t.Error("Expected workflow file to remain unchanged in dry-run mode")
	}
}
