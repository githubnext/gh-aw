package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompilerSharedActionCache(t *testing.T) {
	// Create a temporary directory for test workflows
	tmpDir := t.TempDir()

	// Change to the temp directory so the cache path is consistent
	t.Chdir(tmpDir)

	// Create a compiler instance
	compiler := NewCompiler(false, "", "test")

	// Get the shared action resolver (first time - should initialize)
	cache1, resolver1 := compiler.getSharedActionResolver()
	if cache1 == nil {
		t.Error("Expected cache to be initialized")
	}
	if resolver1 == nil {
		t.Error("Expected resolver to be initialized")
	}

	// Add an entry to the cache
	cache1.Set("actions/checkout", "v5", "test-sha-abc")

	// Get the shared action resolver again (should be same instance)
	cache2, resolver2 := compiler.getSharedActionResolver()

	// Verify it's the same instance
	if cache1 != cache2 {
		t.Error("Expected same cache instance to be returned")
	}
	if resolver1 != resolver2 {
		t.Error("Expected same resolver instance to be returned")
	}

	// Verify the cache entry is still there (proves it's shared)
	sha, found := cache2.Get("actions/checkout", "v5")
	if !found {
		t.Error("Expected to find cached entry")
	}
	if sha != "test-sha-abc" {
		t.Errorf("Expected SHA 'test-sha-abc', got '%s'", sha)
	}
}

func TestCompilerSharedCacheAcrossWorkflows(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()

	// Change to the temp directory
	t.Chdir(tmpDir)

	// Create test workflow files
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflow1Content := `---
on: push
engine: copilot
---
# Test Workflow 1
Test content
`

	workflow2Content := `---
on: pull_request
engine: copilot
---
# Test Workflow 2
Test content
`

	workflow1Path := filepath.Join(workflowsDir, "workflow1.md")
	workflow2Path := filepath.Join(workflowsDir, "workflow2.md")

	if err := os.WriteFile(workflow1Path, []byte(workflow1Content), 0644); err != nil {
		t.Fatalf("Failed to write workflow1: %v", err)
	}
	if err := os.WriteFile(workflow2Path, []byte(workflow2Content), 0644); err != nil {
		t.Fatalf("Failed to write workflow2: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(true)

	// Parse the first workflow
	data1, err := compiler.ParseWorkflowFile(workflow1Path)
	if err != nil {
		t.Fatalf("Failed to parse workflow1: %v", err)
	}

	// Manually add a cache entry via the first workflow's cache
	data1.ActionCache.Set("actions/checkout", "v5", "shared-sha-123")

	// Parse the second workflow
	data2, err := compiler.ParseWorkflowFile(workflow2Path)
	if err != nil {
		t.Fatalf("Failed to parse workflow2: %v", err)
	}

	// Verify the second workflow uses the same cache instance
	if data1.ActionCache != data2.ActionCache {
		t.Error("Expected both workflows to share the same cache instance")
	}

	// Verify the cache entry is available in the second workflow
	sha, found := data2.ActionCache.Get("actions/checkout", "v5")
	if !found {
		t.Error("Expected to find cached entry in second workflow")
	}
	if sha != "shared-sha-123" {
		t.Errorf("Expected SHA 'shared-sha-123', got '%s'", sha)
	}
}
