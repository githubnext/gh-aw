package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestActivationAndAddReactionJobsPermissions(t *testing.T) {
	// Test that activation job has correct permissions when reaction is configured
	tmpDir := testutil.TempDir(t, "permissions-test")

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
strict: false
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
	if err := compiler.CompileWorkflow(testFile); err != nil {
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

	// Test 2: Verify activation job does NOT have checkout step (uses GitHub API instead)
	activationJobSection := extractJobSection(lockContentStr, constants.ActivationJobName)
	if strings.Contains(activationJobSection, "actions/checkout") {
		t.Error("Activation job should NOT contain actions/checkout step - should use GitHub API instead")
	}

	// Verify it does NOT use sparse checkout
	if strings.Contains(activationJobSection, "sparse-checkout:") {
		t.Error("Activation job should NOT use sparse-checkout - uses GitHub API instead")
	}

	// Verify it uses GitHub API for timestamp check
	if !strings.Contains(activationJobSection, "github.rest.repos.listCommits") {
		t.Error("Activation job should use GitHub API (github.rest.repos.listCommits) for timestamp check")
	}

	// Test 3: Verify activation job has contents: read permission for GitHub API access
	if !strings.Contains(activationJobSection, "contents: read") {
		t.Error("Activation job should have contents: read permission for GitHub API access")
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
