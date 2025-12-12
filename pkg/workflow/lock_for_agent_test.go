package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestLockForAgentWorkflow(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "lock-for-agent-test")

	// Create a test markdown file with lock-for-agent enabled
	testContent := `---
on:
  issues:
    types: [opened]
    lock-for-agent: true
  reaction: eyes
engine: copilot
safe-outputs:
  add-comment: {}
---

# Lock For Agent Test

Test workflow with lock-for-agent enabled.
`

	testFile := filepath.Join(tmpDir, "test-lock-for-agent.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify lock-for-agent field is parsed correctly
	if !workflowData.LockForAgent {
		t.Error("Expected LockForAgent to be true")
	}

	// Generate YAML and verify it contains lock/unlock steps
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for lock-specific content in generated YAML
	expectedStrings := []string{
		"Lock issue for agent workflow",
		"Unlock issue after agent workflow",
		"GH_AW_LOCK_FOR_AGENT: \"true\"",
		"lockForAgent && eventName === \"issues\"",
		"This issue has been locked while the workflow is running",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify lock step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Lock issue for agent workflow") {
		t.Error("Activation job should contain the lock step")
	}

	// Verify unlock step is in conclusion job
	conclusionJobSection := extractJobSection(yamlContent, "conclusion")
	if !strings.Contains(conclusionJobSection, "Unlock issue after agent workflow") {
		t.Error("Conclusion job should contain the unlock step")
	}

	// Verify unlock step has always() condition
	if !strings.Contains(conclusionJobSection, "if: (always())") {
		t.Error("Unlock step should have always() condition")
	}
}

func TestLockForAgentWithoutReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "lock-for-agent-no-reaction-test")

	// Create a test markdown file with lock-for-agent but no reaction
	testContent := `---
on:
  issues:
    types: [opened]
    lock-for-agent: true
engine: copilot
safe-outputs:
  add-comment: {}
---

# Lock For Agent Test Without Reaction

Test workflow with lock-for-agent but no reaction.
`

	testFile := filepath.Join(tmpDir, "test-lock-no-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify lock-for-agent field is parsed correctly
	if !workflowData.LockForAgent {
		t.Error("Expected LockForAgent to be true")
	}

	// Generate YAML and verify it contains lock/unlock steps
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Lock and unlock steps should still be present
	if !strings.Contains(yamlContent, "Lock issue for agent workflow") {
		t.Error("Generated YAML should contain lock step even without reaction")
	}

	if !strings.Contains(yamlContent, "Unlock issue after agent workflow") {
		t.Error("Generated YAML should contain unlock step even without reaction")
	}

	// The GH_AW_LOCK_FOR_AGENT env var should not be set (no reaction step to set it)
	if strings.Contains(yamlContent, "GH_AW_LOCK_FOR_AGENT: \"true\"") {
		t.Error("Generated YAML should not set GH_AW_LOCK_FOR_AGENT env var without reaction step")
	}

	// Verify activation job has issues: write permission for locking
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "issues: write") {
		t.Error("Activation job should have issues: write permission when lock-for-agent is enabled")
	}

	// Verify conclusion job has issues: write permission for unlocking
	conclusionJobSection := extractJobSection(yamlContent, "conclusion")
	if !strings.Contains(conclusionJobSection, "issues: write") {
		t.Error("Conclusion job should have issues: write permission when lock-for-agent is enabled")
	}
}

func TestLockForAgentDisabled(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "lock-for-agent-disabled-test")

	// Create a test markdown file without lock-for-agent
	testContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
engine: copilot
safe-outputs:
  add-comment: {}
---

# Test Without Lock For Agent

Test workflow without lock-for-agent.
`

	testFile := filepath.Join(tmpDir, "test-no-lock.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify lock-for-agent field is false by default
	if workflowData.LockForAgent {
		t.Error("Expected LockForAgent to be false by default")
	}

	// Generate YAML and verify it does not contain lock/unlock steps
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Lock and unlock steps should not be present
	if strings.Contains(yamlContent, "Lock issue for agent workflow") {
		t.Error("Generated YAML should not contain lock step when lock-for-agent is disabled")
	}

	if strings.Contains(yamlContent, "Unlock issue after agent workflow") {
		t.Error("Generated YAML should not contain unlock step when lock-for-agent is disabled")
	}

	// The JavaScript code checking for GH_AW_LOCK_FOR_AGENT will still be in the script,
	// but the environment variable itself should not be set
	if strings.Contains(yamlContent, "GH_AW_LOCK_FOR_AGENT: \"true\"") {
		t.Error("Generated YAML should not set GH_AW_LOCK_FOR_AGENT env var when lock-for-agent is disabled")
	}

	// Verify activation job has issues: write permission due to reaction (not lock-for-agent)
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "issues: write") {
		t.Error("Activation job should have issues: write permission when reaction is enabled")
	}
}

func TestLockForAgentDisabledWithoutReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "lock-disabled-no-reaction-test")

	// Create a test markdown file without lock-for-agent and without reaction
	testContent := `---
on:
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  add-comment: {}
---

# Test Without Lock For Agent and Without Reaction

Test workflow without lock-for-agent and without reaction.
`

	testFile := filepath.Join(tmpDir, "test-no-lock-no-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify lock-for-agent field is false by default
	if workflowData.LockForAgent {
		t.Error("Expected LockForAgent to be false by default")
	}

	// Generate YAML and verify it does not contain lock/unlock steps
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Lock and unlock steps should not be present
	if strings.Contains(yamlContent, "Lock issue for agent workflow") {
		t.Error("Generated YAML should not contain lock step when lock-for-agent is disabled")
	}

	if strings.Contains(yamlContent, "Unlock issue after agent workflow") {
		t.Error("Generated YAML should not contain unlock step when lock-for-agent is disabled")
	}

	// Verify activation job does NOT have issues: write permission (no reaction and no lock-for-agent)
	activationJobSection := extractJobSection(yamlContent, "activation")
	if strings.Contains(activationJobSection, "issues: write") {
		t.Error("Activation job should NOT have issues: write permission when lock-for-agent is disabled and no reaction is configured")
	}
}

func TestLockForAgentOnPullRequest(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "lock-for-agent-pr-test")

	// Create a test markdown file with pull_request event (should not cause errors)
	testContent := `---
on:
  pull_request:
    types: [opened]
  reaction: eyes
engine: copilot
safe-outputs:
  add-comment: {}
---

# Test Lock For Agent with PR

Test that lock-for-agent on issues doesn't break PR workflows.
`

	testFile := filepath.Join(tmpDir, "test-pr.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Generate YAML - should succeed without errors
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Lock steps should not be present for PR event (no lock-for-agent in on.pull_request)
	if strings.Contains(yamlContent, "Lock issue for agent workflow") {
		t.Error("Generated YAML should not contain lock step for pull_request event")
	}
}
