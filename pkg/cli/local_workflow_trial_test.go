//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLocalWorkflowTrialMode tests the local workflow installation for trial mode
func TestLocalWorkflowTrialMode(t *testing.T) {
	// Clear the repository slug cache to ensure clean test state
	ClearCurrentRepoSlugCache()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-local-trial-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test workflow file
	testWorkflowContent := `---
description: "Test local workflow"
on:
  workflow_dispatch:
---

# Test Workflow

This is a test workflow.

## Steps

- name: Test step
  run: echo "Hello World"
`

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	testFile := filepath.Join(originalDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}
	defer os.Remove(testFile)

	// Parse the local workflow spec
	spec, err := parseWorkflowSpec("./test-workflow.md")
	if err != nil {
		t.Fatalf("Failed to parse local workflow spec: %v", err)
	}

	// Verify the spec
	if !isLocalWorkflowPath(spec.WorkflowPath) {
		t.Errorf("Expected WorkflowPath to be a local path, got: %s", spec.WorkflowPath)
	}

	if spec.WorkflowName != "test-workflow" {
		t.Errorf("Expected WorkflowName to be 'test-workflow', got: %s", spec.WorkflowName)
	}

	// Test the local workflow writing function (writeWorkflowToTrialDir)
	// First read the workflow content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test workflow file: %v", err)
	}

	result, err := writeWorkflowToTrialDir(tempDir, spec.WorkflowName, content, &TrialOptions{DisableSecurityScanner: true})
	if err != nil {
		t.Fatalf("Failed to write local workflow to trial dir: %v", err)
	}

	// Verify the file was copied correctly
	if _, err := os.Stat(result.DestPath); os.IsNotExist(err) {
		t.Errorf("Expected workflow file to be written to %s, but it doesn't exist", result.DestPath)
	}

	// Verify the content matches
	copiedContent, err := os.ReadFile(result.DestPath)
	if err != nil {
		t.Fatalf("Failed to read copied workflow file: %v", err)
	}

	if string(copiedContent) != testWorkflowContent {
		t.Errorf("Copied workflow content doesn't match original")
	}
}
