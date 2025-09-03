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

func TestPushToBranchDefaultBranch(t *testing.T) {
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

# Test Push to Branch Default Branch

This workflow uses the default branch value.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch-default-branch.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	// This should succeed and use default branch "triggering"
	err := compiler.CompileWorkflow(mdFile)
	if err != nil {
		t.Fatalf("Expected compilation to succeed with default branch, got error: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-push-to-branch-default-branch.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Check that the default branch "triggering" is used
	if !strings.Contains(lockContent, `GITHUB_AW_PUSH_BRANCH: "triggering"`) {
		t.Errorf("Expected default branch 'triggering' to be set in environment variables")
	}

	// Check that the push_to_branch job is generated
	if !strings.Contains(lockContent, "push_to_branch:") {
		t.Errorf("Expected push_to_branch job to be generated")
	}
}

func TestPushToBranchNullConfig(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with null configuration (push-to-branch: with no value)
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-branch: 
---

# Test Push to Branch Null Config

This workflow uses null configuration which should default to "triggering".
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch-null-config.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	// This should succeed and use default branch "triggering"
	err := compiler.CompileWorkflow(mdFile)
	if err != nil {
		t.Fatalf("Expected compilation to succeed with null config, got error: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-push-to-branch-null-config.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Check that the default branch "triggering" is used
	if !strings.Contains(lockContent, `GITHUB_AW_PUSH_BRANCH: "triggering"`) {
		t.Errorf("Expected default branch 'triggering' to be set in environment variables")
	}

	// Check that the push_to_branch job is generated
	if !strings.Contains(lockContent, "push_to_branch:") {
		t.Errorf("Expected push_to_branch job to be generated")
	}

	// Check that no target is set (should use default)
	if strings.Contains(lockContent, "GITHUB_AW_PUSH_TARGET:") {
		t.Errorf("Expected no target to be set when using null config")
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

func TestPushToBranchExplicitTriggering(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with explicit "triggering" branch
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-branch:
    branch: "triggering"
    target: "triggering"
---

# Test Push to Branch Explicit Triggering

This workflow explicitly sets branch to "triggering".
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-branch-explicit-triggering.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	if err := compiler.CompileWorkflow(mdFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-push-to-branch-explicit-triggering.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that push_to_branch job is generated
	if !strings.Contains(lockContentStr, "push_to_branch:") {
		t.Errorf("Generated workflow should contain push_to_branch job")
	}

	// Verify that the explicit "triggering" branch configuration is passed correctly
	if !strings.Contains(lockContentStr, `GITHUB_AW_PUSH_BRANCH: "triggering"`) {
		t.Errorf("Generated workflow should contain explicit triggering branch configuration")
	}

	// Verify that target configuration is included
	if !strings.Contains(lockContentStr, `GITHUB_AW_PUSH_TARGET: "triggering"`) {
		t.Errorf("Generated workflow should contain target configuration")
	}
}
