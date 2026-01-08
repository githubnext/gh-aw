package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify temporary folder step is created
	if !strings.Contains(lockStr, "- name: Append temporary folder instructions to prompt") {
		t.Error("Expected 'Append temporary folder instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the cat command for temp folder prompt file is included
	if !strings.Contains(lockStr, "cat \"/opt/gh-aw/prompts/temp_folder_prompt.md\" >> \"$GH_AW_PROMPT\"") {
		t.Error("Expected cat command for temp folder prompt file in generated workflow")
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
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
