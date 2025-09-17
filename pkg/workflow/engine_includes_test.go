package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineInheritanceFromIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create include file with engine specification
	includeContent := `---
engine: codex
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Engine
This include specifies the codex engine.
`
	includeFile := filepath.Join(workflowsDir, "include-with-engine.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow without engine specification
	mainContent := `---
on: push
---

# Main Workflow Without Engine

@include include-with-engine.md

This should inherit the engine from the included file.
`
	mainFile := filepath.Join(workflowsDir, "main-inherit-engine.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Expected successful compilation, got error: %v", err)
	}

	// Check that lock file was created
	lockFile := filepath.Join(workflowsDir, "main-inherit-engine.lock.yml")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}

	// Verify lock file contains codex engine configuration
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	lockStr := string(lockContent)

	// Should contain references to codex installation and execution
	if !strings.Contains(lockStr, "Install Codex") {
		t.Error("Expected lock file to contain 'Install Codex' step")
	}
	if !strings.Contains(lockStr, "codex login") {
		t.Error("Expected lock file to contain 'codex login' command")
	}
}

func TestEngineConflictDetection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create include file with codex engine
	includeContent := `---
engine: codex
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Codex Engine
`
	includeFile := filepath.Join(workflowsDir, "include-codex.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow with claude engine (conflict)
	mainContent := `---
on: push
engine: claude
---

# Main Workflow with Claude Engine

@include include-codex.md

This should fail due to multiple engine specifications.
`
	mainFile := filepath.Join(workflowsDir, "main-conflict.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow - should fail
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)
	if err == nil {
		t.Fatal("Expected compilation to fail due to multiple engine specifications")
	}

	// Check error message contains expected content
	errMsg := err.Error()
	if !strings.Contains(errMsg, "multiple engine fields found") {
		t.Errorf("Expected error message to contain 'multiple engine fields found', got: %s", errMsg)
	}
}

func TestEngineObjectFormatInIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create include file with object-format engine specification
	includeContent := `---
engine:
  id: claude
  model: claude-3-5-sonnet-20241022
  max-turns: 5
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Object Engine
`
	includeFile := filepath.Join(workflowsDir, "include-object-engine.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow without engine specification
	mainContent := `---
on: push
---

# Main Workflow

@include include-object-engine.md

This should inherit the claude engine from the included file.
`
	mainFile := filepath.Join(workflowsDir, "main-object-engine.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Expected successful compilation, got error: %v", err)
	}

	// Check that lock file was created
	lockFile := filepath.Join(workflowsDir, "main-object-engine.lock.yml")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}
}

func TestNoEngineSpecifiedAnywhere(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create include file without engine specification
	includeContent := `---
tools:
  github:
    allowed: ["list_issues"]
---

# Include without Engine
`
	includeFile := filepath.Join(workflowsDir, "include-no-engine.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow without engine specification
	mainContent := `---
on: push
---

# Main Workflow without Engine

@include include-no-engine.md

This should use the default engine.
`
	mainFile := filepath.Join(workflowsDir, "main-default.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Expected successful compilation, got error: %v", err)
	}

	// Check that lock file was created
	lockFile := filepath.Join(workflowsDir, "main-default.lock.yml")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}

	// Verify lock file contains default claude engine configuration
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	lockStr := string(lockContent)

	// Should contain references to claude CLI (default engine)
	if !strings.Contains(lockStr, "npx @anthropic-ai/claude-code") {
		t.Error("Expected lock file to contain claude CLI reference")
	}
}

func TestMainEngineWithoutIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create main workflow with claude engine (no includes, so no conflict)
	mainContent := `---
on: push
engine: claude
---

# Main Workflow with Claude Engine

This workflow specifies claude engine directly without any includes.
`
	mainFile := filepath.Join(workflowsDir, "main-claude.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow (no includes, so no conflict)
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Expected successful compilation, got error: %v", err)
	}

	// Check that lock file contains claude engine
	lockFile := filepath.Join(workflowsDir, "main-claude.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	lockStr := string(lockContent)

	// Should contain references to claude CLI
	if !strings.Contains(lockStr, "npx @anthropic-ai/claude-code") {
		t.Error("Expected lock file to contain claude CLI reference")
	}
}

func TestMultipleIncludesWithEnginesFailure(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create first include file with codex engine
	includeContent1 := `---
engine: codex
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Codex Engine
`
	includeFile1 := filepath.Join(workflowsDir, "include-codex.md")
	if err := os.WriteFile(includeFile1, []byte(includeContent1), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with claude engine
	includeContent2 := `---
engine: claude
tools:
  github:
    allowed: ["create_issue"]
---

# Include with Claude Engine
`
	includeFile2 := filepath.Join(workflowsDir, "include-claude.md")
	if err := os.WriteFile(includeFile2, []byte(includeContent2), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow without engine specification but with multiple includes
	mainContent := `---
on: push
---

# Main Workflow

@include include-codex.md
@include include-claude.md

This should fail due to multiple engine specifications in includes.
`
	mainFile := filepath.Join(workflowsDir, "main-multiple-engines.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow - should fail
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)
	if err == nil {
		t.Fatal("Expected compilation to fail due to multiple engine specifications")
	}

	// Check error message contains expected content
	errMsg := err.Error()
	if !strings.Contains(errMsg, "multiple engine fields found") {
		t.Errorf("Expected error message to contain 'multiple engine fields found', got: %s", errMsg)
	}
}
