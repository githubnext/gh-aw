package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckActInstalled(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		wantErr bool
	}{
		{
			name:    "verbose mode",
			verbose: true,
			wantErr: false, // Should not error if act is in PATH
		},
		{
			name:    "quiet mode",
			verbose: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkActInstalled(tt.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkActInstalled() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTestWorkflowLocallyValidation(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		event        string
		verbose      bool
		wantErr      bool
	}{
		{
			name:         "empty workflow name",
			workflowName: "",
			event:        "workflow_dispatch",
			verbose:      false,
			wantErr:      true,
		},
		{
			name:         "single workflow test",
			workflowName: "test-workflow",
			event:        "workflow_dispatch",
			verbose:      false,
			wantErr:      true, // Will error on missing workflow file
		},
		{
			name:         "custom event type",
			workflowName: "test-workflow",
			event:        "push",
			verbose:      true,
			wantErr:      true, // Will error on missing workflow file
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TestWorkflowLocally(tt.workflowName, tt.event, tt.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestWorkflowLocally() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTestSingleWorkflowLocally(t *testing.T) {
	// Create a temporary test environment
	tempDir := t.TempDir()

	// Create a mock .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a mock markdown file first
	mdFile := filepath.Join(workflowsDir, "test-workflow.md")
	mdContent := `---
on:
  workflow_dispatch:
---

# Test Workflow

This is a test workflow.
`
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to create mock markdown file: %v", err)
	}

	// Create a corresponding mock lock file
	lockFile := filepath.Join(workflowsDir, "test-workflow.lock.yml")
	lockContent := `name: Test Workflow
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`
	if err := os.WriteFile(lockFile, []byte(lockContent), 0644); err != nil {
		t.Fatalf("Failed to create mock lock file: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name         string
		workflowName string
		event        string
		verbose      bool
		wantErr      bool
	}{
		{
			name:         "valid workflow test",
			workflowName: "test-workflow",
			event:        "workflow_dispatch",
			verbose:      false,
			wantErr:      true, // Will still error due to missing act installation in test environment
		},
		{
			name:         "valid workflow test verbose",
			workflowName: "test-workflow",
			event:        "push",
			verbose:      true,
			wantErr:      true, // Will still error due to missing act installation in test environment
		},
		{
			name:         "non-existent workflow",
			workflowName: "missing-workflow",
			event:        "workflow_dispatch",
			verbose:      false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testSingleWorkflowLocally(tt.workflowName, tt.event, tt.verbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("testSingleWorkflowLocally() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
