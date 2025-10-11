package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestScanWorkflowsDirectory(t *testing.T) {
	// Create a temporary directory for test workflows
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test workflows with MCP servers
	testWorkflow1 := `---
on:
  workflow_dispatch:

permissions: read-all

tools:
  github:
    mcp:
      type: stdio
      command: "npx"
      args: ["@github/github-mcp-server"]
      allowed: ["create_issue"]

---

# Test Workflow 1
This is a test workflow with GitHub MCP server.`

	testWorkflow2 := `---
on:
  workflow_dispatch:

permissions: read-all

mcp-servers:
  custom-server:
    type: stdio
    command: "node"
    args: ["server.js"]
    allowed: ["custom_tool"]

---

# Test Workflow 2
This is a test workflow with custom MCP server.`

	testWorkflowNoMCP := `---
on:
  workflow_dispatch:

permissions: read-all

---

# Test Workflow No MCP
This is a test workflow without MCP servers.`

	// Write test workflow files
	if err := os.WriteFile(filepath.Join(workflowsDir, "test-workflow-1.md"), []byte(testWorkflow1), 0644); err != nil {
		t.Fatalf("Failed to create test workflow 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowsDir, "test-workflow-2.md"), []byte(testWorkflow2), 0644); err != nil {
		t.Fatalf("Failed to create test workflow 2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowsDir, "no-mcp.md"), []byte(testWorkflowNoMCP), 0644); err != nil {
		t.Fatalf("Failed to create test workflow without MCP: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	t.Run("scan_all_workflows", func(t *testing.T) {
		workflows, err := scanWorkflowsDirectory(workflowsDir, "", false)
		if err != nil {
			t.Errorf("scanWorkflowsDirectory failed: %v", err)
		}

		// Should find 2 workflows with MCP servers (excluding no-mcp.md)
		if len(workflows) != 2 {
			t.Errorf("Expected 2 workflows with MCP servers, got %d", len(workflows))
		}

		// Verify workflow names
		names := make(map[string]bool)
		for _, wf := range workflows {
			names[wf.Name] = true
		}

		if !names["test-workflow-1"] || !names["test-workflow-2"] {
			t.Errorf("Expected to find test-workflow-1 and test-workflow-2, got: %v", names)
		}

		// Verify MCP configs are populated
		for _, wf := range workflows {
			if len(wf.MCPConfigs) == 0 {
				t.Errorf("Expected workflow %s to have MCP configs, but got none", wf.Name)
			}
		}
	})

	t.Run("scan_with_server_filter", func(t *testing.T) {
		workflows, err := scanWorkflowsDirectory(workflowsDir, "github", false)
		if err != nil {
			t.Errorf("scanWorkflowsDirectory with filter failed: %v", err)
		}

		// Should find only workflow 1 with github MCP server
		if len(workflows) != 1 {
			t.Errorf("Expected 1 workflow with github MCP server, got %d", len(workflows))
		}

		if len(workflows) > 0 && workflows[0].Name != "test-workflow-1" {
			t.Errorf("Expected to find test-workflow-1, got %s", workflows[0].Name)
		}
	})

	t.Run("scan_nonexistent_directory", func(t *testing.T) {
		_, err := scanWorkflowsDirectory("/nonexistent/path", "", false)
		if err == nil {
			t.Error("Expected error for nonexistent directory, got nil")
		}
	})

	t.Run("scan_verbose_mode", func(t *testing.T) {
		// Create a workflow with invalid frontmatter
		invalidWorkflow := `---
invalid yaml: [
---`
		if err := os.WriteFile(filepath.Join(workflowsDir, "invalid.md"), []byte(invalidWorkflow), 0644); err != nil {
			t.Fatalf("Failed to create invalid workflow: %v", err)
		}

		// Should not error, just skip invalid workflows
		workflows, err := scanWorkflowsDirectory(workflowsDir, "", true)
		if err != nil {
			t.Errorf("scanWorkflowsDirectory verbose failed: %v", err)
		}

		// Should still find the 2 valid workflows
		if len(workflows) != 2 {
			t.Errorf("Expected 2 valid workflows in verbose mode, got %d", len(workflows))
		}
	})
}

func TestLoadWorkflowWithMCP(t *testing.T) {
	// Create a temporary directory for test workflows
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a test workflow with MCP servers
	testWorkflowContent := `---
on:
  workflow_dispatch:

permissions: read-all

tools:
  github:
    mcp:
      type: stdio
      command: "npx"
      args: ["@github/github-mcp-server"]
      allowed: ["create_issue", "add_comment"]

mcp-servers:
  custom-server:
    type: stdio
    command: "node"
    args: ["server.js"]
    allowed: ["custom_tool"]

---

# Test Workflow
This is a test workflow with multiple MCP servers.`

	testWorkflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(testWorkflowPath, []byte(testWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	t.Run("load_workflow_no_filter", func(t *testing.T) {
		info, err := loadWorkflowWithMCP("test-workflow", "")
		if err != nil {
			t.Errorf("loadWorkflowWithMCP failed: %v", err)
		}

		if info == nil {
			t.Fatal("Expected workflow info, got nil")
		}

		if info.Name != "test-workflow" {
			t.Errorf("Expected workflow name 'test-workflow', got '%s'", info.Name)
		}

		// Should have 2 MCP servers (github and custom-server)
		if len(info.MCPConfigs) != 2 {
			t.Errorf("Expected 2 MCP servers, got %d", len(info.MCPConfigs))
		}

		// Verify frontmatter is populated
		if len(info.Frontmatter) == 0 {
			t.Error("Expected frontmatter to be populated")
		}
	})

	t.Run("load_workflow_with_filter", func(t *testing.T) {
		info, err := loadWorkflowWithMCP("test-workflow", "github")
		if err != nil {
			t.Errorf("loadWorkflowWithMCP with filter failed: %v", err)
		}

		if info == nil {
			t.Fatal("Expected workflow info, got nil")
		}

		// Should have only 1 MCP server (github)
		if len(info.MCPConfigs) != 1 {
			t.Errorf("Expected 1 MCP server with filter, got %d", len(info.MCPConfigs))
		}

		if len(info.MCPConfigs) > 0 && info.MCPConfigs[0].Name != "github" {
			t.Errorf("Expected github MCP server, got %s", info.MCPConfigs[0].Name)
		}
	})

	t.Run("load_nonexistent_workflow", func(t *testing.T) {
		_, err := loadWorkflowWithMCP("nonexistent", "")
		if err == nil {
			t.Error("Expected error for nonexistent workflow, got nil")
		}
	})

	t.Run("load_workflow_with_extension", func(t *testing.T) {
		info, err := loadWorkflowWithMCP("test-workflow.md", "")
		if err != nil {
			t.Errorf("loadWorkflowWithMCP with .md extension failed: %v", err)
		}

		if info == nil {
			t.Fatal("Expected workflow info, got nil")
		}

		if info.Name != "test-workflow" {
			t.Errorf("Expected workflow name 'test-workflow', got '%s'", info.Name)
		}
	})
}
