package cli

import (
	"encoding/json"
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
		if r.URL.Path == "/servers" {
			// Return mock search response with new structure
			response := `{
				"servers": [
					{
						"server": {
							"id": "notion-mcp",
							"name": "Notion MCP Server",
							"description": "MCP server for Notion integration",
							"repository": {
								"url": "https://github.com/example/notion-mcp"
							},
							"packages": [
								{
									"registry_name": "notion-mcp",
									"runtime_hint": "node",
									"runtime_arguments": [
										{"value": "notion-mcp"}
									],
									"environment_variables": [
										{
											"NOTION_TOKEN": "${NOTION_TOKEN}"
										}
									]
								}
							]
						}
					}
				]
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

	// Check that the Notion MCP Server tool was added (cleaned from notion-mcp server name)
	if !strings.Contains(updatedContentStr, "Notion MCP Server:") {
		t.Error("Expected Notion MCP Server tool to be added to workflow")
	}

	// Check that it has MCP configuration
	if !strings.Contains(updatedContentStr, "mcp:") {
		t.Error("Expected MCP configuration to be added")
	}

	// Check that it has the correct transport type
	if !strings.Contains(updatedContentStr, "type: stdio") {
		t.Error("Expected stdio transport type")
	}

	// Check that it has the correct command (now uses registry_name)
	if !strings.Contains(updatedContentStr, "command: notion-mcp") {
		t.Error("Expected notion-mcp command")
	}

	// Check that environment variables use GitHub Actions syntax
	if !strings.Contains(updatedContentStr, "${{ secrets.NOTION_TOKEN }}") {
		t.Logf("Workflow content: %s", updatedContentStr)
		t.Error("Expected GitHub Actions syntax for environment variables")
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

	// Create a test workflow file with existing Notion MCP Server tool
	workflowContent := `---
name: Test Workflow
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
  Notion MCP Server:
    mcp:
      type: stdio
      command: notion-mcp
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
					"server": {
						"id": "notion-mcp",
						"name": "Notion MCP Server",
						"repository": {
							"url": "https://github.com/example/notion-mcp"
						},
						"packages": [
							{
								"registry_name": "notion-mcp",
								"runtime_hint": "node",
								"runtime_arguments": [],
								"environment_variables": []
							}
						]
					}
				}
			]
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

	if !strings.Contains(err.Error(), "tool 'Notion MCP Server' already exists") {
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
					"server": {
						"id": "notion-mcp",
						"name": "Notion MCP Server",
						"repository": {
							"url": "https://github.com/example/notion-mcp"
						},
						"packages": [
							{
								"registry_name": "notion-mcp",
								"runtime_hint": "node",
								"runtime_arguments": [
									{"value": "notion-mcp"}
								],
								"environment_variables": []
							}
						]
					}
				}
			]
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

	config, err := createMCPToolConfig(server, "", "https://api.mcp.github.com/v0", false)
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

	// Check that environment variables are converted to GitHub Actions syntax
	env, ok := mcpSection["env"].(map[string]string)
	if !ok {
		t.Fatal("Expected env section to be a map[string]string")
	}

	if env["TEST_TOKEN"] != "${{ secrets.TEST_TOKEN }}" {
		t.Errorf("Expected env TEST_TOKEN to be '${{ secrets.TEST_TOKEN }}', got '%s'", env["TEST_TOKEN"])
	}

	// Check that registry field contains the full registry URL path
	if mcpSection["registry"] != "https://api.mcp.github.com/v0/servers/test-server" {
		t.Errorf("Expected registry to be 'https://api.mcp.github.com/v0/servers/test-server', got '%v'", mcpSection["registry"])
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
	config, err := createMCPToolConfig(server, "docker", "https://api.mcp.github.com/v0", false)
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

	// Check that registry field contains the full registry URL path
	if mcpSection["registry"] != "https://api.mcp.github.com/v0/servers/test-server" {
		t.Errorf("Expected registry to be 'https://api.mcp.github.com/v0/servers/test-server', got '%v'", mcpSection["registry"])
	}
}

func TestListAvailableServers(t *testing.T) {
	// Create a test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/servers" {
			response := MCPRegistryResponse{
				Servers: []MCPRegistryServerWrapper{
					{
						Server: MCPRegistryServerData{
							ID:          "notion-mcp",
							Name:        "Notion MCP Server",
							Description: "Connect to Notion API",
						},
					},
					{
						Server: MCPRegistryServerData{
							ID:          "github-mcp",
							Name:        "GitHub MCP Server",
							Description: "Connect to GitHub API",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer testServer.Close()

	// Test listing servers
	err := listAvailableServers(testServer.URL, false)
	if err != nil {
		t.Errorf("listAvailableServers failed: %v", err)
	}
}

func TestCleanMCPToolID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove -mcp suffix",
			input:    "notion-mcp",
			expected: "notion",
		},
		{
			name:     "remove mcp- prefix",
			input:    "mcp-notion",
			expected: "notion",
		},
		{
			name:     "remove both prefix and suffix",
			input:    "mcp-notion-mcp",
			expected: "notion",
		},
		{
			name:     "no changes needed",
			input:    "notion",
			expected: "notion",
		},
		{
			name:     "complex name with mcp suffix",
			input:    "some-server-mcp",
			expected: "some-server",
		},
		{
			name:     "complex name with mcp prefix",
			input:    "mcp-some-server",
			expected: "some-server",
		},
		{
			name:     "mcp only should remain unchanged",
			input:    "mcp",
			expected: "mcp",
		},
		{
			name:     "edge case: mcp-mcp",
			input:    "mcp-mcp",
			expected: "mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanMCPToolID(tt.input)
			if result != tt.expected {
				t.Errorf("cleanMCPToolID(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
