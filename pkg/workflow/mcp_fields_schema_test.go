package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestMCPFieldsInIncludedFiles tests that entrypointArgs, headers, and url
// fields can be used in included files without schema validation errors
func TestMCPFieldsInIncludedFiles(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an included file with MCP server using all three fields
	includedFilePath := filepath.Join(tempDir, "mcp-with-fields.md")
	includedFileContent := `---
mcp-servers:
  test-server:
    type: stdio
    container: "test/mcp-server"
    entrypointArgs: ["--arg1", "value1"]
    url: "https://example.com/mcp"
    headers:
      Authorization: "Bearer token"
      X-Custom-Header: "custom-value"
    allowed: ["*"]
---
`
	if err := os.WriteFile(includedFilePath, []byte(includedFileContent), 0644); err != nil {
		t.Fatalf("Failed to write included file: %v", err)
	}

	// Create a workflow file that imports the MCP configuration
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - mcp-with-fields.md
---

# Test Workflow

This workflow imports an MCP server with entrypointArgs, headers, and url fields.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow - should succeed without schema validation errors
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed with schema validation error: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify that the compiled workflow contains the MCP server
	if !strings.Contains(workflowData, "test-server") {
		t.Error("Expected compiled workflow to contain test-server MCP configuration")
	}
}

// TestEntrypointArgsInIncludedFile specifically tests entrypointArgs field
func TestEntrypointArgsInIncludedFile(t *testing.T) {
	tempDir := t.TempDir()

	includedFilePath := filepath.Join(tempDir, "mcp-entrypoint.md")
	includedFileContent := `---
mcp-servers:
  entrypoint-test:
    type: stdio
    container: "test/server"
    entrypointArgs: ["--config", "/path/to/config", "--verbose"]
    allowed: ["test_function"]
---
`
	if err := os.WriteFile(includedFilePath, []byte(includedFileContent), 0644); err != nil {
		t.Fatalf("Failed to write included file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "workflow.md")
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - mcp-entrypoint.md
---

# Test entrypointArgs
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Verify lock file was created
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}
}

// TestHeadersInIncludedFile specifically tests headers field
func TestHeadersInIncludedFile(t *testing.T) {
	tempDir := t.TempDir()

	includedFilePath := filepath.Join(tempDir, "mcp-headers.md")
	includedFileContent := `---
mcp-servers:
  headers-test:
    type: http
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer secret-token"
      X-API-Key: "api-key-value"
      Content-Type: "application/json"
    allowed: ["get_data"]
---
`
	if err := os.WriteFile(includedFilePath, []byte(includedFileContent), 0644); err != nil {
		t.Fatalf("Failed to write included file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "workflow.md")
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - mcp-headers.md
---

# Test headers
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Verify lock file was created
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}
}

// TestURLInIncludedFile specifically tests url field
func TestURLInIncludedFile(t *testing.T) {
	tempDir := t.TempDir()

	includedFilePath := filepath.Join(tempDir, "mcp-url.md")
	includedFileContent := `---
mcp-servers:
  url-test:
    type: http
    url: "https://mcp.service.com/api/v1"
    allowed: ["fetch_resource"]
---
`
	if err := os.WriteFile(includedFilePath, []byte(includedFileContent), 0644); err != nil {
		t.Fatalf("Failed to write included file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "workflow.md")
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - mcp-url.md
---

# Test url
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Verify lock file was created
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatal("Expected lock file to be created")
	}
}
