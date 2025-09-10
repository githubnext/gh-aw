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

	// Verify that the "Print sanitized agent output" step exists
	if !strings.Contains(lockContent, "- name: Print sanitized agent output") {
		t.Error("Expected 'Print sanitized agent output' step")
	}

	// Verify that the step includes the original JSONL output section
	if !strings.Contains(lockContent, "## Agent Output (JSONL)") {
		t.Error("Expected '## Agent Output (JSONL)' section in step summary")
	}

	// Verify that the step includes the new processed output section
	if !strings.Contains(lockContent, "## Processed Output") {
		t.Error("Expected '## Processed Output' section in step summary")
	}

	// Verify that the processed output references the collect_output step output
	if !strings.Contains(lockContent, "${{ steps.collect_output.outputs.output }}") {
		t.Error("Expected reference to steps.collect_output.outputs.output in step summary")
	}

	// Verify both outputs are in code blocks
	jsonlBlockCount := strings.Count(lockContent, "echo '``````json'")
	if jsonlBlockCount < 2 {
		t.Errorf("Expected at least 2 JSON code blocks in step summary, got %d", jsonlBlockCount)
	}

	codeBlockEndCount := strings.Count(lockContent, "echo '``````'")
	if codeBlockEndCount < 2 {
		t.Errorf("Expected at least 2 code block end markers in step summary, got %d", codeBlockEndCount)
	}

	t.Log("Step summary correctly includes both JSONL and processed output sections")
}
