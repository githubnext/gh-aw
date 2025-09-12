package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushToPullRequestBranchConfigParsing(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with push-to-pr-branch configuration
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
    target: "triggering"
---

# Test Push to PR Branch

This is a test workflow to validate push-to-pr-branch configuration parsing.

Please make changes and push them to the feature branch.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch.md")
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

	// Verify that push_to_pr_branch job is generated
	if !strings.Contains(lockContentStr, "push_to_pr_branch:") {
		t.Errorf("Generated workflow should contain push_to_pr_branch job")
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
	if !strings.Contains(lockContentStr, "needs: test-push-to-pr-branch") {
		t.Errorf("Generated workflow should have dependency on main job")
	}

	// Verify conditional execution for pull request context
	if !strings.Contains(lockContentStr, "if: (github.event.issue.number && github.event.issue.pull_request) || github.event.pull_request") {
		t.Errorf("Generated workflow should have pull request context condition")
	}
}

func TestPushToPullRequestBranchWithTargetAsterisk(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with target: "*"
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
    target: "*"
---

# Test Push to Branch with Target *

This workflow allows pushing to any pull request.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-asterisk.md")
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

func TestPushToPullRequestBranchDefaultBranch(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file without branch configuration
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
    target: "triggering"
---

# Test Push to Branch Default Branch

This workflow uses the default branch value.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-default-branch.md")
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
	lockFile := filepath.Join(tmpDir, "test-push-to-pr-branch-default-branch.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Check that the push_to_pr_branch job is generated
	if !strings.Contains(lockContent, "push_to_pr_branch:") {
		t.Errorf("Expected push_to_pr_branch job to be generated")
	}
}

func TestPushToPullRequestBranchNullConfig(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with null configuration (push-to-pr-branch: with no value)
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch: 
---

# Test Push to Branch Null Config

This workflow uses null configuration which should default to "triggering".
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-null-config.md")
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
	lockFile := filepath.Join(tmpDir, "test-push-to-pr-branch-null-config.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Check that the push_to_pr_branch job is generated
	if !strings.Contains(lockContent, "push_to_pr_branch:") {
		t.Errorf("Expected push_to_pr_branch job to be generated")
	}

	// Check that no target is set (should use default)
	if strings.Contains(lockContent, "GITHUB_AW_PUSH_TARGET:") {
		t.Errorf("Expected no target to be set when using null config, %s", lockContent)
	}
}

func TestPushToPullRequestBranchMinimalConfig(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with minimal configuration
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
---

# Test Push to Branch Minimal

This workflow has minimal push-to-pr-branch configuration.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-minimal.md")
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

	// Verify that push_to_pr_branch job is generated
	if !strings.Contains(lockContentStr, "push_to_pr_branch:") {
		t.Errorf("Generated workflow should contain push_to_pr_branch job")
	}

	// Verify that target defaults to triggering behavior (no explicit target env var)
	if strings.Contains(lockContentStr, "GITHUB_AW_PUSH_TARGET:") {
		t.Errorf("Generated workflow should not contain target configuration when not specified")
	}

	// Verify default conditional execution for pull request context
	if !strings.Contains(lockContentStr, "if: (github.event.issue.number && github.event.issue.pull_request) || github.event.pull_request") {
		t.Errorf("Generated workflow should have default pull request context condition")
	}
}

func TestPushToPullRequestBranchWithIfNoChangesError(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with if-no-changes: error
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
    target: "triggering"
    if-no-changes: "error"
---

# Test Push to Branch with if-no-changes: error

This workflow fails when there are no changes.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-error.md")
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

	// Verify that if-no-changes configuration is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_IF_NO_CHANGES: \"error\"") {
		t.Errorf("Generated workflow should contain if-no-changes configuration")
	}
}

func TestPushToPullRequestBranchWithIfNoChangesIgnore(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with if-no-changes: ignore
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
    if-no-changes: "ignore"
---

# Test Push to Branch with if-no-changes: ignore

This workflow ignores when there are no changes.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-ignore.md")
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

	// Verify that if-no-changes configuration is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_IF_NO_CHANGES: \"ignore\"") {
		t.Errorf("Generated workflow should contain if-no-changes ignore configuration")
	}
}

func TestPushToPullRequestBranchDefaultIfNoChanges(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file without if-no-changes (should default to "warn")
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
---

# Test Push to Branch Default if-no-changes

This workflow uses default if-no-changes behavior.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-default-if-no-changes.md")
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

	// Verify that default if-no-changes configuration ("warn") is passed correctly
	if !strings.Contains(lockContentStr, "GITHUB_AW_PUSH_IF_NO_CHANGES: \"warn\"") {
		t.Errorf("Generated workflow should contain default if-no-changes configuration (warn)")
	}
}

func TestPushToPullRequestBranchExplicitTriggering(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with explicit "triggering" branch
	testMarkdown := `---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pr-branch:
    target: "triggering"
---

# Test Push to Branch Explicit Triggering

This workflow explicitly sets branch to "triggering".
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-push-to-pr-branch-explicit-triggering.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Create compiler and compile the workflow
	compiler := NewCompiler(false, "", "test")

	if err := compiler.CompileWorkflow(mdFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-push-to-pr-branch-explicit-triggering.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that push_to_pr_branch job is generated
	if !strings.Contains(lockContentStr, "push_to_pr_branch:") {
		t.Errorf("Generated workflow should contain push_to_pr_branch job")
	}

	// Verify that target configuration is included
	if !strings.Contains(lockContentStr, `GITHUB_AW_PUSH_TARGET: "triggering"`) {
		t.Errorf("Generated workflow should contain target configuration")
	}
}
