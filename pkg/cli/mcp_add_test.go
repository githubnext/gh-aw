package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddMCPTool_BasicFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
name: Test Workflow
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
---

# Test Workflow

This is a test workflow.
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a mock registry server
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/servers/search") {
			// Return mock search response
			response := `{
				"servers": [
					{
						"id": "notion-mcp",
						"name": "Notion MCP Server",
						"description": "MCP server for Notion integration",
						"repository": "https://github.com/example/notion-mcp",
						"command": "npx",
						"args": ["notion-mcp"],
						"transport": "stdio",
						"config": {
							"env": {
								"NOTION_TOKEN": "${NOTION_TOKEN}"
							}
						}
					}
				],
				"total": 1
			}`

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		}
	}))
	defer registryServer.Close()

	// Test adding MCP tool
	err = AddMCPTool("test-workflow", "notion", registryServer.URL, "", "", false)
	if err != nil {
		t.Fatalf("AddMCPTool failed: %v", err)
	}

	// Read the updated workflow file
	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read updated workflow: %v", err)
	}

	updatedContentStr := string(updatedContent)

	// Check that the notion-mcp tool was added
	if !strings.Contains(updatedContentStr, "notion-mcp:") {
		t.Error("Expected notion-mcp tool to be added to workflow")
	}

	// Check that it has MCP configuration
	if !strings.Contains(updatedContentStr, "mcp:") {
		t.Error("Expected MCP configuration to be added")
	}

	// Check that it has the correct transport type
	if !strings.Contains(updatedContentStr, "type: stdio") {
		t.Error("Expected stdio transport type")
	}

	// Check that it has the correct command
	if !strings.Contains(updatedContentStr, "command: npx") {
		t.Error("Expected npx command")
	}
}

func TestAddMCPTool_WorkflowNotFound(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a mock registry server
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"servers": [], "total": 0}`))
	}))
	defer registryServer.Close()

	// Test with nonexistent workflow
	err = AddMCPTool("nonexistent-workflow", "notion", registryServer.URL, "", "", false)
	if err == nil {
		t.Fatal("Expected error for nonexistent workflow, got nil")
	}

	if !strings.Contains(err.Error(), "workflow file not found") {
		t.Errorf("Expected 'workflow file not found' error, got: %v", err)
	}
}

func TestAddMCPTool_ToolAlreadyExists(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file with existing notion-mcp tool
	workflowContent := `---
name: Test Workflow
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
  notion-mcp:
    mcp:
      type: stdio
      command: npx
      args: ["notion-mcp"]
---

# Test Workflow

This is a test workflow.
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a mock registry server
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"servers": [
				{
					"id": "notion-mcp",
					"name": "Notion MCP Server",
					"transport": "stdio"
				}
			],
			"total": 1
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer registryServer.Close()

	// Test adding tool that already exists
	err = AddMCPTool("test-workflow", "notion-mcp", registryServer.URL, "", "", false)
	if err == nil {
		t.Fatal("Expected error for existing tool, got nil")
	}

	if !strings.Contains(err.Error(), "tool 'notion-mcp' already exists") {
		t.Errorf("Expected 'tool already exists' error, got: %v", err)
	}
}

func TestAddMCPTool_CustomToolID(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
name: Test Workflow
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
---

# Test Workflow

This is a test workflow.
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Create a mock registry server
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"servers": [
				{
					"id": "notion-mcp",
					"name": "Notion MCP Server",
					"transport": "stdio",
					"command": "npx",
					"args": ["notion-mcp"],
					"config": {}
				}
			],
			"total": 1
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer registryServer.Close()

	// Test adding tool with custom ID
	customToolID := "my-notion"
	err = AddMCPTool("test-workflow", "notion", registryServer.URL, "", customToolID, false)
	if err != nil {
		t.Fatalf("AddMCPTool failed: %v", err)
	}

	// Read the updated workflow file
	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read updated workflow: %v", err)
	}

	updatedContentStr := string(updatedContent)

	// Check that the custom tool ID was used
	if !strings.Contains(updatedContentStr, "my-notion:") {
		t.Error("Expected custom tool ID 'my-notion' to be used")
	}

	// Check that the original ID is not used
	if strings.Contains(updatedContentStr, "notion-mcp:") {
		t.Error("Expected original tool ID 'notion-mcp' not to be used")
	}
}

func TestCreateMCPToolConfig_StdioTransport(t *testing.T) {
	server := &MCPRegistryServer{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"test-server"},
		Config: map[string]interface{}{
			"env": map[string]interface{}{
				"TEST_TOKEN": "${TEST_TOKEN}",
			},
		},
	}

	config, err := createMCPToolConfig(server, "", false)
	if err != nil {
		t.Fatalf("createMCPToolConfig failed: %v", err)
	}

	mcpSection, ok := config["mcp"].(map[string]any)
	if !ok {
		t.Fatal("Expected mcp section to be a map")
	}

	if mcpSection["type"] != "stdio" {
		t.Errorf("Expected type 'stdio', got '%v'", mcpSection["type"])
	}

	if mcpSection["command"] != "npx" {
		t.Errorf("Expected command 'npx', got '%v'", mcpSection["command"])
	}

	args, ok := mcpSection["args"].([]string)
	if !ok || len(args) != 1 || args[0] != "test-server" {
		t.Errorf("Expected args ['test-server'], got %v", mcpSection["args"])
	}
}

func TestCreateMCPToolConfig_PreferredTransport(t *testing.T) {
	server := &MCPRegistryServer{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"test-server"},
		Config: map[string]interface{}{
			"container": "test-image:latest",
		},
	}

	// Test with preferred docker transport
	config, err := createMCPToolConfig(server, "docker", false)
	if err != nil {
		t.Fatalf("createMCPToolConfig failed: %v", err)
	}

	mcpSection, ok := config["mcp"].(map[string]any)
	if !ok {
		t.Fatal("Expected mcp section to be a map")
	}

	if mcpSection["type"] != "docker" {
		t.Errorf("Expected type 'docker', got '%v'", mcpSection["type"])
	}
}
