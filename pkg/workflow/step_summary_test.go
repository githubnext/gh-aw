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

	// Verify that the step includes the original JSONL output section
	if !strings.Contains(lockContent, "## Safe Outputs (JSONL)") {
		t.Error("Expected '## Safe Outputs (JSONL)' section in step summary")
	}

	// Verify that the JavaScript code includes the new processed output section via core.summary
	if !strings.Contains(lockContent, "## Processed Output") {
		t.Error("Expected '## Processed Output' section in JavaScript code")
	}

	// Verify that the JavaScript code uses core.summary to write processed output
	if !strings.Contains(lockContent, "core.summary") {
		t.Error("Expected 'core.summary' usage in JavaScript code")
	}

	// Verify that the JavaScript uses addRaw to build the summary
	if strings.Count(lockContent, ".addRaw(") < 2 {
		t.Error("Expected at least 2 '.addRaw(' calls in JavaScript code for summary building")
	}

	t.Log("Step summary correctly includes both JSONL and processed output sections")
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

	// Verify that the step includes the "Agentic Run Information" section in step summary
	if !strings.Contains(lockContent, "## Agentic Run Information") {
		t.Error("Expected '## Agentic Run Information' section in step summary")
	}

	// Verify that the step uses core.summary for step summary output
	if !strings.Contains(lockContent, "core.summary") {
		t.Error("Expected 'core.summary' usage for step summary output")
	}

	// Verify that the step includes addRaw method calls
	if !strings.Contains(lockContent, ".addRaw(") {
		t.Error("Expected '.addRaw(' method calls for step summary content")
	}

	// Verify that the step includes JSON code block markers in the summary
	if !strings.Contains(lockContent, "```json") {
		t.Error("Expected '```json' code block markers in step summary")
	}

	// Verify that the step includes write() call to finalize summary
	if !strings.Contains(lockContent, ".write();") {
		t.Error("Expected '.write();' call to finalize step summary")
	}

	t.Log("Step summary correctly includes agentic run information")
}
