package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
)

// ============================================================================
// Safe Outputs Prompt Tests
// ============================================================================

func TestGenerateSafeOutputsPromptStep_IncludesWhenEnabled(t *testing.T) {
	compiler := &Compiler{}
	var yaml strings.Builder

	// Create a config with create-issue enabled
	safeOutputs := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{},
	}

	compiler.generateSafeOutputsPromptStep(&yaml, safeOutputs)

	output := yaml.String()
	if !strings.Contains(output, "Append safe outputs instructions to prompt") {
		t.Error("Expected safe outputs prompt step to be generated when enabled")
	}
	if !strings.Contains(output, "safe output tool") {
		t.Error("Expected prompt to mention safe output tools")
	}
	if !strings.Contains(output, "gh CLI is NOT authenticated") {
		t.Error("Expected prompt to warn about gh CLI not being authenticated")
	}
	if !strings.Contains(output, "create_issue") {
		t.Error("Expected prompt to list create_issue tool")
	}
}

func TestGenerateSafeOutputsPromptStep_SkippedWhenDisabled(t *testing.T) {
	compiler := &Compiler{}
	var yaml strings.Builder

	// Pass nil for disabled
	compiler.generateSafeOutputsPromptStep(&yaml, nil)

	output := yaml.String()
	if strings.Contains(output, "safe outputs") {
		t.Error("Expected safe outputs prompt step to NOT be generated when disabled")
	}
}

func TestSafeOutputsPromptText_FollowsXMLFormat(t *testing.T) {
	// This test is for the embedded prompt text which is no longer used
	// Skip it as we now generate the prompt dynamically
	t.Skip("Safe outputs prompt is now generated dynamically based on enabled tools")
}

// ============================================================================
// Cache Memory Prompt Tests
// ============================================================================

func TestCacheMemoryPromptIncludedWhenEnabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-cache-memory-prompt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with cache-memory enabled
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  cache-memory: true
---

# Test Workflow with Cache Memory

This is a test workflow with cache-memory enabled.
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

	// Test 1: Verify cache memory prompt step is created
	if !strings.Contains(lockStr, "- name: Append cache memory instructions to prompt") {
		t.Error("Expected 'Append cache memory instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the instruction text contains cache folder information
	if !strings.Contains(lockStr, "Cache Folder Available") {
		t.Error("Expected 'Cache Folder Available' header in generated workflow")
	}

	// Test 3: Verify the instruction text contains the cache directory path
	if !strings.Contains(lockStr, "/tmp/gh-aw/cache-memory/") {
		t.Error("Expected '/tmp/gh-aw/cache-memory/' reference in generated workflow")
	}

	// Test 4: Verify the instruction mentions persistent cache
	if !strings.Contains(lockStr, "persist") {
		t.Error("Expected 'persist' reference in generated workflow")
	}

	t.Logf("Successfully verified cache memory instructions are included in generated workflow")
}

func TestCacheMemoryPromptNotIncludedWhenDisabled(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-no-cache-memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow WITHOUT cache-memory
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  github:
---

# Test Workflow without Cache Memory

This is a test workflow without cache-memory.
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

	// Test: Verify cache memory prompt step is NOT created
	if strings.Contains(lockStr, "- name: Append cache memory instructions to prompt") {
		t.Error("Did not expect 'Append cache memory instructions to prompt' step in workflow without cache-memory")
	}

	if strings.Contains(lockStr, "Cache Folder Available") {
		t.Error("Did not expect 'Cache Folder Available' header in workflow without cache-memory")
	}

	t.Logf("Successfully verified cache memory instructions are NOT included when cache-memory is disabled")
}

func TestCacheMemoryPromptMultipleCaches(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-multi-cache-memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with multiple cache-memory entries
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
tools:
  cache-memory:
    - id: default
      key: cache-1
    - id: session
      key: cache-2
---

# Test Workflow with Multiple Caches

This is a test workflow with multiple cache-memory entries.
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

	// Test 1: Verify cache memory prompt step is created
	if !strings.Contains(lockStr, "- name: Append cache memory instructions to prompt") {
		t.Error("Expected 'Append cache memory instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify plural form is used for multiple caches
	if !strings.Contains(lockStr, "Cache Folders Available") {
		t.Error("Expected 'Cache Folders Available' (plural) header for multiple caches")
	}

	// Test 3: Verify both cache directories are mentioned
	if !strings.Contains(lockStr, "/tmp/gh-aw/cache-memory/") {
		t.Error("Expected '/tmp/gh-aw/cache-memory/' reference for default cache")
	}

	if !strings.Contains(lockStr, "/tmp/gh-aw/cache-memory-session/") {
		t.Error("Expected '/tmp/gh-aw/cache-memory-session/' reference for session cache")
	}

	t.Logf("Successfully verified cache memory instructions handle multiple caches")
}

// ============================================================================
// Playwright Prompt Tests
// ============================================================================

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
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify playwright prompt step is created
	if !strings.Contains(lockStr, "- name: Append playwright output directory instructions to prompt") {
		t.Error("Expected 'Append playwright output directory instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the cat command for playwright prompt file is included
	if !strings.Contains(lockStr, "cat \"/opt/gh-aw/prompts/playwright_prompt.md\" >> \"$GH_AW_PROMPT\"") {
		t.Error("Expected cat command for playwright prompt file in generated workflow")
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
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

// ============================================================================
// PR Context Prompt Tests
// ============================================================================

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
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify PR context prompt step is created
	if !strings.Contains(lockStr, "- name: Append PR context instructions to prompt") {
		t.Error("Expected 'Append PR context instructions to prompt' step in generated workflow")
	}

	// Test 2: Verify the cat command for PR context prompt file is included
	if !strings.Contains(lockStr, "cat \"/opt/gh-aw/prompts/pr_context_prompt.md\" >> \"$GH_AW_PROMPT\"") {
		t.Error("Expected cat command for PR context prompt file in generated workflow")
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
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
	lockFile := stringutil.MarkdownToLockFile(testFile)
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
