package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestCheckExistingPRImport(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create shared directory
	sharedDir := filepath.Join(tempDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create the check-existing-pr.md shared file
	sharedFilePath := filepath.Join(sharedDir, "check-existing-pr.md")
	sharedFileContent := `---
tools:
  github:
    allowed:
      - search_pull_requests
      - list_pull_requests
---

## Check for Existing Pull Request

Instructions for checking existing PRs.
`
	if err := os.WriteFile(sharedFilePath, []byte(sharedFileContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create a workflow file that imports check-existing-pr
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/check-existing-pr.md
tools:
  edit:
safe-outputs:
  create-issue:
    title-prefix: "[test] "
    labels: [automation, test]
---

# Test Workflow

Test workflow that uses check-existing-pr shared workflow.
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

	// Verify that the GitHub tools are included in the allowed tools
	if !strings.Contains(workflowData, "mcp__github__search_pull_requests") {
		t.Error("Expected compiled workflow to include search_pull_requests in allowed tools")
	}

	if !strings.Contains(workflowData, "mcp__github__list_pull_requests") {
		t.Error("Expected compiled workflow to include list_pull_requests in allowed tools")
	}

	// Verify the instructions from the shared file are included
	if !strings.Contains(workflowData, "Check for Existing Pull Request") {
		t.Error("Expected compiled workflow to include instructions from shared file")
	}

	// Verify GitHub MCP server configuration is present
	if !strings.Contains(workflowData, "github") {
		t.Error("Expected compiled workflow to contain GitHub MCP server configuration")
	}
}

func TestCheckExistingPRImportWithCopilot(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create shared directory
	sharedDir := filepath.Join(tempDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create the check-existing-pr.md shared file
	sharedFilePath := filepath.Join(sharedDir, "check-existing-pr.md")
	sharedFileContent := `---
tools:
  github:
    allowed:
      - search_pull_requests
      - list_pull_requests
---

## Check for Existing Pull Request

Instructions for checking existing PRs.
`
	if err := os.WriteFile(sharedFilePath, []byte(sharedFileContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create a workflow file that imports check-existing-pr
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - shared/check-existing-pr.md
tools:
  edit:
safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
    labels: [automation, test]
---

# Test Workflow

Test workflow that uses check-existing-pr shared workflow with Copilot.
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

	// For Copilot, verify that tools array is present in MCP config
	if !strings.Contains(workflowData, `"tools":`) {
		t.Error("Expected Copilot workflow to include tools array in MCP config")
	}

	if !strings.Contains(workflowData, `"search_pull_requests"`) {
		t.Error("Expected Copilot workflow to include search_pull_requests in tools array")
	}

	if !strings.Contains(workflowData, `"list_pull_requests"`) {
		t.Error("Expected Copilot workflow to include list_pull_requests in tools array")
	}

	// Verify the instructions from the shared file are included
	if !strings.Contains(workflowData, "Check for Existing Pull Request") {
		t.Error("Expected compiled workflow to include instructions from shared file")
	}
}
