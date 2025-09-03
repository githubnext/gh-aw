package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushToBranchConfigParsing(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with push-to-branch configuration
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-branch:
    branch: feature-updates
    target: "triggering"
---

# Test Push to Branch

This is a test workflow to validate push-to-branch configuration parsing.

Please make changes and push them to the feature branch.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	if err := compiler.CompileWorkflow(mdFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that push_to_branch job is generated
	if !strings.Contains(lockContentStr, "push_to_branch:") {
		t.Errorf("Generated workflow should contain push_to_branch job")
	}

	// Verify that the branch configuration is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_BRANCH: \"feature-updates\"") {
		t.Errorf("Generated workflow should contain branch configuration")
	}

	// Verify that the target configuration is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_TARGET: \"triggering\"") {
		t.Errorf("Generated workflow should contain target configuration")
	}

	// Verify that required permissions are present
	if !strings.Contains(lockContentStr, "contents: write") {
		t.Errorf("Generated workflow should have contents: write permission")
	}

	// Verify that the job depends on the main workflow job
	if !strings.Contains(lockContentStr, "needs: test-push-to-branch") {
		t.Errorf("Generated workflow should have dependency on main job")
	}

	// Verify conditional execution for pull request context
	if !strings.Contains(lockContentStr, "if: github.event.pull_request.number") {
		t.Errorf("Generated workflow should have pull request context condition")
	}
}

func TestPushToBranchWithTargetAsterisk(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with target: "*"
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-branch:
    branch: feature-updates
    target: "*"
---

# Test Push to Branch with Target *

This workflow allows pushing to any pull request.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch-asterisk.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	if err := compiler.CompileWorkflow(mdFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that the target configuration is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_TARGET: \"*\"") {
		t.Errorf("Generated workflow should contain target configuration with asterisk")
	}

	// Verify conditional execution allows any context
	if !strings.Contains(lockContentStr, "if: always()") {
		t.Errorf("Generated workflow should have always() condition for target: *")
	}
}

func TestPushToBranchMissingBranch(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file without branch configuration
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-branch:
    target: "triggering"
---

# Test Push to Branch Missing Branch

This workflow is missing the required branch field.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch-missing-branch.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	// This should fail because branch is required
	err := compiler.CompileWorkflow(mdFile)
	if err == nil {
		t.Fatalf("Expected compilation to fail when branch is missing")
	}

	if !strings.Contains(err.Error(), "missing property 'branch'") {
		t.Errorf("Error should mention missing branch field, got: %v", err)
	}
}

func TestPushToBranchMinimalConfig(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with minimal configuration
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-branch:
    branch: main
---

# Test Push to Branch Minimal

This workflow has minimal push-to-branch configuration.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch-minimal.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	if err := compiler.CompileWorkflow(mdFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that push_to_branch job is generated
	if !strings.Contains(lockContentStr, "push_to_branch:") {
		t.Errorf("Generated workflow should contain push_to_branch job")
	}

	// Verify that the branch configuration is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_BRANCH: \"main\"") {
		t.Errorf("Generated workflow should contain branch configuration")
	}

	// Verify that target defaults to triggering behavior (no explicit target env var)
	if strings.Contains(lockContentStr, "GITHUB_AW_PUSH_TARGET:") {
		t.Errorf("Generated workflow should not contain target configuration when not specified")
	}

	// Verify default conditional execution for pull request context
	if !strings.Contains(lockContentStr, "if: github.event.pull_request.number") {
		t.Errorf("Generated workflow should have default pull request context condition")
	}
}
