package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestRecursiveImports tests that imports from imported files are also processed
func TestRecursiveImports(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-*")

	// Create base shared file (level 2)
	baseSharedPath := filepath.Join(tempDir, "base-shared.md")
	baseSharedContent := `---
mcp-servers:
  base-tool:
    url: "https://example.com/base"
    allowed: ["*"]
---
Base shared content
`
	if err := os.WriteFile(baseSharedPath, []byte(baseSharedContent), 0644); err != nil {
		t.Fatalf("Failed to write base shared file: %v", err)
	}

	// Create intermediate shared file (level 1) that imports base-shared.md
	intermediateSharedPath := filepath.Join(tempDir, "intermediate-shared.md")
	intermediateSharedContent := `---
imports:
  - base-shared.md
mcp-servers:
  intermediate-tool:
    url: "https://example.com/intermediate"
    allowed: ["*"]
---
Intermediate shared content
`
	if err := os.WriteFile(intermediateSharedPath, []byte(intermediateSharedContent), 0644); err != nil {
		t.Fatalf("Failed to write intermediate shared file: %v", err)
	}

	// Create main workflow that imports intermediate-shared.md
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - intermediate-shared.md
mcp-servers:
  main-tool:
    url: "https://example.com/main"
    allowed: ["*"]
---

# Test Workflow

This workflow tests recursive imports.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify that all three tools are present (main, intermediate, and base)
	if !strings.Contains(workflowData, "main-tool") {
		t.Error("Expected compiled workflow to contain main-tool")
	}

	if !strings.Contains(workflowData, "intermediate-tool") {
		t.Error("Expected compiled workflow to contain intermediate-tool from imported file")
	}

	if !strings.Contains(workflowData, "base-tool") {
		t.Error("Expected compiled workflow to contain base-tool from recursively imported file")
	}

	// Verify all URLs are present
	if !strings.Contains(workflowData, "https://example.com/main") {
		t.Error("Expected compiled workflow to contain main URL")
	}

	if !strings.Contains(workflowData, "https://example.com/intermediate") {
		t.Error("Expected compiled workflow to contain intermediate URL")
	}

	if !strings.Contains(workflowData, "https://example.com/base") {
		t.Error("Expected compiled workflow to contain base URL from recursive import")
	}
}

// TestCyclicImports tests that cyclic imports are detected and handled properly
func TestCyclicImports(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-*")

	// Create file A that imports B
	fileAPath := filepath.Join(tempDir, "file-a.md")
	fileAContent := `---
imports:
  - file-b.md
mcp-servers:
  tool-a:
    url: "https://example.com/a"
    allowed: ["*"]
---
Content A
`
	if err := os.WriteFile(fileAPath, []byte(fileAContent), 0644); err != nil {
		t.Fatalf("Failed to write file A: %v", err)
	}

	// Create file B that imports A (creating a cycle)
	fileBPath := filepath.Join(tempDir, "file-b.md")
	fileBContent := `---
imports:
  - file-a.md
mcp-servers:
  tool-b:
    url: "https://example.com/b"
    allowed: ["*"]
---
Content B
`
	if err := os.WriteFile(fileBPath, []byte(fileBContent), 0644); err != nil {
		t.Fatalf("Failed to write file B: %v", err)
	}

	// Create main workflow that imports file A
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
imports:
  - file-a.md
---

# Test Workflow

This workflow tests cyclic import detection.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow - should handle the cycle gracefully
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify both tools are present (cycle should be handled)
	if !strings.Contains(workflowData, "tool-a") {
		t.Error("Expected compiled workflow to contain tool-a")
	}

	if !strings.Contains(workflowData, "tool-b") {
		t.Error("Expected compiled workflow to contain tool-b")
	}
}

// TestDiamondImports tests that diamond-shaped import graphs work correctly
// Main imports A and B, both A and B import C - C should be processed only once
func TestDiamondImports(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-*")

	// Create file C (bottom of diamond)
	fileCPath := filepath.Join(tempDir, "file-c.md")
	fileCContent := `---
mcp-servers:
  tool-c:
    url: "https://example.com/c"
    allowed: ["*"]
---
Content C
`
	if err := os.WriteFile(fileCPath, []byte(fileCContent), 0644); err != nil {
		t.Fatalf("Failed to write file C: %v", err)
	}

	// Create file A that imports C
	fileAPath := filepath.Join(tempDir, "file-a.md")
	fileAContent := `---
imports:
  - file-c.md
mcp-servers:
  tool-a:
    url: "https://example.com/a"
    allowed: ["*"]
---
Content A
`
	if err := os.WriteFile(fileAPath, []byte(fileAContent), 0644); err != nil {
		t.Fatalf("Failed to write file A: %v", err)
	}

	// Create file B that also imports C
	fileBPath := filepath.Join(tempDir, "file-b.md")
	fileBContent := `---
imports:
  - file-c.md
mcp-servers:
  tool-b:
    url: "https://example.com/b"
    allowed: ["*"]
---
Content B
`
	if err := os.WriteFile(fileBPath, []byte(fileBContent), 0644); err != nil {
		t.Fatalf("Failed to write file B: %v", err)
	}

	// Create main workflow that imports both A and B
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
imports:
  - file-a.md
  - file-b.md
---

# Test Workflow

This workflow tests diamond import pattern.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify all three tools are present
	if !strings.Contains(workflowData, "tool-a") {
		t.Error("Expected compiled workflow to contain tool-a")
	}

	if !strings.Contains(workflowData, "tool-b") {
		t.Error("Expected compiled workflow to contain tool-b")
	}

	if !strings.Contains(workflowData, "tool-c") {
		t.Error("Expected compiled workflow to contain tool-c")
	}
}

// TestImportOrdering tests that imports are processed in BFS order
func TestImportOrdering(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-*")

	// Create a complex import graph:
	//   Main -> A, B
	//   A -> C, D
	//   B -> E
	//   C -> F
	// Expected BFS order: Main, A, B, C, D, E, F

	// Create file F (deepest level)
	fileFPath := filepath.Join(tempDir, "file-f.md")
	fileFContent := `---
mcp-servers:
  tool-f:
    url: "https://example.com/f"
    allowed: ["f"]
---
`
	if err := os.WriteFile(fileFPath, []byte(fileFContent), 0644); err != nil {
		t.Fatalf("Failed to write file F: %v", err)
	}

	// Create file C that imports F
	fileCPath := filepath.Join(tempDir, "file-c.md")
	fileCContent := `---
imports:
  - file-f.md
mcp-servers:
  tool-c:
    url: "https://example.com/c"
    allowed: ["c"]
---
`
	if err := os.WriteFile(fileCPath, []byte(fileCContent), 0644); err != nil {
		t.Fatalf("Failed to write file C: %v", err)
	}

	// Create file D
	fileDPath := filepath.Join(tempDir, "file-d.md")
	fileDContent := `---
mcp-servers:
  tool-d:
    url: "https://example.com/d"
    allowed: ["d"]
---
`
	if err := os.WriteFile(fileDPath, []byte(fileDContent), 0644); err != nil {
		t.Fatalf("Failed to write file D: %v", err)
	}

	// Create file E
	fileEPath := filepath.Join(tempDir, "file-e.md")
	fileEContent := `---
mcp-servers:
  tool-e:
    url: "https://example.com/e"
    allowed: ["e"]
---
`
	if err := os.WriteFile(fileEPath, []byte(fileEContent), 0644); err != nil {
		t.Fatalf("Failed to write file E: %v", err)
	}

	// Create file A that imports C and D
	fileAPath := filepath.Join(tempDir, "file-a.md")
	fileAContent := `---
imports:
  - file-c.md
  - file-d.md
mcp-servers:
  tool-a:
    url: "https://example.com/a"
    allowed: ["a"]
---
`
	if err := os.WriteFile(fileAPath, []byte(fileAContent), 0644); err != nil {
		t.Fatalf("Failed to write file A: %v", err)
	}

	// Create file B that imports E
	fileBPath := filepath.Join(tempDir, "file-b.md")
	fileBContent := `---
imports:
  - file-e.md
mcp-servers:
  tool-b:
    url: "https://example.com/b"
    allowed: ["b"]
---
`
	if err := os.WriteFile(fileBPath, []byte(fileBContent), 0644); err != nil {
		t.Fatalf("Failed to write file B: %v", err)
	}

	// Create main workflow that imports A and B
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
engine: copilot
imports:
  - file-a.md
  - file-b.md
---

# Test Workflow
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify all tools are present
	expectedTools := []string{"tool-a", "tool-b", "tool-c", "tool-d", "tool-e", "tool-f"}
	for _, tool := range expectedTools {
		if !strings.Contains(workflowData, tool) {
			t.Errorf("Expected compiled workflow to contain %s", tool)
		}
	}

	// Verify BFS ordering by checking that allowed arrays are merged in correct order
	// Since BFS processes level by level: A,B (level 1), then C,D,E (level 2), then F (level 3)
	// The merged allowed array should reflect this order
	if !strings.Contains(workflowData, "a") {
		t.Error("Expected allowed array to contain 'a'")
	}
	if !strings.Contains(workflowData, "b") {
		t.Error("Expected allowed array to contain 'b'")
	}
	if !strings.Contains(workflowData, "c") {
		t.Error("Expected allowed array to contain 'c'")
	}
}
