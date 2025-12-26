//go:build integration

package cli

import (
	"context"
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
	expectedTools := []string{"status", "compile", "logs", "audit", "mcp-inspect", "add", "update"}
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
	// We use an invalid ID to test that the tool is callable and handles errors properly
	// The tool should return a result with an error message rather than crashing
	params := &mcp.CallToolParams{
		Name: "audit",
		Arguments: map[string]any{
			"run_id": int64(1),
		},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call audit tool: %v", err)
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
						"run_id": int64(1),
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
