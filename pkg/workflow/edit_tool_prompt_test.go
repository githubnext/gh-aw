package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditToolPromptIncludedWhenEnabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-edit-prompt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with edit tool enabled
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  edit:
---

# Test Workflow with Edit Tool

This is a test workflow with edit tool enabled.
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

	// Test 1: Verify edit tool prompt step is created
	if !strings.Contains(lockStr, "- name: Append edit tool accessibility instructions to prompt") {
		t.Error("Expected 'Append edit tool accessibility instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the instruction text contains the workspace path
	if !strings.Contains(lockStr, "$GITHUB_WORKSPACE") {
		t.Error("Expected $GITHUB_WORKSPACE reference in generated workflow")
	}

	// Test 3: Verify the instruction text contains the /tmp/gh-aw/ path
	if !strings.Contains(lockStr, "/tmp/gh-aw/") {
		t.Error("Expected /tmp/gh-aw/ reference in generated workflow")
	}

	// Test 4: Verify the instruction mentions File Editing Access
	if !strings.Contains(lockStr, "File Editing Access") {
		t.Error("Expected 'File Editing Access' header in generated workflow")
	}

	// Test 5: Verify the instruction mentions accessible directories
	if !strings.Contains(lockStr, "write access") {
		t.Error("Expected 'write access' reference in generated workflow")
	}

	t.Logf("Successfully verified edit tool accessibility instructions are included in generated workflow")
}

func TestEditToolPromptNotIncludedWhenDisabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-no-edit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow WITHOUT edit tool
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: codex
tools:
  github:
---

# Test Workflow without Edit Tool

This is a test workflow without edit tool.
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

	// Test: Verify edit tool prompt step is NOT created
	if strings.Contains(lockStr, "- name: Append edit tool accessibility instructions to prompt") {
		t.Error("Did not expect 'Append edit tool accessibility instructions to prompt' step in workflow without edit tool")
	}

	if strings.Contains(lockStr, "File Editing Access") {
		t.Error("Did not expect 'File Editing Access' header in workflow without edit tool")
	}

	t.Logf("Successfully verified edit tool accessibility instructions are NOT included when edit tool is disabled")
}

func TestEditToolPromptOrderAfterPlaywright(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-edit-order-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with both playwright and edit tools
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  playwright:
  edit:
---

# Test Workflow

This is a test workflow to verify edit instructions come after playwright.
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

	// Find positions of playwright and edit instructions
	playwrightPos := strings.Index(lockStr, "Append playwright output directory instructions to prompt")
	editPos := strings.Index(lockStr, "Append edit tool accessibility instructions to prompt")

	// Test: Verify edit instructions come after playwright instructions
	if playwrightPos == -1 {
		t.Error("Expected playwright output directory instructions in generated workflow")
	}

	if editPos == -1 {
		t.Error("Expected edit tool accessibility instructions in generated workflow")
	}

	if playwrightPos != -1 && editPos != -1 && editPos <= playwrightPos {
		t.Errorf("Expected edit instructions to come after playwright instructions, but found at positions Playwright=%d, Edit=%d", playwrightPos, editPos)
	}

	t.Logf("Successfully verified edit instructions come after playwright instructions in generated workflow")
}
