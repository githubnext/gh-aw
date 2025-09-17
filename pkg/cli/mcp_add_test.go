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

	// Check that the notion tool was added (cleaned from notion-mcp)
	if !strings.Contains(updatedContentStr, "notion:") {
		t.Error("Expected notion tool to be added to workflow")
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

	// Check that environment variables use GitHub Actions syntax
	if !strings.Contains(updatedContentStr, "${{ secrets.NOTION_TOKEN }}") {
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

	// Create a test workflow file with existing notion tool (cleaned from notion-mcp)
	workflowContent := `---
name: Test Workflow
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
  notion:
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

	if !strings.Contains(err.Error(), "tool 'notion' already exists") {
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

	// Check that environment variables are converted to GitHub Actions syntax
	env, ok := mcpSection["env"].(map[string]string)
	if !ok {
		t.Fatal("Expected env section to be a map[string]string")
	}

	if env["TEST_TOKEN"] != "${{ secrets.TEST_TOKEN }}" {
		t.Errorf("Expected env TEST_TOKEN to be '${{ secrets.TEST_TOKEN }}', got '%s'", env["TEST_TOKEN"])
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

func TestConvertToGitHubActionsEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]string
	}{
		{
			name: "shell syntax conversion",
			input: map[string]interface{}{
				"API_TOKEN":    "${API_TOKEN}",
				"NOTION_TOKEN": "${NOTION_TOKEN}",
			},
			expected: map[string]string{
				"API_TOKEN":    "${{ secrets.API_TOKEN }}",
				"NOTION_TOKEN": "${{ secrets.NOTION_TOKEN }}",
			},
		},
		{
			name: "mixed syntax",
			input: map[string]interface{}{
				"API_TOKEN":  "${API_TOKEN}",
				"PLAIN_VAR":  "plain_value",
				"GITHUB_VAR": "${{ secrets.EXISTING }}",
			},
			expected: map[string]string{
				"API_TOKEN":  "${{ secrets.API_TOKEN }}",
				"PLAIN_VAR":  "plain_value",
				"GITHUB_VAR": "${{ secrets.EXISTING }}",
			},
		},
		{
			name: "no shell syntax",
			input: map[string]interface{}{
				"PLAIN_VAR": "plain_value",
				"NUMBER":    "123",
			},
			expected: map[string]string{
				"PLAIN_VAR": "plain_value",
				"NUMBER":    "123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToGitHubActionsEnv(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d environment variables, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key '%s', expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		})
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
