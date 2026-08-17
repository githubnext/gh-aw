//go:build integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestValidateAcceptsIssueFieldActivityTypes ensures that `gh aw validate` (via
// CompileWorkflows with Validate: true, mirroring the validate command) accepts
// the `typed`, `untyped`, `field_added`, and `field_removed` issues activity
// types added to the GitHub Actions schema patch in the Makefile.
func TestValidateAcceptsIssueFieldActivityTypes(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	tmpDir, err := os.MkdirTemp("", "validate-issues-types-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// CompileWorkflows requires running inside a git repository.
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowContent := `---
on:
  issues:
    types: [typed, untyped, field_added, field_removed]
engine: copilot
---

# Test Workflow

This workflow reacts to issue field activity type changes.
`
	workflowFile := filepath.Join(workflowsDir, "issue-field-activity.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	config := CompileConfig{
		MarkdownFiles: []string{},
		Validate:      true,
		NoEmit:        true,
		WorkflowDir:   workflowsDir,
	}

	if _, err := CompileWorkflows(context.Background(), config); err != nil {
		t.Fatalf("gh aw validate should accept issues type-change activity types, got error: %v", err)
	}

	// NoEmit should prevent lock file generation during validation.
	lockFile := filepath.Join(workflowsDir, "issue-field-activity.lock.yml")
	if _, statErr := os.Stat(lockFile); statErr == nil {
		t.Error("Expected no lock file to be created when validating with NoEmit")
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("Failed to stat lock file: %v", statErr)
	}
}
