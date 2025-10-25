package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestNetworkMergeMultipleImports(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create first shared file with network configuration
	shared1Path := filepath.Join(tempDir, "shared-python.md")
	shared1Content := `---
network:
  allowed:
    - python
---

# Python Network Configuration

Provides network access to Python package indexes.
`
	if err := os.WriteFile(shared1Path, []byte(shared1Content), 0644); err != nil {
		t.Fatalf("Failed to write shared-python file: %v", err)
	}

	// Create second shared file with network configuration
	shared2Path := filepath.Join(tempDir, "shared-node.md")
	shared2Content := `---
network:
  allowed:
    - node
---

# Node Network Configuration

Provides network access to Node.js package registries.
`
	if err := os.WriteFile(shared2Path, []byte(shared2Content), 0644); err != nil {
		t.Fatalf("Failed to write shared-node file: %v", err)
	}

	// Create a workflow file that imports both shared files and has its own network config
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
engine: claude
network:
  allowed:
    - defaults
    - github.com
imports:
  - shared-python.md
  - shared-node.md
---

# Test Workflow with Multiple Network Imports

This workflow should have merged network domains from multiple sources.
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
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Check for presence of ALLOWED_DOMAINS
	if !strings.Contains(workflowData, "ALLOWED_DOMAINS") {
		t.Fatal("Expected ALLOWED_DOMAINS to be present in compiled workflow")
	}

	// Should contain github.com from top-level
	if !strings.Contains(workflowData, "github.com") {
		t.Error("Expected github.com from top-level network config")
	}

	// Should contain PyPI domains from python ecosystem
	if !strings.Contains(workflowData, "pypi.org") {
		t.Error("Expected pypi.org from python ecosystem")
	}

	// Should contain NPM registry from node ecosystem
	if !strings.Contains(workflowData, "registry.npmjs.org") {
		t.Error("Expected registry.npmjs.org from node ecosystem")
	}

	// Should contain default domains
	if !strings.Contains(workflowData, "json-schema.org") {
		t.Error("Expected json-schema.org from defaults")
	}

	t.Log("✓ All network domains successfully merged from multiple imports")
}
