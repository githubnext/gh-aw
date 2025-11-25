package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestConclusionStepsGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "conclusion-steps-test")

	// Create a test markdown file with conclusion-steps
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
  conclusion-steps:
    - name: Custom step 1
      run: echo "Hello from conclusion step 1"
    - name: Custom step 2
      run: echo "Hello from conclusion step 2"
---

# Test Conclusion Steps

Test that custom steps are generated inside the conclusion job.
`

	testFile := filepath.Join(tmpDir, "test-conclusion-steps.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-conclusion-steps.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that conclusion job exists
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Extract conclusion job section
	conclusionSection := extractJobSection(compiled, "conclusion")
	if conclusionSection == "" {
		t.Fatal("Could not extract conclusion job section")
	}

	// Verify that custom conclusion-steps are in the conclusion job
	if !strings.Contains(conclusionSection, "Custom step 1") {
		t.Error("Conclusion job should contain 'Custom step 1'")
	}

	if !strings.Contains(conclusionSection, "Custom step 2") {
		t.Error("Conclusion job should contain 'Custom step 2'")
	}

	if !strings.Contains(conclusionSection, "echo \"Hello from conclusion step 1\"") {
		t.Error("Conclusion job should contain the run command for step 1")
	}

	if !strings.Contains(conclusionSection, "echo \"Hello from conclusion step 2\"") {
		t.Error("Conclusion job should contain the run command for step 2")
	}
}

func TestConclusionStepsOrder(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "conclusion-steps-order-test")

	// Create a test markdown file with conclusion-steps and noop
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 3
  conclusion-steps:
    - name: Custom conclusion step
      run: echo "This should run after download but before noop"
---

# Test Conclusion Steps Order

Verify that conclusion-steps come after artifact download but before noop processing.
`

	testFile := filepath.Join(tmpDir, "test-conclusion-steps-order.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-conclusion-steps-order.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Extract conclusion job section
	conclusionSection := extractJobSection(compiled, "conclusion")
	if conclusionSection == "" {
		t.Fatal("Could not extract conclusion job section")
	}

	// Verify order: Download artifact should come before custom step
	downloadIndex := strings.Index(conclusionSection, "Download agent output artifact")
	customStepIndex := strings.Index(conclusionSection, "Custom conclusion step")
	noopIndex := strings.Index(conclusionSection, "Process No-Op Messages")

	if downloadIndex == -1 {
		t.Error("Could not find 'Download agent output artifact' in conclusion job")
	}
	if customStepIndex == -1 {
		t.Error("Could not find 'Custom conclusion step' in conclusion job")
	}
	if noopIndex == -1 {
		t.Error("Could not find 'Process No-Op Messages' in conclusion job")
	}

	// Verify order: download < custom step < noop
	if downloadIndex > customStepIndex {
		t.Error("Download artifact step should come before custom conclusion step")
	}
	if customStepIndex > noopIndex {
		t.Error("Custom conclusion step should come before noop processing step")
	}
}

func TestConclusionStepsImport(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "conclusion-steps-import-test")

	// Create a shared workflow file with conclusion-steps
	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	sharedContent := `---
safe-outputs:
  conclusion-steps:
    - name: Imported conclusion step
      run: echo "Hello from imported step"
---

# Shared conclusion steps configuration
`

	sharedFile := filepath.Join(sharedDir, "shared-conclusion.md")
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow that imports the shared file
	mainContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
imports:
  - shared/shared-conclusion.md
safe-outputs:
  noop:
---

# Test Conclusion Steps Import

Test that conclusion-steps can be imported from shared workflows.
`

	mainFile := filepath.Join(tmpDir, "test-import.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-import.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that conclusion job exists
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Extract conclusion job section
	conclusionSection := extractJobSection(compiled, "conclusion")
	if conclusionSection == "" {
		t.Fatal("Could not extract conclusion job section")
	}

	// Verify that imported conclusion-steps are in the conclusion job
	if !strings.Contains(conclusionSection, "Imported conclusion step") {
		t.Error("Conclusion job should contain 'Imported conclusion step'")
	}

	if !strings.Contains(conclusionSection, "echo \"Hello from imported step\"") {
		t.Error("Conclusion job should contain the imported run command")
	}
}
