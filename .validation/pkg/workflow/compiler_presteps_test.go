//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestPreStepsGeneration verifies that pre-steps are emitted before checkout and all
// other built-in steps in the agent job.
func TestPreStepsGeneration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-steps-test")

	testContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    allowed: [list_issues]
pre-steps:
  - name: Mint short-lived token
    id: mint
    uses: some-org/token-minting-action@a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
    with:
      scope: target-org/target-repo
steps:
  - name: Custom Setup Step
    run: echo "Custom setup"
post-steps:
  - name: Post AI Step
    run: echo "This runs after AI"
engine: claude
strict: false
---

# Test Pre-Steps Workflow

This workflow tests the pre-steps functionality.
`

	testFile := filepath.Join(tmpDir, "test-pre-steps.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow with pre-steps: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test-pre-steps.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify all three step types appear (check name value, not "- name:" prefix
	// since steps with an id field have id: first in the YAML output)
	if !strings.Contains(lockContent, "name: Mint short-lived token") {
		t.Error("Expected pre-step 'Mint short-lived token' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "name: Custom Setup Step") {
		t.Error("Expected custom step 'Custom Setup Step' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "name: Post AI Step") {
		t.Error("Expected post-step 'Post AI Step' to be in generated workflow")
	}

	// Pre-steps must appear before checkout, custom steps, and AI execution
	preStepIndex := indexInNonCommentLines(lockContent, "name: Mint short-lived token")
	checkoutIndex := indexInNonCommentLines(lockContent, "- name: Checkout repository")
	customStepIndex := indexInNonCommentLines(lockContent, "- name: Custom Setup Step")
	aiStepIndex := indexInNonCommentLines(lockContent, "- name: Execute Claude Code CLI")
	postStepIndex := indexInNonCommentLines(lockContent, "- name: Post AI Step")

	if preStepIndex == -1 {
		t.Fatal("Could not find pre-step in generated workflow")
	}
	if checkoutIndex == -1 {
		t.Fatal("Could not find checkout step in generated workflow")
	}
	if customStepIndex == -1 {
		t.Fatal("Could not find custom step in generated workflow")
	}
	if aiStepIndex == -1 {
		t.Fatal("Could not find AI execution step in generated workflow")
	}
	if postStepIndex == -1 {
		t.Fatal("Could not find post-step in generated workflow")
	}

	if preStepIndex >= checkoutIndex {
		t.Errorf("Pre-step (%d) should appear before checkout step (%d)", preStepIndex, checkoutIndex)
	}
	if preStepIndex >= customStepIndex {
		t.Errorf("Pre-step (%d) should appear before custom step (%d)", preStepIndex, customStepIndex)
	}
	if preStepIndex >= aiStepIndex {
		t.Errorf("Pre-step (%d) should appear before AI execution step (%d)", preStepIndex, aiStepIndex)
	}
	if postStepIndex <= aiStepIndex {
		t.Errorf("Post-step (%d) should appear after AI execution step (%d)", postStepIndex, aiStepIndex)
	}

	t.Logf("Step order verified: pre-step(%d) < checkout(%d) < custom(%d) < AI(%d) < post(%d)",
		preStepIndex, checkoutIndex, customStepIndex, aiStepIndex, postStepIndex)
}

// TestPreStepsTokenAvailableForCheckout verifies that a token minted in a pre-step
// can be referenced in checkout.token via a steps expression, avoiding the cross-job
// masked-value issue.
func TestPreStepsTokenAvailableForCheckout(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-steps-token-test")

	testContent := `---
on: workflow_dispatch
permissions:
  contents: read
  id-token: write
pre-steps:
  - name: Mint token
    id: mint
    uses: some-org/token-action@b1c2d3e4f5a6b1c2d3e4f5a6b1c2d3e4f5a6b1c2
    with:
      scope: target-org/target-repo
checkout:
  - repository: target-org/target-repo
    path: target
    token: ${{ steps.mint.outputs.token }}
    current: false
  - path: .
engine: claude
strict: false
---

Read a file from the checked-out repo.
`

	testFile := filepath.Join(tmpDir, "test-pre-steps-token.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test-pre-steps-token.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// The minting step must appear in the agent job
	agentJobSection := extractJobSection(lockContent, "agent")
	if agentJobSection == "" {
		t.Fatal("Agent job section not found in generated workflow")
	}

	if !strings.Contains(agentJobSection, "name: Mint token") {
		t.Error("Expected pre-step 'Mint token' to be in the agent job")
	}

	// The token reference must appear in the checkout step
	if !strings.Contains(agentJobSection, "steps.mint.outputs.token") {
		t.Error("Expected steps.mint.outputs.token reference in agent job checkout step")
	}

	// The pre-step must appear before the checkout step
	mintIndex := indexInNonCommentLines(agentJobSection, "name: Mint token")
	checkoutIndex := indexInNonCommentLines(agentJobSection, "- name: Checkout target-org/target-repo into target")
	if mintIndex == -1 {
		t.Fatal("Could not find mint step in agent job")
	}
	if checkoutIndex == -1 {
		t.Fatal("Could not find cross-repo checkout step in agent job")
	}
	if mintIndex >= checkoutIndex {
		t.Errorf("Pre-step mint (%d) should appear before cross-repo checkout (%d)", mintIndex, checkoutIndex)
	}
}

// TestPreStepsOnly verifies that a workflow with only pre-steps (no custom steps or post-steps)
// compiles correctly.
func TestPreStepsOnly(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-steps-only-test")

	testContent := `---
on: issues
permissions:
  contents: read
  issues: read
pre-steps:
  - name: Only Pre Step
    run: echo "This runs before checkout"
engine: claude
strict: false
---

# Test Pre-Steps Only Workflow

This workflow tests pre-steps without custom steps or post-steps.
`

	testFile := filepath.Join(tmpDir, "test-pre-steps-only.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow with pre-steps only: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test-pre-steps-only.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	if !strings.Contains(lockContent, "- name: Only Pre Step") {
		t.Error("Expected pre-step 'Only Pre Step' to be in generated workflow")
	}

	// Default checkout must still be present and after the pre-step
	preStepIndex := indexInNonCommentLines(lockContent, "- name: Only Pre Step")
	checkoutIndex := indexInNonCommentLines(lockContent, "- name: Checkout repository")
	aiStepIndex := indexInNonCommentLines(lockContent, "- name: Execute Claude Code CLI")

	if preStepIndex == -1 {
		t.Fatal("Could not find pre-step in generated workflow")
	}
	if checkoutIndex == -1 {
		t.Error("Expected default checkout step to still be present")
	}
	if aiStepIndex == -1 {
		t.Fatal("Could not find AI execution step in generated workflow")
	}

	if checkoutIndex != -1 && preStepIndex >= checkoutIndex {
		t.Errorf("Pre-step (%d) should appear before checkout step (%d)", preStepIndex, checkoutIndex)
	}
	if preStepIndex >= aiStepIndex {
		t.Errorf("Pre-step (%d) should appear before AI execution step (%d)", preStepIndex, aiStepIndex)
	}
}

// TestPreStepsSecretsValidation verifies that secrets in pre-steps trigger the same
// strict-mode error and non-strict warning as secrets in steps and post-steps.
func TestPreStepsSecretsValidation(t *testing.T) {
	compiler := NewCompiler()
	compiler.strictMode = true

	frontmatter := map[string]any{
		"pre-steps": []any{
			map[string]any{
				"name": "Use secret in pre-step",
				"run":  "echo ${{ secrets.MY_SECRET }}",
			},
		},
	}

	err := compiler.validateStepsSecrets(frontmatter)
	if err == nil {
		t.Error("Expected strict-mode error for secrets in pre-steps but got nil")
	}
	if !strings.Contains(err.Error(), "pre-steps") {
		t.Errorf("Expected error to mention 'pre-steps', got: %v", err)
	}
}
