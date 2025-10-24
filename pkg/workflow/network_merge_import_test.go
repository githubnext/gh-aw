package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestNetworkMergeWithImports(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a shared file with network configuration
	sharedNetworkPath := filepath.Join(tempDir, "shared-network.md")
	sharedNetworkContent := `---
network:
  allowed:
    - example.com
    - api.example.com
---

# Shared Network Configuration

This file provides network access to example.com domains.
`
	if err := os.WriteFile(sharedNetworkPath, []byte(sharedNetworkContent), 0644); err != nil {
		t.Fatalf("Failed to write shared network file: %v", err)
	}

	// Create a workflow file that imports the shared network and has its own network config
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
  - shared-network.md
---

# Test Workflow

This workflow should have merged network domains.
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

	// The compiled workflow should contain all network domains from both files
	expectedDomains := []string{
		"example.com",
		"api.example.com",
		"github.com",
	}

	for _, domain := range expectedDomains {
		if !strings.Contains(workflowData, domain) {
			t.Errorf("Expected compiled workflow to contain domain %s, but it was not found", domain)
		}
	}

	// Should also have defaults expanded
	if !strings.Contains(workflowData, "ALLOWED_DOMAINS") {
		t.Error("Expected compiled workflow to contain ALLOWED_DOMAINS configuration")
	}
}
