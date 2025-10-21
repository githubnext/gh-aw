package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStepsFormat_ObjectWithPositions(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "new-steps-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test new object format with multiple positions
	testContent := `---
on: push
permissions:
  contents: read
tools:
  edit:
steps:
  pre:
    - name: Pre Step
      run: echo "runs before checkout"
  pre-agent:
    - name: Pre Agent Step
      run: echo "runs after setup, before agent"
  post-agent:
    - name: Post Agent Step
      run: echo "runs after agent"
  post:
    - name: Final Step
      run: echo "runs at the very end"
engine: claude
---

# Test New Steps Format

This tests the new object-based steps format.
`

	testFile := filepath.Join(tmpDir, "test-new-steps.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with new steps format: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-new-steps.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify all steps are present
	if !strings.Contains(lockContent, "- name: Pre Step") {
		t.Error("Expected 'Pre Step' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Pre Agent Step") {
		t.Error("Expected 'Pre Agent Step' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Post Agent Step") {
		t.Error("Expected 'Post Agent Step' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Final Step") {
		t.Error("Expected 'Final Step' to be in generated workflow")
	}

	// Verify step order
	preStepPos := strings.Index(lockContent, "- name: Pre Step")
	checkoutPos := strings.Index(lockContent, "- name: Checkout repository")
	preAgentPos := strings.Index(lockContent, "- name: Pre Agent Step")
	agentPos := strings.Index(lockContent, "- name: Execute Claude Code CLI")
	postAgentPos := strings.Index(lockContent, "- name: Post Agent Step")
	finalStepPos := strings.Index(lockContent, "- name: Final Step")

	if preStepPos == -1 || checkoutPos == -1 || preAgentPos == -1 || agentPos == -1 || postAgentPos == -1 || finalStepPos == -1 {
		t.Fatal("Could not find all expected steps in generated workflow")
	}

	// Verify correct order: Pre < Checkout < Pre-Agent < Agent < Post-Agent < Final
	if preStepPos >= checkoutPos {
		t.Errorf("Pre step should come before checkout: pre=%d, checkout=%d", preStepPos, checkoutPos)
	}
	if checkoutPos >= preAgentPos {
		t.Errorf("Checkout should come before pre-agent: checkout=%d, pre-agent=%d", checkoutPos, preAgentPos)
	}
	if preAgentPos >= agentPos {
		t.Errorf("Pre-agent step should come before agent execution: pre-agent=%d, agent=%d", preAgentPos, agentPos)
	}
	if agentPos >= postAgentPos {
		t.Errorf("Agent execution should come before post-agent step: agent=%d, post-agent=%d", agentPos, postAgentPos)
	}
	if postAgentPos >= finalStepPos {
		t.Errorf("Post-agent step should come before final step: post-agent=%d, final=%d", postAgentPos, finalStepPos)
	}

	t.Log("Step order verified: Pre < Checkout < Pre-Agent < Agent < Post-Agent < Final")
}

func TestNewStepsFormat_PostAgentField(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "post-agent-field-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test post-agent steps within steps object
	testContent := `---
on: push
permissions:
  contents: read
tools:
  edit:
steps:
  post-agent:
    - name: Post Agent Step
      run: echo "runs after agent"
    - name: Another Post Agent Step
      run: echo "also runs after agent"
engine: claude
---

# Test Post-Agent Steps

This tests the post-agent steps within the steps object.
`

	testFile := filepath.Join(tmpDir, "test-post-agent.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with post-agent field: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-post-agent.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify post-agent steps are present
	if !strings.Contains(lockContent, "- name: Post Agent Step") {
		t.Error("Expected 'Post Agent Step' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Another Post Agent Step") {
		t.Error("Expected 'Another Post Agent Step' to be in generated workflow")
	}

	// Verify they come after agent execution
	agentPos := strings.Index(lockContent, "- name: Execute Claude Code CLI")
	postAgentPos := strings.Index(lockContent, "- name: Post Agent Step")

	if agentPos == -1 || postAgentPos == -1 {
		t.Fatal("Could not find expected elements in generated workflow")
	}

	if agentPos >= postAgentPos {
		t.Errorf("Post-agent steps should come after agent execution: agent=%d, post-agent=%d", agentPos, postAgentPos)
	}

	t.Log("Post-agent field verified successfully")
}

func TestLegacyStepsFormat_BackwardCompatibility(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "legacy-steps-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test legacy array format still works for pre-agent steps
	testContent := `---
on: push
permissions:
  contents: read
tools:
  edit:
steps:
  pre-agent:
    - name: Legacy Step 1
      run: echo "legacy step 1"
    - name: Legacy Step 2
      run: echo "legacy step 2"
  post-agent:
    - name: Legacy Post Agent Step
      run: echo "legacy post-agent step"
engine: claude
---

# Test Legacy Format

This tests backward compatibility with the legacy array format converted to object format.
`

	testFile := filepath.Join(tmpDir, "test-legacy.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with legacy format: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-legacy.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify steps are present
	if !strings.Contains(lockContent, "- name: Legacy Step 1") {
		t.Error("Expected 'Legacy Step 1' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Legacy Step 2") {
		t.Error("Expected 'Legacy Step 2' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Legacy Post Agent Step") {
		t.Error("Expected 'Legacy Post Agent Step' to be in generated workflow")
	}

	t.Log("Legacy format backward compatibility verified")
}
