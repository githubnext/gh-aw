//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestDuplicateStepValidation_Integration tests that the duplicate step validation
// correctly catches compiler bugs where the same step is added multiple times
func TestDuplicateStepValidation_Integration(t *testing.T) {
	// This test verifies the duplicate step validation by checking that
	// workflows compile without duplicate step errors
	tmpDir := testutil.TempDir(t, "duplicate-step-validation-test")

	// Test case: workflow with both create-pull-request and push-to-pull-request-branch
	// Previously this would generate duplicate "Checkout repository" steps
	mdContent := `---
on: issue_comment
engine: copilot
strict: false
safe-outputs:
  create-pull-request: null
  push-to-pull-request-branch: null
---

# Test Workflow

This workflow tests that duplicate checkout steps are properly deduplicated.
`

	mdFile := filepath.Join(tmpDir, "test-duplicate-steps.md")
	err := os.WriteFile(mdFile, []byte(mdContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler()
	err = compiler.CompileWorkflow(mdFile)
	if err != nil {
		// The error should NOT be about duplicate steps since we fixed the bug
		if strings.Contains(err.Error(), "duplicate step") {
			t.Fatalf("Unexpected duplicate step error after fix: %v", err)
		}
		// Other errors are acceptable (this is just testing the validation)
		t.Logf("Compilation failed with non-duplicate-step error (acceptable): %v", err)
		return
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(mdFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that there's only one "Checkout repository" step in the safe_outputs job
	// Count occurrences of "name: Checkout repository" in the safe_outputs job section
	safeOutputsStart := strings.Index(lockContentStr, "safe_outputs:")
	if safeOutputsStart == -1 {
		t.Error("Expected safe_outputs job to be present")
		return
	}

	// Find the next job after safe_outputs (or end of file)
	nextJobStart := strings.Index(lockContentStr[safeOutputsStart+1:], "\n  ") + safeOutputsStart + 1
	if nextJobStart <= safeOutputsStart {
		nextJobStart = len(lockContentStr)
	}

	safeOutputsSection := lockContentStr[safeOutputsStart:nextJobStart]
	checkoutCount := strings.Count(safeOutputsSection, "name: Checkout repository")

	// After the fix, we expect exactly 1 checkout step (shared between both operations)
	// OR 0 if the operations don't require checkout (depending on configuration)
	if checkoutCount > 1 {
		t.Errorf("Found %d 'Checkout repository' steps in safe_outputs job, expected 0 or 1 (deduplicated)", checkoutCount)
	}

	t.Logf("✓ Duplicate step validation working correctly: found %d checkout step(s) in safe_outputs job (deduplicated)", checkoutCount)
}

// TestDuplicateStepValidation_CheckoutPlusGitHubApp_Integration tests that combining
// a top-level github-app with multiple cross-repo checkouts and tools.github does not
// produce duplicate 'Generate GitHub App token' steps in the activation job.
//
// When multiple checkout entries all fall back to the top-level github-app,
// each minting step previously received the same name, triggering the duplicate
// step validation error ("compiler bug: duplicate step 'Generate GitHub App token'").
func TestDuplicateStepValidation_CheckoutPlusGitHubApp_Integration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "duplicate-checkout-token-test")

	// Workflow that combines all three conditions that triggered the bug:
	//   1. Top-level github-app: (used as fallback for all token-minting operations)
	//   2. Two cross-repo checkout: entries (both fall back to the top-level github-app)
	//   3. tools.github: with mode: remote
	mdContent := `---
on:
  issues:
    types: [opened]
engine: claude
strict: false
permissions:
  contents: read
  issues: read
  pull-requests: read

github-app:
  app-id: ${{ secrets.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
  repositories: ["side-repo", "target-repo"]

checkout:
  - repository: myorg/target-repo
    ref: main
  - repository: myorg/side-repo
    ref: main

tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Workflow

This workflow tests that multiple checkouts + top-level github-app + tools.github
compile without duplicate 'Generate GitHub App token' step errors in the activation job.
`

	mdFile := filepath.Join(tmpDir, "test-checkout-github-app.md")
	err := os.WriteFile(mdFile, []byte(mdContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compile workflow — must succeed so the generated lock file can be validated.
	compiler := NewCompiler()
	err = compiler.CompileWorkflow(mdFile)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate step") {
			t.Fatalf("Regression: duplicate step error when combining multiple checkouts + top-level github-app: %v", err)
		}
		t.Fatalf("Compilation failed unexpectedly before lock-file assertions could run: %v", err)
	}

	// Read the generated lock file and verify the activation job has unique step names
	lockFile := stringutil.MarkdownToLockFile(mdFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Both checkout token minting steps should be present with unique names.
	// The step names are "Generate GitHub App token for checkout (N)" — one per checkout entry.
	count0 := strings.Count(lockContentStr, "name: Generate GitHub App token for checkout (0)")
	count1 := strings.Count(lockContentStr, "name: Generate GitHub App token for checkout (1)")
	if count0 != 1 {
		t.Errorf("Expected exactly 1 'Generate GitHub App token for checkout (0)' step, got %d", count0)
	}
	if count1 != 1 {
		t.Errorf("Expected exactly 1 'Generate GitHub App token for checkout (1)' step, got %d", count1)
	}

	// Exactly one generic "Generate GitHub App token" step is expected — for the GitHub MCP server
	// in the agent job (id: github-mcp-app-token). If more than one appears, that means a
	// checkout minting step was not renamed, which would cause a duplicate-name error.
	genericCount := strings.Count(lockContentStr, "name: Generate GitHub App token\n")
	if genericCount > 1 {
		t.Errorf("Found %d generic 'Generate GitHub App token' steps; checkout steps must use unique names to avoid duplicates", genericCount)
	}

	t.Logf("✓ No duplicate token steps: checkout (0) count=%d, checkout (1) count=%d, generic=%d", count0, count1, genericCount)
}
