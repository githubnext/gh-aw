//go:build integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestLocalWorkflowIntegration(t *testing.T) {
	// Create a temporary directory
	tempDir := testutil.TempDir(t, "test-*")
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test workflow file
	workflowsDir := "workflows"
	err = os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testWorkflowPath := filepath.Join(workflowsDir, "test-local.md")
	testContent := `---
description: "Test local workflow"
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [list_commits]
---

# Test Local Workflow

This is a test local workflow.
`

	err = os.WriteFile(testWorkflowPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test parsing local workflow spec
	// Local workflows don't require a git repository or remote
	spec, err := parseWorkflowSpec("./workflows/test-local.md")
	if err != nil {
		t.Fatalf("Failed to parse local workflow spec: %v", err)
	}

	// Verify parsed spec
	if spec.WorkflowPath != "./workflows/test-local.md" {
		t.Errorf("Expected WorkflowPath './workflows/test-local.md', got %q", spec.WorkflowPath)
	}
	if spec.WorkflowName != "test-local" {
		t.Errorf("Expected WorkflowName 'test-local', got %q", spec.WorkflowName)
	}
	if spec.Version != "" {
		t.Errorf("Expected empty Version for local workflow, got %q", spec.Version)
	}
	// Local workflows have no RepoSlug since they don't come from a remote source
	if spec.RepoSlug != "" {
		t.Errorf("Expected empty RepoSlug for local workflow, got %q", spec.RepoSlug)
	}

	// Test String() method
	stringResult := spec.String()
	if stringResult != "./workflows/test-local.md" {
		t.Errorf("Expected String() './workflows/test-local.md', got %q", stringResult)
	}

	// Test buildSourceString - returns empty for local workflows (no remote source to track)
	sourceString := buildSourceString(spec)
	if sourceString != "" {
		t.Errorf("Expected buildSourceString() to return empty string for local workflow, got %q", sourceString)
	}
}
