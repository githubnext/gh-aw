package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeExecutionLogCapture(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "log-capture-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
on: push
engine: claude
tools:
  github:
    allowed: [get_issue]
---

# Test Workflow

This is a test workflow.`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	result := string(lockContent)

	// Log capture is now done inline with tee during execution
	// No longer uses separate "Print agent log" step
	expected := []string{
		"2>&1 | tee /tmp/gh-aw/agent-stdio.log",
	}

	for _, expected := range expected {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected compiled workflow to contain '%s', but it didn't.\nCompiled content:\n%s", expected, result)
		}
	}

	// Verify that the old log capture step is NOT present
	notExpected := []string{
		"touch /tmp/gh-aw/agent-stdio.log",
		"cat /tmp/gh-aw/agent-stdio.log >> $GITHUB_STEP_SUMMARY",
		"cat /tmp/gh-aw/agent-stdio.log >> \"$GITHUB_STEP_SUMMARY\"",
	}

	for _, notExpected := range notExpected {
		if strings.Contains(result, notExpected) {
			t.Errorf("Expected compiled workflow NOT to contain '%s' (old log capture method), but it did.\nCompiled content:\n%s", notExpected, result)
		}
	}
}
