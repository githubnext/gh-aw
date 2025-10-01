package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWebFetchMCPServerAddition tests that when a Codex workflow uses web-fetch,
// the mcp/fetch server is automatically added
func TestWebFetchMCPServerAddition(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that uses web-fetch with Codex engine (which doesn't support web-fetch natively)
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: codex
tools:
  web-fetch:
---

# Test Workflow

Fetch content from the web.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Verify that the compiled workflow contains the mcp/fetch server configuration
	lockContent := string(lockData)

	// The TOML config should contain the mcp/fetch server
	if !strings.Contains(lockContent, `[mcp_servers."mcp/fetch"]`) {
		t.Errorf("Expected compiled workflow to contain mcp/fetch server configuration, but it didn't")
	}

	// Verify the Docker command is present
	if !strings.Contains(lockContent, `ghcr.io/modelcontextprotocol/servers/fetch:latest`) {
		t.Errorf("Expected mcp/fetch server to use the Docker image, but it didn't")
	}

	// Verify that web-fetch is no longer in the tools section (it should be replaced by mcp/fetch)
	// This is harder to verify directly, but we can check that the mcp/fetch server is configured
	if !strings.Contains(lockContent, `command = "docker"`) {
		t.Errorf("Expected mcp/fetch server to have Docker command")
	}
}

// TestWebFetchNotAddedForClaudeEngine tests that when a Claude workflow uses web-fetch,
// the mcp/fetch server is NOT added (because Claude has native support)
func TestWebFetchNotAddedForClaudeEngine(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that uses web-fetch with Claude engine (which supports web-fetch natively)
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
tools:
  web-fetch:
---

# Test Workflow

Fetch content from the web.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Verify that the compiled workflow does NOT contain the mcp/fetch server configuration
	lockContent := string(lockData)

	// Claude uses JSON format, so check for JSON-style mcp/fetch
	if strings.Contains(lockContent, `"mcp/fetch"`) {
		t.Errorf("Expected Claude workflow NOT to contain mcp/fetch server (since Claude has native web-fetch support), but it did")
	}

	// Instead, Claude should have the WebFetch tool in its allowed tools list
	if !strings.Contains(lockContent, "WebFetch") {
		t.Errorf("Expected Claude workflow to have WebFetch in allowed tools, but it didn't")
	}
}

// TestNoWebFetchNoMCPFetchServer tests that when a workflow doesn't use web-fetch,
// the mcp/fetch server is not added
func TestNoWebFetchNoMCPFetchServer(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow that doesn't use web-fetch
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: codex
tools:
  bash:
    - echo
---

# Test Workflow

Run some bash commands.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a compiler
	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Verify that the compiled workflow does NOT contain the mcp/fetch server configuration
	lockContent := string(lockData)

	if strings.Contains(lockContent, `mcp/fetch`) {
		t.Errorf("Expected workflow without web-fetch NOT to contain mcp/fetch server, but it did")
	}
}
