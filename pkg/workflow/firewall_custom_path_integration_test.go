package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFirewallCustomPathIntegration tests that workflows with custom firewall paths compile correctly
func TestFirewallCustomPathIntegration(t *testing.T) {
	t.Run("workflow with absolute path compiles correctly", func(t *testing.T) {
		// Create a temporary test directory
		testDir := t.TempDir()
		workflowsDir := filepath.Join(testDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create workflows directory: %v", err)
		}

		// Create a test workflow with custom AWF path
		workflowContent := `---
name: Test Custom AWF Path (Absolute)
on:
  workflow_dispatch:
engine: copilot
tools:
  github:
    mode: remote
network:
  allowed:
    - defaults
    - "*.example.com"
  firewall:
    path: /custom/path/to/awf
    log-level: debug
permissions:
  issues: read
  pull-requests: read
---
Test workflow with custom AWF binary path.
`
		workflowPath := filepath.Join(workflowsDir, "test-custom-awf-path.md")
		err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test-custom-awf-path")
		compiler.SetSkipValidation(true)
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the compiled lock file
		lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockStr := string(lockContent)

		// Verify the custom path is used
		if !strings.Contains(lockStr, "Validate custom AWF binary") {
			t.Error("Expected lock file to contain 'Validate custom AWF binary' step")
		}

		if !strings.Contains(lockStr, "/custom/path/to/awf") {
			t.Error("Expected lock file to contain custom AWF path '/custom/path/to/awf'")
		}

		// Verify the install step is NOT present
		if strings.Contains(lockStr, "Install awf binary") {
			t.Error("Expected lock file NOT to contain 'Install awf binary' step when custom path is specified")
		}

		// Verify the execution step uses the custom path
		// Note: The path is shell-escaped only if it contains special chars
		if !strings.Contains(lockStr, "sudo -E /custom/path/to/awf") &&
			!strings.Contains(lockStr, "sudo -E '/custom/path/to/awf'") {
			t.Error("Expected execution step to use custom AWF path")
		}
	})

	t.Run("workflow with relative path compiles correctly", func(t *testing.T) {
		// Create a temporary test directory
		testDir := t.TempDir()
		workflowsDir := filepath.Join(testDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create workflows directory: %v", err)
		}

		// Create a test workflow with relative AWF path
		workflowContent := `---
name: Test Custom AWF Path (Relative)
on:
  workflow_dispatch:
engine: copilot
tools:
  github:
    mode: remote
network:
  allowed:
    - defaults
  firewall:
    path: bin/my-custom-awf
permissions:
  issues: read
  pull-requests: read
---
Test workflow with relative custom AWF binary path.
`
		workflowPath := filepath.Join(workflowsDir, "test-relative-awf-path.md")
		err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test-relative-awf-path")
		compiler.SetSkipValidation(true)
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the compiled lock file
		lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockStr := string(lockContent)

		// Verify the relative path is resolved against GITHUB_WORKSPACE
		if !strings.Contains(lockStr, "${GITHUB_WORKSPACE}/bin/my-custom-awf") {
			t.Error("Expected lock file to contain resolved relative path '${GITHUB_WORKSPACE}/bin/my-custom-awf'")
		}
	})

	t.Run("workflow without custom path uses default installation", func(t *testing.T) {
		// Create a temporary test directory
		testDir := t.TempDir()
		workflowsDir := filepath.Join(testDir, ".github", "workflows")
		err := os.MkdirAll(workflowsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create workflows directory: %v", err)
		}

		// Create a test workflow without custom AWF path
		workflowContent := `---
name: Test Default AWF Installation
on:
  workflow_dispatch:
engine: copilot
tools:
  github:
    mode: remote
network:
  allowed:
    - defaults
  firewall:
    log-level: info
permissions:
  issues: read
  pull-requests: read
---
Test workflow with default AWF installation.
`
		workflowPath := filepath.Join(workflowsDir, "test-default-awf.md")
		err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test-default-awf")
		compiler.SetSkipValidation(true)
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the compiled lock file
		lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockStr := string(lockContent)

		// Verify the install step is present
		if !strings.Contains(lockStr, "Install awf binary") {
			t.Error("Expected lock file to contain 'Install awf binary' step when no custom path is specified")
		}

		// Verify the validation step is NOT present
		if strings.Contains(lockStr, "Validate custom AWF binary") {
			t.Error("Expected lock file NOT to contain 'Validate custom AWF binary' step when no custom path")
		}
	})
}
