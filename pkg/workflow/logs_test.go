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

	expected := []string{
		"touch /tmp/agent-output.log",
		"tail -10 /tmp/agent-output.log",
	}

	for _, expected := range expected {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected compiled workflow to contain '%s', but it didn't.\nCompiled content:\n%s", expected, result)
		}
	}
}
