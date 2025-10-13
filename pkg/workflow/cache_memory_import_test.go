package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheMemoryImportMerge(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a shared workflow directory
	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Write a shared workflow with cache-memory configuration
	sharedPath := filepath.Join(sharedDir, "cache-config.md")
	sharedContent := `---
tools:
  cache-memory:
    - id: session
      key: shared-session
---

# Shared Cache Configuration

This workflow provides session cache configuration.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Write the main workflow that imports the shared config
	mainPath := filepath.Join(tmpDir, ".github", "workflows", "main.md")
	mainContent := `---
name: Test Import Merge
on: workflow_dispatch
permissions:
  contents: read
engine: claude
imports:
  - shared/cache-config.md
tools:
  cache-memory:
    - id: default
      key: main-default
  github:
    allowed: [get_repository]
---

# Main Workflow

Test cache-memory import and merge.
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(mainPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(mainPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// When both main and imported have arrays, main takes precedence
	// So we expect only the default cache from main
	expectedStrings := []string{
		"- name: Create cache-memory directory (default)",
		"path: /tmp/gh-aw/cache-memory/default",
		"key: main-default-${{ github.run_id }}",
	}

	// These should NOT be present since main's array replaces imported array
	notExpectedStrings := []string{
		"- name: Create cache-memory directory (session)",
		"key: shared-session-${{ github.run_id }}",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(lockStr, expected) {
			t.Errorf("Expected to find '%s' in lock file but it was missing", expected)
		}
	}

	for _, notExpected := range notExpectedStrings {
		if strings.Contains(lockStr, notExpected) {
			t.Errorf("Did not expect to find '%s' in lock file but it was present", notExpected)
		}
	}
}

func TestCacheMemoryImportOnly(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a shared workflow directory
	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Write a shared workflow with cache-memory configuration
	sharedPath := filepath.Join(sharedDir, "cache-config.md")
	sharedContent := `---
tools:
  cache-memory:
    - id: session
      key: shared-session
    - id: logs
      key: shared-logs
---

# Shared Cache Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Write the main workflow that imports the shared config WITHOUT defining its own cache-memory
	mainPath := filepath.Join(tmpDir, ".github", "workflows", "main.md")
	mainContent := `---
name: Test Import Only
on: workflow_dispatch
permissions:
  contents: read
engine: claude
imports:
  - shared/cache-config.md
tools:
  github:
    allowed: [get_repository]
---

# Main Workflow

Test cache-memory import without local definition.
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(mainPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(mainPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// We expect the imported caches to be present
	expectedStrings := []string{
		"- name: Create cache-memory directory (session)",
		"path: /tmp/gh-aw/cache-memory/session",
		"key: shared-session-${{ github.run_id }}",
		"- name: Create cache-memory directory (logs)",
		"path: /tmp/gh-aw/cache-memory/logs",
		"key: shared-logs-${{ github.run_id }}",
		"## Cache Folders Available",
		"- **session**: `/tmp/gh-aw/cache-memory/session/`",
		"- **logs**: `/tmp/gh-aw/cache-memory/logs/`",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(lockStr, expected) {
			t.Errorf("Expected to find '%s' in lock file but it was missing", expected)
		}
	}
}

func TestCacheMemorySingleImportWithArrayMain(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a shared workflow directory
	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Write a shared workflow with single cache-memory configuration
	sharedPath := filepath.Join(sharedDir, "cache-single.md")
	sharedContent := `---
tools:
  cache-memory:
    key: shared-single
---

# Shared Single Cache Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Write the main workflow that imports the shared config
	mainPath := filepath.Join(tmpDir, ".github", "workflows", "main.md")
	mainContent := `---
name: Test Import Single With Array
on: workflow_dispatch
permissions:
  contents: read
engine: claude
imports:
  - shared/cache-single.md
tools:
  cache-memory:
    - id: local
      key: main-local
  github:
    allowed: [get_repository]
---

# Main Workflow

Test cache-memory import merge when shared is single and main is array.
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(mainPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(mainPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// The main workflow's array should take precedence
	// We expect to see the local cache from main
	expectedStrings := []string{
		"- name: Create cache-memory directory (local)",
		"path: /tmp/gh-aw/cache-memory/local",
		"key: main-local-${{ github.run_id }}",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(lockStr, expected) {
			t.Errorf("Expected to find '%s' in lock file but it was missing", expected)
		}
	}
}
