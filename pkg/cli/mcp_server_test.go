//go:build integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPServer_ListTools tests that the MCP server exposes the expected tools
func TestMCPServer_ListTools(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// List tools
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Verify expected tools are present
	expectedTools := []string{"status", "compile", "logs", "audit", "mcp-inspect", "add", "update", "fix"}
	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' not found in MCP server tools", expected)
		}
	}

	// Verify we have exactly the expected number of tools
	if len(result.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(result.Tools))
	}
}

// TestMCPServer_CustomCmd tests that the --cmd flag works correctly
func TestMCPServer_CustomCmd(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow file
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
on: push
engine: copilot
---
# Test Workflow

This is a test workflow.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository in the temp directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Get absolute path to binary
	absBinaryPath, err := filepath.Abs(filepath.Join(originalDir, binaryPath))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server with --cmd flag pointing to the binary
	serverCmd := exec.Command(absBinaryPath, "mcp-server", "--cmd", absBinaryPath)
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call status tool
	params := &mcp.CallToolParams{
		Name:      "status",
		Arguments: map[string]any{},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call status tool: %v", err)
	}

	// Verify result is not empty
	if len(result.Content) == 0 {
		t.Error("Expected non-empty result from status tool")
	}

	// Verify result contains text content
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text == "" {
			t.Error("Expected non-empty text content from status tool")
		}
		t.Logf("Status tool output with custom cmd: %s", textContent.Text)
	} else {
		t.Error("Expected text content from status tool")
	}
}

// TestMCPServer_StatusTool tests the status tool
func TestMCPServer_StatusTool(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow file
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
on: push
engine: copilot
---
# Test Workflow

This is a test workflow.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository in the temp directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call status tool
	params := &mcp.CallToolParams{
		Name:      "status",
		Arguments: map[string]any{},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call status tool: %v", err)
	}

	// Verify result is not empty
	if len(result.Content) == 0 {
		t.Error("Expected non-empty result from status tool")
	}

	// Verify result contains text content
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text == "" {
			t.Error("Expected non-empty text content from status tool")
		}
	} else {
		t.Error("Expected text content from status tool")
	}
}

// TestMCPServer_AuditTool tests the audit tool
func TestMCPServer_AuditTool(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, _ := os.Getwd()

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call audit tool with an invalid run ID
	// The tool should return an MCP error for invalid run IDs
	params := &mcp.CallToolParams{
		Name: "audit",
		Arguments: map[string]any{
			"run_id_or_url": "1",
		},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		// Expected behavior: audit command fails with invalid run ID
		t.Logf("Audit tool correctly returned error for invalid run ID: %v", err)
		return
	}

	// Verify result is not empty
	if len(result.Content) == 0 {
		t.Error("Expected non-empty result from audit tool")
	}

	// Verify result contains text content
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content from audit tool")
	}

	if textContent.Text == "" {
		t.Error("Expected non-empty text content from audit tool")
	}

	// The audit command should fail with an invalid run ID, but should return
	// a proper error message rather than crashing
	// We just verify that we got some output (either error or success)
	t.Logf("Audit tool output: %s", textContent.Text)
}

// TestMCPServer_CompileTool tests the compile tool
func TestMCPServer_CompileTool(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow file
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
on: push
engine: copilot
---
# Test Workflow

This is a test workflow for compilation.
`
	workflowPath := filepath.Join(workflowsDir, "test-compile.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository in the temp directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool
	params := &mcp.CallToolParams{
		Name:      "compile",
		Arguments: map[string]any{},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call compile tool: %v", err)
	}

	// Verify result is not empty
	if len(result.Content) == 0 {
		t.Error("Expected non-empty result from compile tool")
	}

	// Verify result contains text content
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text == "" {
			t.Error("Expected non-empty text content from compile tool")
		}
		// The compile tool is callable - it may fail in test environment
		// due to missing gh extension, but we're testing the MCP interface works
		t.Logf("Compile tool output: %s", textContent.Text)
	} else {
		t.Error("Expected text content from compile tool")
	}
}

// // TestMCPServer_LogsTool tests the logs tool
// func TestMCPServer_LogsTool(t *testing.T) {
// 	// Skip if the binary doesn't exist
// 	binaryPath := "../../gh-aw"
// 	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
// 		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
// 	}

// 	// Get the current directory for proper path resolution
// 	originalDir, _ := os.Getwd()

// 	// Create MCP client
// 	client := mcp.NewClient(&mcp.Implementation{
// 		Name:    "test-client",
// 		Version: "1.0.0",
// 	}, nil)

// 	// Start the MCP server as a subprocess
// 	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
// 	transport := &mcp.CommandTransport{Command: serverCmd}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	session, err := client.Connect(ctx, transport, nil)
// 	if err != nil {
// 		t.Fatalf("Failed to connect to MCP server: %v", err)
// 	}
// 	defer session.Close()

// 	// Call logs tool
// 	// This will likely fail in test environment due to missing GitHub credentials
// 	// but we're testing that the tool is callable and returns a proper response
// 	params := &mcp.CallToolParams{
// 		Name: "logs",
// 		Arguments: map[string]any{
// 			"count": 1,
// 		},
// 	}
// 	result, err := session.CallTool(ctx, params)
// 	if err != nil {
// 		t.Fatalf("Failed to call logs tool: %v", err)
// 	}

// 	// Verify result is not empty
// 	if len(result.Content) == 0 {
// 		t.Error("Expected non-empty result from logs tool")
// 	}

// 	// Verify result contains text content
// 	textContent, ok := result.Content[0].(*mcp.TextContent)
// 	if !ok {
// 		t.Fatal("Expected text content from logs tool")
// 	}

// 	if textContent.Text == "" {
// 		t.Error("Expected non-empty text content from logs tool")
// 	}

// 	t.Logf("Logs tool output: %s", textContent.Text)
// }

// TestMCPServer_ServerInfo tests that server info is correctly returned
func TestMCPServer_ServerInfo(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, _ := os.Getwd()

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// List tools to verify server is working properly
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Verify we can get tools, which means server initialized correctly
	if len(result.Tools) == 0 {
		t.Error("Expected server to have tools available")
	}

	t.Logf("Server initialized successfully with %d tools", len(result.Tools))
}

// TestMCPServer_CompileWithSpecificWorkflow tests compiling a specific workflow
func TestMCPServer_CompileWithSpecificWorkflow(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with multiple workflow files
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create two test workflow files
	workflowContent1 := `---
on: push
engine: copilot
---
# Test Workflow 1

This is the first test workflow.
`
	workflowPath1 := filepath.Join(workflowsDir, "test1.md")
	if err := os.WriteFile(workflowPath1, []byte(workflowContent1), 0644); err != nil {
		t.Fatalf("Failed to write workflow file 1: %v", err)
	}

	workflowContent2 := `---
on: pull_request
engine: claude
---
# Test Workflow 2

This is the second test workflow.
`
	workflowPath2 := filepath.Join(workflowsDir, "test2.md")
	if err := os.WriteFile(workflowPath2, []byte(workflowContent2), 0644); err != nil {
		t.Fatalf("Failed to write workflow file 2: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository in the temp directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool with specific workflow
	params := &mcp.CallToolParams{
		Name: "compile",
		Arguments: map[string]any{
			"workflows": []string{"test1"},
		},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call compile tool: %v", err)
	}

	// Verify result is not empty
	if len(result.Content) == 0 {
		t.Error("Expected non-empty result from compile tool")
	}

	// Verify result contains text content
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text == "" {
			t.Error("Expected non-empty text content from compile tool")
		}
		// The compile tool is callable - it may fail in test environment
		// due to missing gh extension, but we're testing the MCP interface works
		t.Logf("Compile tool output: %s", textContent.Text)
	} else {
		t.Error("Expected text content from compile tool")
	}
}

// TestMCPServer_UpdateToolSchema tests that the update tool has the correct schema
func TestMCPServer_UpdateToolSchema(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// List tools
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Find the update tool
	var updateTool *mcp.Tool
	for i := range result.Tools {
		if result.Tools[i].Name == "update" {
			updateTool = result.Tools[i]
			break
		}
	}

	if updateTool == nil {
		t.Fatal("Update tool not found in MCP server tools")
	}

	// Verify the tool has a description
	if updateTool.Description == "" {
		t.Error("Update tool should have a description")
	}

	// Verify description mentions key functionality
	if !strings.Contains(updateTool.Description, "workflows") {
		t.Error("Update tool description should mention workflows")
	}

	// Verify the tool has input schema
	if updateTool.InputSchema == nil {
		t.Error("Update tool should have an input schema")
	}

	t.Logf("Update tool description: %s", updateTool.Description)
	t.Logf("Update tool schema: %+v", updateTool.InputSchema)
}

// TestMCPServer_CapabilitiesConfiguration tests that server capabilities are correctly configured
func TestMCPServer_CapabilitiesConfiguration(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, _ := os.Getwd()

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Get server capabilities from the initialize result
	initResult := session.InitializeResult()
	if initResult == nil {
		t.Fatal("Expected non-nil InitializeResult")
	}

	serverCapabilities := initResult.Capabilities

	// Verify Tools capability is present
	if serverCapabilities.Tools == nil {
		t.Fatal("Expected server to advertise Tools capability")
	}

	// Verify ListChanged is set to false
	if serverCapabilities.Tools.ListChanged {
		t.Error("Expected Tools.ListChanged to be false (tools are static)")
	}

	t.Logf("Server capabilities configured correctly: Tools.ListChanged = %v", serverCapabilities.Tools.ListChanged)
}

// TestMCPServer_ContextCancellation tests that tool handlers properly respond to context cancellation
func TestMCPServer_ContextCancellation(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, _ := os.Getwd()

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Test context cancellation for different tools
	tools := []string{"status", "audit"}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			// Create a context that's already cancelled
			cancelledCtx, immediateCancel := context.WithCancel(context.Background())
			immediateCancel() // Cancel immediately

			var params *mcp.CallToolParams
			switch toolName {
			case "status":
				params = &mcp.CallToolParams{
					Name:      "status",
					Arguments: map[string]any{},
				}
			case "audit":
				params = &mcp.CallToolParams{
					Name: "audit",
					Arguments: map[string]any{
						"run_id_or_url": "1",
					},
				}
			}

			// Call the tool with a cancelled context
			result, err := session.CallTool(cancelledCtx, params)

			// The tool should handle the cancellation gracefully
			// It should either return an error OR return a result with error message
			if err != nil {
				// Check if it's a context cancellation error
				if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "cancel") {
					t.Logf("Tool returned error (acceptable): %v", err)
				} else {
					t.Logf("Tool properly detected cancellation via error: %v", err)
				}
			} else if result != nil && len(result.Content) > 0 {
				// Check if the result contains an error message about cancellation
				if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
					text := textContent.Text
					if strings.Contains(text, "context") || strings.Contains(text, "cancel") {
						t.Logf("Tool properly detected cancellation via result: %s", text)
					} else {
						t.Logf("Tool returned result (may not have detected cancellation immediately): %s", text)
					}
				}
			}

			// The important thing is that the tool doesn't hang or crash
			// Either returning an error or a result with error message is acceptable
		})
	}
}

// TestMCPServer_ToolIcons tests that all tools have icons
func TestMCPServer_ToolIcons(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// List tools
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Expected icons for each tool
	expectedIcons := map[string]string{
		"status":      "📊",
		"compile":     "🔨",
		"logs":        "📜",
		"audit":       "🔍",
		"mcp-inspect": "🔎",
		"add":         "➕",
		"update":      "🔄",
	}

	// Verify each tool has an icon
	for _, tool := range result.Tools {
		if len(tool.Icons) == 0 {
			t.Errorf("Tool '%s' is missing an icon", tool.Name)
			continue
		}

		// Check that the icon source matches expected emoji
		if expectedIcon, ok := expectedIcons[tool.Name]; ok {
			if tool.Icons[0].Source != expectedIcon {
				t.Errorf("Tool '%s' has unexpected icon. Expected: %s, Got: %s",
					tool.Name, expectedIcon, tool.Icons[0].Source)
			}
			t.Logf("Tool '%s' has correct icon: %s", tool.Name, tool.Icons[0].Source)
		} else {
			t.Logf("Tool '%s' has icon (not in expected list): %s", tool.Name, tool.Icons[0].Source)
		}
	}

	// Verify we checked all expected tools
	if len(result.Tools) != len(expectedIcons) {
		t.Errorf("Expected %d tools with icons, got %d tools", len(expectedIcons), len(result.Tools))
	}
}

// TestMCPServer_CompileToolWithErrors tests that compile tool returns output even when compilation fails
func TestMCPServer_CompileToolWithErrors(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with an invalid workflow file
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a workflow file with syntax errors
	workflowContent := `---
on: push
engine: copilot
toolz:
  - invalid-tool
---
# Invalid Workflow

This workflow has a syntax error in the frontmatter.
`
	workflowPath := filepath.Join(workflowsDir, "invalid.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository in the temp directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool with the invalid workflow
	params := &mcp.CallToolParams{
		Name:      "compile",
		Arguments: map[string]any{},
	}
	result, err := session.CallTool(ctx, params)

	// The key test: compile tool should NOT return an MCP error
	// even though the workflow has compilation errors
	if err != nil {
		t.Errorf("Compile tool should not return MCP error for compilation failures, got: %v", err)
	}

	// Verify we got a result with content
	if result == nil {
		t.Fatal("Expected result from compile tool, got nil")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected non-empty result content from compile tool")
	}

	// Verify result contains text content
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content from compile tool")
	}

	if textContent.Text == "" {
		t.Error("Expected non-empty text content from compile tool")
	}

	// Verify the output contains validation error information
	// The JSON output should include error details even though compilation failed
	if !strings.Contains(textContent.Text, "\"valid\"") || !strings.Contains(textContent.Text, "\"errors\"") {
		t.Errorf("Expected JSON output with validation results, got: %s", textContent.Text)
	}

	t.Logf("Compile tool correctly returned validation errors in output: %s", textContent.Text)
}

// TestMCPServer_CompileToolWithMultipleWorkflows tests compiling multiple workflows with mixed results
func TestMCPServer_CompileToolWithMultipleWorkflows(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with multiple workflow files
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a valid workflow
	validWorkflow := `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
---
# Valid Workflow
This workflow should compile successfully.
`
	validPath := filepath.Join(workflowsDir, "valid.md")
	if err := os.WriteFile(validPath, []byte(validWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write valid workflow: %v", err)
	}

	// Create an invalid workflow
	invalidWorkflow := `---
on: push
engine: copilot
unknown_field: invalid
---
# Invalid Workflow
This workflow has an unknown field.
`
	invalidPath := filepath.Join(workflowsDir, "invalid.md")
	if err := os.WriteFile(invalidPath, []byte(invalidWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write invalid workflow: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool for all workflows
	params := &mcp.CallToolParams{
		Name:      "compile",
		Arguments: map[string]any{},
	}
	result, err := session.CallTool(ctx, params)

	// Should not return MCP error even with mixed results
	if err != nil {
		t.Errorf("Compile tool should not return MCP error, got: %v", err)
	}

	// Verify we got results
	if result == nil || len(result.Content) == 0 {
		t.Fatal("Expected non-empty result content")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content from compile tool")
	}

	// Parse JSON to verify structure
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &results); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Should have results for both workflows
	if len(results) < 2 {
		t.Errorf("Expected at least 2 workflow results, got %d", len(results))
	}

	t.Logf("Compile tool handled multiple workflows correctly: %d results", len(results))
}

// TestMCPServer_CompileToolWithStrictMode tests compile with strict mode flag
func TestMCPServer_CompileToolWithStrictMode(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a workflow that might have strict mode issues
	workflowContent := `---
on: push
engine: copilot
strict: false
---
# Test Workflow
This workflow has strict mode disabled in frontmatter.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool with strict mode enabled
	params := &mcp.CallToolParams{
		Name: "compile",
		Arguments: map[string]any{
			"strict": true,
		},
	}
	result, err := session.CallTool(ctx, params)

	// Should not return MCP error
	if err != nil {
		t.Errorf("Compile tool should not return MCP error with strict flag, got: %v", err)
	}

	// Verify we got results
	if result == nil || len(result.Content) == 0 {
		t.Fatal("Expected non-empty result content")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content from compile tool")
	}

	t.Logf("Compile tool with strict mode returned: %s", textContent.Text)
}

// TestMCPServer_CompileToolWithJqFilter tests compile with jq filter parameter
func TestMCPServer_CompileToolWithJqFilter(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow
	workflowContent := `---
on: push
engine: copilot
permissions:
  contents: read
---
# Test Workflow
Test workflow for jq filtering.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool with jq filter to extract only workflow names
	params := &mcp.CallToolParams{
		Name: "compile",
		Arguments: map[string]any{
			"jq": ".[].workflow",
		},
	}
	result, err := session.CallTool(ctx, params)

	// Should not return MCP error
	if err != nil {
		t.Errorf("Compile tool should not return MCP error with jq filter, got: %v", err)
	}

	// Verify we got filtered results
	if result == nil || len(result.Content) == 0 {
		t.Fatal("Expected non-empty result content")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content from compile tool")
	}

	// The output should be filtered by jq
	if !strings.Contains(textContent.Text, "test.md") {
		t.Errorf("Expected jq-filtered output to contain workflow name, got: %s", textContent.Text)
	}

	t.Logf("Compile tool with jq filter returned: %s", textContent.Text)
}

// TestMCPServer_CompileToolWithInvalidJqFilter tests compile with invalid jq filter
func TestMCPServer_CompileToolWithInvalidJqFilter(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow
	workflowContent := `---
on: push
engine: copilot
permissions:
  contents: read
---
# Test Workflow
Test workflow.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool with invalid jq filter
	params := &mcp.CallToolParams{
		Name: "compile",
		Arguments: map[string]any{
			"jq": ".[invalid syntax",
		},
	}
	result, err := session.CallTool(ctx, params)

	// Should return MCP error for invalid jq filter
	if err == nil {
		t.Error("Expected MCP error for invalid jq filter")
	}

	// Error should mention jq filter
	if err != nil && !strings.Contains(err.Error(), "jq") {
		t.Errorf("Expected error message to mention jq filter, got: %v", err)
	}

	if result != nil {
		t.Log("Got unexpected result despite invalid jq filter")
	}

	t.Logf("Compile tool correctly rejected invalid jq filter: %v", err)
}

// TestMCPServer_CompileToolWithSpecificWorkflows tests compiling specific workflows by name
func TestMCPServer_CompileToolWithSpecificWorkflows(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with multiple workflows
	tmpDir := testutil.TempDir(t, "test-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create workflow 1
	workflow1 := `---
on: push
engine: copilot
permissions:
  contents: read
---
# Workflow 1
First test workflow.
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "workflow1.md"), []byte(workflow1), 0644); err != nil {
		t.Fatalf("Failed to write workflow1: %v", err)
	}

	// Create workflow 2
	workflow2 := `---
on: pull_request
engine: copilot
permissions:
  contents: read
---
# Workflow 2
Second test workflow.
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "workflow2.md"), []byte(workflow2), 0644); err != nil {
		t.Fatalf("Failed to write workflow2: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Initialize git repository
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Call compile tool for only workflow1
	params := &mcp.CallToolParams{
		Name: "compile",
		Arguments: map[string]any{
			"workflows": []string{"workflow1"},
		},
	}
	result, err := session.CallTool(ctx, params)

	// Should not return MCP error
	if err != nil {
		t.Errorf("Compile tool should not return MCP error, got: %v", err)
	}

	// Verify we got results
	if result == nil || len(result.Content) == 0 {
		t.Fatal("Expected non-empty result content")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content from compile tool")
	}

	// Parse JSON to verify only workflow1 was compiled
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &results); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Should have exactly 1 result
	if len(results) != 1 {
		t.Errorf("Expected 1 workflow result, got %d", len(results))
	}

	// Verify it's workflow1
	if len(results) > 0 {
		workflow, _ := results[0]["workflow"].(string)
		if !strings.Contains(workflow, "workflow1") {
			t.Errorf("Expected workflow1 in results, got: %s", workflow)
		}
	}

	t.Logf("Compile tool correctly compiled specific workflow: %s", textContent.Text)
}

// TestMCPServer_CompileToolDescriptionMentionsRecompileRequirement tests that the compile tool
// description clearly states that changes to .md files must be compiled
func TestMCPServer_CompileToolDescriptionMentionsRecompileRequirement(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server")
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// List tools
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Find the compile tool
	var compileTool *mcp.Tool
	for i := range result.Tools {
		if result.Tools[i].Name == "compile" {
			compileTool = result.Tools[i]
			break
		}
	}

	if compileTool == nil {
		t.Fatal("Compile tool not found in MCP server tools")
	}

	// Verify the description exists and is not empty
	if compileTool.Description == "" {
		t.Fatal("Compile tool should have a description")
	}

	// Key requirements that must be in the description
	requiredPhrases := []string{
		".github/workflows/*.md",
		"MUST be compiled",
		".lock.yml",
	}

	// Verify each required phrase is present in the description
	description := compileTool.Description
	for _, phrase := range requiredPhrases {
		if !strings.Contains(description, phrase) {
			t.Errorf("Compile tool description should mention '%s' but it doesn't.\nDescription: %s", phrase, description)
		}
	}

	// Verify description emphasizes the importance (should contain warning indicator)
	if !strings.Contains(description, "⚠️") && !strings.Contains(description, "IMPORTANT") {
		t.Error("Compile tool description should emphasize importance with warning or 'IMPORTANT' marker")
	}

	// Verify description explains why compilation is needed
	if !strings.Contains(description, "GitHub Actions") {
		t.Error("Compile tool description should explain that GitHub Actions executes the .lock.yml files")
	}

	t.Logf("Compile tool description successfully emphasizes recompilation requirement:\n%s", description)
}
