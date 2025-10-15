package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// Test container names for validation tests - these should not exist
	testInvalidContainer1 = "nonexistent-invalid-image-for-testing-12345"
	testInvalidContainer2 = "nonexistent-invalid-image-for-testing-67890"
)

// TestCompileWithInvalidContainerImage verifies that container image validation
// failures produce warnings instead of errors when validation is enabled
func TestCompileWithInvalidContainerImage(t *testing.T) {
	// Skip test if docker is not available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping test")
	}

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "container-validation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a workflow with an invalid container image
	workflowContent := `---
on: push
engine: claude
mcp-servers:
  test-tool:
    type: stdio
    container: ` + testInvalidContainer1 + `
    allowed: ["test_function"]
---

# Test Workflow

This workflow has an invalid container image.
`

	workflowFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create compiler with validation enabled (default behavior)
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(false) // Ensure validation is enabled

	// Compile the workflow - this should succeed with a warning, not fail with an error
	err = compiler.CompileWorkflow(workflowFile)

	// The compilation should succeed (no error returned) despite invalid container
	if err != nil {
		t.Errorf("compilation should succeed with warning, but got error: %v", err)
	}

	// Verify the lock file was created
	lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("lock file should be created despite container validation warning")
	}
}

// TestCompileWithInvalidContainerValidationDisabled verifies that when validation
// is disabled, no warning is produced for invalid container images
func TestCompileWithInvalidContainerValidationDisabled(t *testing.T) {
	// Skip test if docker is not available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping test")
	}

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "container-validation-disabled-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a workflow with an invalid container image
	workflowContent := `---
on: push
engine: claude
mcp-servers:
  test-tool:
    type: stdio
    container: ` + testInvalidContainer2 + `
    allowed: ["test_function"]
---

# Test Workflow

This workflow has an invalid container image.
`

	workflowFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create compiler with validation disabled
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true) // Disable validation

	// Compile the workflow - this should succeed without validation
	err = compiler.CompileWorkflow(workflowFile)

	// The compilation should succeed (no error returned)
	if err != nil {
		t.Errorf("compilation should succeed when validation disabled, but got error: %v", err)
	}

	// Verify the lock file was created
	lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("lock file should be created when validation is disabled")
	}
}
