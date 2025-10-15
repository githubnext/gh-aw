package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlaywrightPromptIncludedWhenEnabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-playwright-prompt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with playwright tool enabled
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  playwright:
---

# Test Workflow with Playwright

This is a test workflow with playwright enabled.
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

	// Test 1: Verify playwright prompt step is created
	if !strings.Contains(lockStr, "- name: Append playwright output directory instructions to prompt") {
		t.Error("Expected 'Append playwright output directory instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the instruction text contains the output directory path
	if !strings.Contains(lockStr, "/tmp/gh-aw/mcp-logs/playwright/") {
		t.Error("Expected playwright output directory path /tmp/gh-aw/mcp-logs/playwright/ in generated workflow")
	}

	// Test 3: Verify the instruction mentions Playwright and output-dir
	if !strings.Contains(lockStr, "Playwright Output Directory") {
		t.Error("Expected 'Playwright Output Directory' header in generated workflow")
	}

	if !strings.Contains(lockStr, "--output-dir") {
		t.Error("Expected '--output-dir' reference in generated workflow")
	}

	t.Logf("Successfully verified playwright output directory instructions are included in generated workflow")
}

func TestPlaywrightPromptNotIncludedWhenDisabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-no-playwright-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow WITHOUT playwright tool
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: codex
tools:
  github:
---

# Test Workflow without Playwright

This is a test workflow without playwright.
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

	// Test: Verify playwright prompt step is NOT created
	if strings.Contains(lockStr, "- name: Append playwright output directory instructions to prompt") {
		t.Error("Did not expect 'Append playwright output directory instructions to prompt' step in workflow without playwright")
	}

	if strings.Contains(lockStr, "Playwright Output Directory") {
		t.Error("Did not expect 'Playwright Output Directory' header in workflow without playwright")
	}

	t.Logf("Successfully verified playwright output directory instructions are NOT included when playwright is disabled")
}

func TestPlaywrightPromptOrderAfterTempFolder(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-playwright-order-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with playwright
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  playwright:
---

# Test Workflow

This is a test workflow to verify playwright instructions come after temp folder.
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

	// Find positions of temp folder and playwright instructions
	tempFolderPos := strings.Index(lockStr, "Append temporary folder instructions to prompt")
	playwrightPos := strings.Index(lockStr, "Append playwright output directory instructions to prompt")

	// Test: Verify playwright instructions come after temp folder instructions
	if tempFolderPos == -1 {
		t.Error("Expected temporary folder instructions in generated workflow")
	}

	if playwrightPos == -1 {
		t.Error("Expected playwright output directory instructions in generated workflow")
	}

	if tempFolderPos != -1 && playwrightPos != -1 && playwrightPos <= tempFolderPos {
		t.Errorf("Expected playwright instructions to come after temp folder instructions, but found at positions TempFolder=%d, Playwright=%d", tempFolderPos, playwrightPos)
	}

	t.Logf("Successfully verified playwright instructions come after temp folder instructions in generated workflow")
}
