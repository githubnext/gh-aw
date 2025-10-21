package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestImportedPostSteps(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "imported-post-steps-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shared workflow file with post-steps
	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared dir: %v", err)
	}

	sharedContent := `---
post-steps:
  - name: Shared Post Step
    run: echo "This is from the imported post-step"
---

# Shared Post-Step Configuration
This is a shared workflow configuration with post-steps.
`
	sharedFile := filepath.Join(sharedDir, "shared-post.md")
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create a main workflow that imports the shared post-steps
	mainWorkflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/shared-post.md
post-steps:
  - name: Main Post Step
    run: echo "This is from the main workflow post-step"
---

# Test Workflow with Imported Post-Steps

Test that post-steps are properly merged from imports.
`
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(mainWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with imported post-steps: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify both post-steps are present
	if !strings.Contains(lockStr, "Shared Post Step") {
		t.Error("Expected to find 'Shared Post Step' from imported file")
	}

	if !strings.Contains(lockStr, "Main Post Step") {
		t.Error("Expected to find 'Main Post Step' from main workflow")
	}

	if !strings.Contains(lockStr, "This is from the imported post-step") {
		t.Error("Expected to find imported post-step command")
	}

	if !strings.Contains(lockStr, "This is from the main workflow post-step") {
		t.Error("Expected to find main workflow post-step command")
	}

	// Parse the YAML to verify structure
	var workflow map[string]any
	if err := yaml.Unmarshal(lockContent, &workflow); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify the agent job has steps
	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		t.Fatal("Expected jobs in workflow")
	}

	agent, ok := jobs["agent"].(map[string]any)
	if !ok {
		t.Fatal("Expected agent job in workflow")
	}

	steps, ok := agent["steps"].([]any)
	if !ok {
		t.Fatal("Expected steps in agent job")
	}

	// Find the imported and main post-steps
	var foundSharedPostStep, foundMainPostStep bool
	var sharedPostStepIndex, mainPostStepIndex int

	for i, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		name, ok := stepMap["name"].(string)
		if !ok {
			continue
		}
		if name == "Shared Post Step" {
			foundSharedPostStep = true
			sharedPostStepIndex = i
		}
		if name == "Main Post Step" {
			foundMainPostStep = true
			mainPostStepIndex = i
		}
	}

	if !foundSharedPostStep {
		t.Error("Expected to find 'Shared Post Step' in the steps")
	}

	if !foundMainPostStep {
		t.Error("Expected to find 'Main Post Step' in the steps")
	}

	// Verify that imported post-steps come before main post-steps
	if foundSharedPostStep && foundMainPostStep && sharedPostStepIndex > mainPostStepIndex {
		t.Errorf("Expected shared post-step (index %d) to come before main post-step (index %d)",
			sharedPostStepIndex, mainPostStepIndex)
	}
}

func TestMultipleImportedPostSteps(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "multiple-post-steps-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create shared directory
	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared dir: %v", err)
	}

	// Create first shared workflow file with post-steps
	shared1Content := `---
post-steps:
  - name: First Shared Post Step
    run: echo "First"
---

# First Shared Post-Step
`
	shared1File := filepath.Join(sharedDir, "shared-post-1.md")
	if err := os.WriteFile(shared1File, []byte(shared1Content), 0644); err != nil {
		t.Fatalf("Failed to write shared1 file: %v", err)
	}

	// Create second shared workflow file with post-steps
	shared2Content := `---
post-steps:
  - name: Second Shared Post Step
    run: echo "Second"
---

# Second Shared Post-Step
`
	shared2File := filepath.Join(sharedDir, "shared-post-2.md")
	if err := os.WriteFile(shared2File, []byte(shared2Content), 0644); err != nil {
		t.Fatalf("Failed to write shared2 file: %v", err)
	}

	// Create a main workflow that imports both shared post-steps
	mainWorkflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/shared-post-1.md
  - shared/shared-post-2.md
---

# Test Workflow with Multiple Imported Post-Steps

Test that multiple imported post-steps are properly merged.
`
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(mainWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with multiple imported post-steps: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify both imported post-steps are present
	if !strings.Contains(lockStr, "First Shared Post Step") {
		t.Error("Expected to find 'First Shared Post Step' from first imported file")
	}

	if !strings.Contains(lockStr, "Second Shared Post Step") {
		t.Error("Expected to find 'Second Shared Post Step' from second imported file")
	}

	// Parse the YAML to verify structure
	var workflow map[string]any
	if err := yaml.Unmarshal(lockContent, &workflow); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify the agent job has steps
	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		t.Fatal("Expected jobs in workflow")
	}

	agent, ok := jobs["agent"].(map[string]any)
	if !ok {
		t.Fatal("Expected agent job in workflow")
	}

	steps, ok := agent["steps"].([]any)
	if !ok {
		t.Fatal("Expected steps in agent job")
	}

	// Find the imported post-steps in order
	var foundFirstStep, foundSecondStep bool
	var firstStepIndex, secondStepIndex int

	for i, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		name, ok := stepMap["name"].(string)
		if !ok {
			continue
		}
		if name == "First Shared Post Step" {
			foundFirstStep = true
			firstStepIndex = i
		}
		if name == "Second Shared Post Step" {
			foundSecondStep = true
			secondStepIndex = i
		}
	}

	if !foundFirstStep {
		t.Error("Expected to find 'First Shared Post Step' in the steps")
	}

	if !foundSecondStep {
		t.Error("Expected to find 'Second Shared Post Step' in the steps")
	}

	// Verify that post-steps are in import order
	if foundFirstStep && foundSecondStep && firstStepIndex > secondStepIndex {
		t.Errorf("Expected first post-step (index %d) to come before second post-step (index %d)",
			firstStepIndex, secondStepIndex)
	}
}
