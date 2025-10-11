package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStepSummaryIncludesProcessedOutput(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "step-summary-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with Claude engine
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
safe-outputs:
  create-issue:
---

# Test Step Summary with Processed Output

This workflow tests that the step summary includes both JSONL and processed output.
`

	testFile := filepath.Join(tmpDir, "test-step-summary.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-step-summary.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify that the "Print sanitized agent output" step no longer exists (moved to JavaScript)
	if strings.Contains(lockContent, "- name: Print sanitized agent output") {
		t.Error("Did not expect 'Print sanitized agent output' step (should be in JavaScript now)")
	}

	// Verify that the JavaScript uses addRaw to build the summary
	if strings.Count(lockContent, ".addRaw(") < 2 {
		t.Error("Expected at least 2 '.addRaw(' calls in JavaScript code for summary building")
	}

	t.Log("Step summary correctly includes processed output sections")
}

func TestStepSummaryIncludesAgenticRunInfo(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "agentic-run-info-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with Claude engine including extended configuration
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine:
  id: claude
  model: claude-3-5-sonnet-20241022
  version: beta
---

# Test Agentic Run Info Step Summary

This workflow tests that the step summary includes agentic run information.
`

	testFile := filepath.Join(tmpDir, "test-agentic-run-info.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-agentic-run-info.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify that the "Generate agentic run info" step exists
	if !strings.Contains(lockContent, "- name: Generate agentic run info") {
		t.Error("Expected 'Generate agentic run info' step")
	}

	// Verify that the step does NOT include the "Agentic Run Information" section in step summary
	if strings.Contains(lockContent, "## Agentic Run Information") {
		t.Error("Did not expect '## Agentic Run Information' section in step summary (it should only be in action logs)")
	}

	// Verify that the aw_info.json file is still created and logged to console
	if !strings.Contains(lockContent, "aw_info.json") {
		t.Error("Expected 'aw_info.json' to be created")
	}

	if !strings.Contains(lockContent, "console.log('Generated aw_info.json at:', tmpPath);") {
		t.Error("Expected console.log output for aw_info.json")
	}

	t.Log("Step correctly creates aw_info.json without adding to step summary")
}
