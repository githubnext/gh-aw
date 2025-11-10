package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
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
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
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
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
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
