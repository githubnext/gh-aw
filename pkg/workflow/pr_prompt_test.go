package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestPRContextPromptIncludedForIssueComment(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-pr-context-prompt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with issue_comment trigger
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on:
  issue_comment:
    types: [created]
permissions:
  contents: read
engine: claude
---

# Test Workflow with Issue Comment

This is a test workflow with issue_comment trigger.
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

	// Test 1: Verify PR context prompt step is created
	if !strings.Contains(lockStr, "- name: Append PR context instructions to prompt") {
		t.Error("Expected 'Append PR context instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the instruction mentions PR branch checkout
	if !strings.Contains(lockStr, "pull request") {
		t.Error("Expected 'pull request' reference in generated workflow")
	}

	t.Logf("Successfully verified PR context instructions are included for issue_comment trigger")
}

func TestPRContextPromptIncludedForCommand(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-pr-context-command-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with command trigger
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on:
  command:
    name: mybot
permissions:
  contents: read
engine: claude
---

# Test Workflow with Command

This is a test workflow with command trigger.
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

	// Test: Verify PR context prompt step is created for command triggers
	if !strings.Contains(lockStr, "- name: Append PR context instructions to prompt") {
		t.Error("Expected 'Append PR context instructions to prompt' step in workflow with command trigger")
	}

	t.Logf("Successfully verified PR context instructions are included for command trigger")
}

func TestPRContextPromptNotIncludedForPush(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-no-pr-context-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with push trigger (no comment triggers)
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
permissions:
  contents: read
engine: claude
---

# Test Workflow without Comment Triggers

This is a test workflow with push trigger only.
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

	// Test: Verify PR context prompt step is NOT created for push triggers
	if strings.Contains(lockStr, "- name: Append PR context instructions to prompt") {
		t.Error("Did not expect 'Append PR context instructions to prompt' step for push trigger")
	}

	t.Logf("Successfully verified PR context instructions are NOT included for push trigger")
}

func TestPRContextPromptNotIncludedWithoutCheckout(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-pr-no-checkout-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with comment trigger but no checkout (no contents permission)
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on:
  issue_comment:
    types: [created]
permissions:
  issues: read
engine: claude
---

# Test Workflow without Contents Permission

This is a test workflow without contents read permission.
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

	// Test: Verify PR context prompt step is NOT created without contents permission
	if strings.Contains(lockStr, "- name: Append PR context instructions to prompt") {
		t.Error("Did not expect 'Append PR context instructions to prompt' step without contents read permission")
	}

	t.Logf("Successfully verified PR context instructions are NOT included without contents permission")
}
