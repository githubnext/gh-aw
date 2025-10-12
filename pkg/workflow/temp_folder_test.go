package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTempFolderPromptIncluded(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-temp-folder-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple test workflow
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: codex
---

# Test Workflow

This is a test workflow to verify temp folder instructions are included.
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify temporary folder step is created
	if !strings.Contains(lockStr, "- name: Append temporary folder instructions to prompt") {
		t.Error("Expected 'Append temporary folder instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the instruction text is present
	if !strings.Contains(lockStr, "always use the `/tmp/gh-aw/` directory") {
		t.Error("Expected temp folder instruction text in generated workflow")
	}

	// Test 3: Verify the DO NOT message is present
	if !strings.Contains(lockStr, "DO NOT") && !strings.Contains(lockStr, "use the root `/tmp/` directory") {
		t.Error("Expected warning about not using root /tmp/ directory in generated workflow")
	}

	// Test 4: Verify example usage is present
	if !strings.Contains(lockStr, "mkdir -p /tmp/gh-aw/my-temp-work") {
		t.Error("Expected example usage in generated workflow")
	}

	t.Logf("Successfully verified temporary folder instructions are included in generated workflow")
}

func TestTempFolderPromptOrderAfterXPIA(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-temp-folder-order-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple test workflow
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: codex
---

# Test Workflow

This is a test workflow to verify temp folder instructions come after XPIA.
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Find positions of XPIA and temp folder instructions
	xpiaPos := strings.Index(lockStr, "Append XPIA security instructions to prompt")
	tempFolderPos := strings.Index(lockStr, "Append temporary folder instructions to prompt")

	// Test: Verify temp folder instructions come after XPIA
	if xpiaPos == -1 {
		t.Error("Expected XPIA security instructions in generated workflow")
	}

	if tempFolderPos == -1 {
		t.Error("Expected temporary folder instructions in generated workflow")
	}

	if xpiaPos != -1 && tempFolderPos != -1 && tempFolderPos <= xpiaPos {
		t.Errorf("Expected temporary folder instructions to come after XPIA security instructions, but found at positions XPIA=%d, TempFolder=%d", xpiaPos, tempFolderPos)
	}

	t.Logf("Successfully verified temporary folder instructions come after XPIA in generated workflow")
}
