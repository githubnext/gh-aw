package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestActivationAndAddReactionJobsPermissions(t *testing.T) {
	// Test that activation job has correct permissions when reaction is configured
	tmpDir, err := os.MkdirTemp("", "permissions-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with reaction configured (reaction step is now in activation job)
	testContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
tools:
  github:
    allowed: [list_issues]
engine: claude
---

# Test Workflow for Task and Add Reaction

This workflow should generate activation job with reaction permissions.

The activation job references text output: "${{ needs.activation.outputs.text }}"
`

	testFile := filepath.Join(tmpDir, "test-permissions.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Calculate the lock file path
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

	// Read the generated lock file
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Test 1: Verify activation job exists and has reaction permissions
	if !strings.Contains(lockContentStr, constants.ActivationJobName+":") {
		t.Error("Expected activation job to be present in generated workflow")
	}

	// Test 2: Verify activation job has checkout step for timestamp check (sparse checkout)
	activationJobSection := extractJobSection(lockContentStr, constants.ActivationJobName)
	if !strings.Contains(activationJobSection, "actions/checkout") {
		t.Error("Activation job should contain actions/checkout step for timestamp check")
	}

	// Verify it's a sparse checkout of workflows directory
	if !strings.Contains(activationJobSection, "sparse-checkout:") {
		t.Error("Activation job checkout should use sparse-checkout")
	}
	if !strings.Contains(activationJobSection, ".github/workflows") {
		t.Error("Activation job checkout should checkout .github/workflows directory")
	}

	// Test 3: Verify activation job has no contents permission explicitly set
	// (sparse checkout works with read-all default permissions)
	if strings.Contains(activationJobSection, "contents:") {
		t.Error("Activation job should not explicitly set contents permission")
	}

	// Test 4: Verify no separate add_reaction job exists
	if strings.Contains(lockContentStr, "add_reaction:") {
		t.Error("Expected no separate add_reaction job - reaction should be in activation job")
	}

	// Test 5: Verify activation job has required permissions for reactions
	if !strings.Contains(activationJobSection, "discussions: write") {
		t.Error("Activation job should have discussions: write permission")
	}
	if !strings.Contains(activationJobSection, "issues: write") {
		t.Error("Activation job should have issues: write permission")
	}
	if !strings.Contains(activationJobSection, "pull-requests: write") {
		t.Error("Activation job should have pull-requests: write permission")
	}

	// Test 6: Verify reaction step is in activation job
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
	}
}
