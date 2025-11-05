package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestStepValidatorWithImports(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("imported step with gh command but no GH_TOKEN should fail", func(t *testing.T) {
		// Create a shared steps file with gh command but no GH_TOKEN
		sharedStepsPath := filepath.Join(tempDir, "shared-steps-invalid.md")
		sharedStepsContent := `---
on: push
steps:
  - name: Run gh command
    run: gh issue list
---
`
		if err := os.WriteFile(sharedStepsPath, []byte(sharedStepsContent), 0644); err != nil {
			t.Fatalf("Failed to write shared steps file: %v", err)
		}

		// Create a workflow file that imports the shared steps
		workflowPath := filepath.Join(tempDir, "test-workflow-invalid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
imports:
  - shared-steps-invalid.md
---

# Test Workflow

This workflow should fail validation.
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow - should fail
		compiler := workflow.NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(workflowPath)

		if err == nil {
			t.Fatal("Expected compilation to fail due to missing GH_TOKEN in imported step")
		}

		if !strings.Contains(err.Error(), "imported steps validation failed") {
			t.Errorf("Expected error to mention 'imported steps validation failed', got: %v", err)
		}

		if !strings.Contains(err.Error(), "GH_TOKEN") {
			t.Errorf("Expected error to mention 'GH_TOKEN', got: %v", err)
		}
	})

	t.Run("imported step with gh command and GH_TOKEN should succeed", func(t *testing.T) {
		// Create a shared steps file with gh command and GH_TOKEN
		sharedStepsPath := filepath.Join(tempDir, "shared-steps-valid.md")
		sharedStepsContent := `---
on: push
steps:
  - name: Run gh command
    run: gh issue list
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---
`
		if err := os.WriteFile(sharedStepsPath, []byte(sharedStepsContent), 0644); err != nil {
			t.Fatalf("Failed to write shared steps file: %v", err)
		}

		// Create a workflow file that imports the shared steps
		workflowPath := filepath.Join(tempDir, "test-workflow-valid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
imports:
  - shared-steps-valid.md
---

# Test Workflow

This workflow should pass validation.
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow - should succeed
		compiler := workflow.NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(workflowPath)

		if err != nil {
			t.Fatalf("Expected compilation to succeed, but got error: %v", err)
		}

		// Verify lock file was created
		lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			t.Error("Expected lock file to be created")
		}
	})

	t.Run("main step with gh command but no GH_TOKEN should fail", func(t *testing.T) {
		// Create a workflow file with main steps that have gh command but no GH_TOKEN
		workflowPath := filepath.Join(tempDir, "test-workflow-main-invalid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
steps:
  - name: Run gh command
    run: gh pr list
---

# Test Workflow

This workflow should fail validation due to missing GH_TOKEN in main steps.
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow - should fail
		compiler := workflow.NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(workflowPath)

		if err == nil {
			t.Fatal("Expected compilation to fail due to missing GH_TOKEN in main step")
		}

		if !strings.Contains(err.Error(), "main steps validation failed") {
			t.Errorf("Expected error to mention 'main steps validation failed', got: %v", err)
		}

		if !strings.Contains(err.Error(), "GH_TOKEN") {
			t.Errorf("Expected error to mention 'GH_TOKEN', got: %v", err)
		}
	})

	t.Run("main step with gh command and GH_TOKEN should succeed", func(t *testing.T) {
		// Create a workflow file with main steps that have gh command and GH_TOKEN
		workflowPath := filepath.Join(tempDir, "test-workflow-main-valid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
steps:
  - name: Run gh command
    run: gh pr list
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

# Test Workflow

This workflow should pass validation.
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow - should succeed
		compiler := workflow.NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(workflowPath)

		if err != nil {
			t.Fatalf("Expected compilation to succeed, but got error: %v", err)
		}

		// Verify lock file was created
		lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			t.Error("Expected lock file to be created")
		}
	})

	t.Run("step without gh command does not require GH_TOKEN", func(t *testing.T) {
		// Create a workflow file with steps that don't use gh
		workflowPath := filepath.Join(tempDir, "test-workflow-no-gh.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
steps:
  - name: Echo something
    run: echo "Hello world"
---

# Test Workflow

This workflow should pass validation without GH_TOKEN.
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow - should succeed
		compiler := workflow.NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(workflowPath)

		if err != nil {
			t.Fatalf("Expected compilation to succeed, but got error: %v", err)
		}

		// Verify lock file was created
		lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			t.Error("Expected lock file to be created")
		}
	})
}
