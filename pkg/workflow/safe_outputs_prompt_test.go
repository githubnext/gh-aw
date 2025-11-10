package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeOutputsPromptIncludedWhenEnabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-safe-outputs-prompt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with safe-outputs enabled
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: issues
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  create-issue:
    labels: [automation]
---

# Test Workflow with Safe Outputs

This is a test workflow with safe-outputs enabled.
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify safe outputs prompt step is created
	if !strings.Contains(lockStr, "- name: Append safe outputs instructions to prompt") {
		t.Error("Expected 'Append safe outputs instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the instruction text mentions creating issues
	if !strings.Contains(lockStr, "Creating an Issue") {
		t.Error("Expected 'Creating an Issue' reference in generated workflow")
	}

	// Test 3: Verify the instruction text contains JSON output format
	if !strings.Contains(lockStr, "create_issue") {
		t.Error("Expected 'create_issue' output type reference in generated workflow")
	}

	t.Logf("Successfully verified safe outputs instructions are included in generated workflow")
}

func TestSafeOutputsPromptNotIncludedWhenDisabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-no-safe-outputs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow WITHOUT safe-outputs
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  github:
---

# Test Workflow without Safe Outputs

This is a test workflow without safe-outputs.
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test: Verify safe outputs prompt step is NOT created
	if strings.Contains(lockStr, "- name: Append safe outputs instructions to prompt") {
		t.Error("Did not expect 'Append safe outputs instructions to prompt' step in workflow without safe-outputs")
	}

	t.Logf("Successfully verified safe outputs instructions are NOT included when safe-outputs is disabled")
}

func TestSafeOutputsPromptMultipleOutputTypes(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-multi-safe-outputs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with multiple safe-output types
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: issues
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  create-issue:
    labels: [automation]
  add-comment:
    max: 3
  create-pull-request:
    draft: true
---

# Test Workflow with Multiple Safe Outputs

This is a test workflow with multiple safe-output types.
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify safe outputs prompt step is created
	if !strings.Contains(lockStr, "- name: Append safe outputs instructions to prompt") {
		t.Error("Expected 'Append safe outputs instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify all output types are mentioned
	if !strings.Contains(lockStr, "Creating an Issue") {
		t.Error("Expected 'Creating an Issue' reference in generated workflow")
	}

	if !strings.Contains(lockStr, "Adding a Comment") {
		t.Error("Expected 'Adding a Comment' reference in generated workflow")
	}

	if !strings.Contains(lockStr, "Creating a Pull Request") {
		t.Error("Expected 'Creating a Pull Request' reference in generated workflow")
	}

	t.Logf("Successfully verified safe outputs instructions handle multiple output types")
}
