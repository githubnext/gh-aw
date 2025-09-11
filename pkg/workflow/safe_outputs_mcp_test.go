package workflow

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSafeOutputsMCPServer(t *testing.T) {
	// Test that the embedded MCP server script exists and is valid
	script := SafeOutputsMCPServerScript()
	if script == "" {
		t.Fatal("MCP server script is empty")
	}

	// Check that it looks like JavaScript
	if !strings.Contains(script, "#!/usr/bin/env node") {
		t.Error("MCP server script doesn't appear to be a Node.js script")
	}

	if !strings.Contains(script, "SafeOutputsMCPServer") {
		t.Error("MCP server script doesn't contain SafeOutputsMCPServer class")
	}
}

func TestWriteSafeOutputsMCPServerToTemp(t *testing.T) {
	serverPath, err := WriteSafeOutputsMCPServerToTemp()
	if err != nil {
		t.Fatalf("Failed to write MCP server to temp: %v", err)
	}

	// Check that the file exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Error("MCP server file was not created")
	}

	// Check that the content is correct
	content, err := os.ReadFile(serverPath)
	if err != nil {
		t.Fatalf("Failed to read MCP server file: %v", err)
	}

	if !strings.Contains(string(content), "SafeOutputsMCPServer") {
		t.Error("MCP server file doesn't contain expected content")
	}

	// Clean up
	os.RemoveAll("/tmp/safe-outputs-mcp")
}

func TestMCPFlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expectMCP   *bool
	}{
		{
			name:        "no safe-outputs",
			frontmatter: map[string]any{"engine": "claude"},
			expectMCP:   nil,
		},
		{
			name: "mcp: true",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"mcp":          true,
					"create-issue": map[string]any{},
				},
			},
			expectMCP: &[]bool{true}[0],
		},
		{
			name: "mcp: false",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"mcp":          false,
					"create-issue": map[string]any{},
				},
			},
			expectMCP: &[]bool{false}[0],
		},
		{
			name: "no mcp field",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{},
				},
			},
			expectMCP: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)

			if config == nil && tt.expectMCP != nil {
				t.Error("Expected safe outputs config but got nil")
				return
			}

			if config != nil {
				if tt.expectMCP == nil && config.MCP != nil {
					t.Errorf("Expected no MCP flag but got %v", *config.MCP)
				} else if tt.expectMCP != nil && config.MCP == nil {
					t.Error("Expected MCP flag but got nil")
				} else if tt.expectMCP != nil && config.MCP != nil && *tt.expectMCP != *config.MCP {
					t.Errorf("Expected MCP flag %v but got %v", *tt.expectMCP, *config.MCP)
				}
			}
		})
	}
}

func TestMCPServerIntegration(t *testing.T) {
	// This test requires Node.js to be available, so skip if not available
	if !isNodeJSAvailable() {
		t.Skip("Node.js not available, skipping MCP server integration test")
	}

	// Create a temporary safe outputs file
	tmpFile, err := os.CreateTemp("", "safe_outputs_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create configuration for all safe output types
	safeOutputsConfig := map[string]interface{}{
		"create-issue": map[string]interface{}{
			"max": 1,
		},
		"add-issue-comment": map[string]interface{}{
			"max": 1,
		},
		"create-pull-request": map[string]interface{}{
			"max": 1,
		},
		"create-pull-request-review-comment": map[string]interface{}{
			"max": 1,
		},
		"create-repository-security-advisory": map[string]interface{}{
			"max": 5,
		},
		"add-issue-label": map[string]interface{}{
			"max": 3,
		},
		"update-issue": map[string]interface{}{
			"max": 1,
		},
		"push-to-branch": map[string]interface{}{
			"max": 1,
		},
		"missing-tool": map[string]interface{}{
			"max": 5,
		},
		"create-discussion": map[string]interface{}{
			"max": 1,
		},
	}

	configJSON, err := json.Marshal(safeOutputsConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Set up environment variables
	os.Setenv("GITHUB_AW_SAFE_OUTPUTS", tmpFile.Name())
	os.Setenv("GITHUB_AW_SAFE_OUTPUTS_CONFIG", string(configJSON))
	defer func() {
		os.Unsetenv("GITHUB_AW_SAFE_OUTPUTS")
		os.Unsetenv("GITHUB_AW_SAFE_OUTPUTS_CONFIG")
	}()

	// Write MCP server to temp directory
	serverPath, err := WriteSafeOutputsMCPServerToTemp()
	if err != nil {
		t.Fatalf("Failed to write MCP server: %v", err)
	}
	defer os.RemoveAll("/tmp/safe-outputs-mcp")

	// Test MCP server via Go SDK
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = testMCPServerWithGoSDK(ctx, serverPath, t)
	if err != nil {
		t.Errorf("MCP server test failed: %v", err)
	}
}

// Helper function to test MCP server with Go SDK
func testMCPServerWithGoSDK(ctx context.Context, serverPath string, t *testing.T) error {
	// Create a command to run the MCP server
	cmd := []string{"node", serverPath}
	
	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	transport := mcp.NewCommandTransport(cmd...)

	// Connect to the server
	session, err := client.Connect(ctx, transport, &mcp.ClientSessionOptions{})
	if err != nil {
		return err
	}
	defer session.Close()

	// List tools
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return err
	}

	// Verify we got the expected number of tools (10 safe output types)
	expectedToolCount := 10
	if len(toolsResult.Tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(toolsResult.Tools))
	}

	// Verify we have the expected tools
	expectedTools := []string{
		"create-issue", "add-issue-comment", "create-pull-request",
		"create-pull-request-review-comment", "create-repository-security-advisory",
		"add-issue-label", "update-issue", "push-to-branch",
		"missing-tool", "create-discussion",
	}

	toolNames := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		toolNames[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool '%s' not found", expectedTool)
		}
	}

	// Test calling a tool
	createIssueArgs := map[string]any{
		"title": "Test Issue",
		"body":  "This is a test issue created by MCP server test",
		"labels": []string{"test", "automated"},
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "create-issue",
		Arguments: createIssueArgs,
	})
	if err != nil {
		return err
	}

	if len(result.Content) == 0 {
		return err
	}

	// Check that the JSONL entry was written to the file
	outputFile := os.Getenv("GITHUB_AW_SAFE_OUTPUTS")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return err
	}

	if !strings.Contains(string(content), "create-issue") {
		t.Error("Expected 'create-issue' entry in safe outputs file")
	}

	if !strings.Contains(string(content), "Test Issue") {
		t.Error("Expected 'Test Issue' in safe outputs file")
	}

	return nil
}

// Helper function to check if Node.js is available
func isNodeJSAvailable() bool {
	// Try to run node --version
	return true // For now, assume Node.js is available in CI environment
}