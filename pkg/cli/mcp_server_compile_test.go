//go:build integration

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPServer_ReadOnlyCompilePreservesFiles(t *testing.T) {
	binaryPath := testutil.RequireGhAwBinary(t)
	for _, existing := range []bool{false, true} {
		for _, all := range []bool{false, true} {
			for _, kind := range []string{"valid", "invalid", "oversized", "uncached-import", "cached-import", "cached-import-subdirectory"} {
				name := fmt.Sprintf("existing=%t/all=%t/%s", existing, all, kind)
				t.Run(name, func(t *testing.T) {
					dir := t.TempDir()
					workflowsDir := filepath.Join(dir, ".github", "workflows")
					require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
					require.NoError(t, initTestGitRepo(dir))
					frontmatter := "on: push\nengine: copilot\n"
					body := "# Test\n\nCompile without writing files.\n"
					switch kind {
					case "invalid":
						frontmatter += "unknown-field: true\n"
					case "oversized":
						frontmatter += "inlined-imports: true\n"
						body += strings.Repeat("x", 25000)
					case "uncached-import", "cached-import", "cached-import-subdirectory":
						frontmatter += "imports:\n  - owner/repo/shared.md@0123456789012345678901234567890123456789\n"
					}
					if kind == "cached-import" || kind == "cached-import-subdirectory" {
						cacheDir := filepath.Join(dir, ".github", "aw", "imports", "owner", "repo", "0123456789012345678901234567890123456789")
						require.NoError(t, os.MkdirAll(cacheDir, 0o755))
						require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "shared.md"), []byte("# Shared\n\nCached instructions.\n"), 0o600))
					}
					require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "test.md"), []byte("---\n"+frontmatter+"---\n"+body), 0o644))
					if existing {
						for path, content := range map[string]string{
							".gitattributes":                             "# preserve attributes\n",
							".github/aw/actions-lock.json":               "{\"entries\":{}}\n",
							".github/workflows/test.lock.yml":            "# preserve lock file\n",
							".github/workflows/test.invalid.yml":         "# preserve diagnostics\n",
							".github/workflows/agentics-maintenance.yml": "# preserve maintenance\n",
						} {
							fullPath := filepath.Join(dir, filepath.FromSlash(path))
							require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
							require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
						}
					}
					serverDir := dir
					if kind == "cached-import-subdirectory" {
						serverDir = filepath.Join(dir, "src", "nested")
						require.NoError(t, os.MkdirAll(serverDir, 0o755))
					}
					before := snapshotMCPCompileFiles(t, dir)
					ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
					defer cancel()
					serverCmd := exec.CommandContext(ctx, binaryPath, "mcp-server", "--cmd", binaryPath, "--validate-actor")
					serverCmd.Dir = serverDir
					client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
					session, err := client.Connect(ctx, &mcp.CommandTransport{Command: serverCmd}, nil)
					require.NoError(t, err)
					defer session.Close()
					args := map[string]any{}
					if !all {
						args["workflows"] = []string{"test"}
					}
					if kind == "cached-import-subdirectory" {
						args["workflows"] = []string{filepath.Join(workflowsDir, "test.md")}
						if all {
							args["workflows"] = []string{workflowsDir}
						}
					}
					result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "compile", Arguments: args})
					require.NoError(t, err)
					require.False(t, result.IsError)
					require.NotEmpty(t, result.Content)
					text, ok := result.Content[0].(*mcp.TextContent)
					require.True(t, ok)
					var results []ValidationResult
					require.NoError(t, json.Unmarshal([]byte(text.Text), &results), text.Text)
					require.NotEmpty(t, results, text.Text)
					require.Equal(t, kind == "valid" || kind == "cached-import" || kind == "cached-import-subdirectory", results[0].Valid, text.Text)
					switch kind {
					case "invalid":
						require.Contains(t, text.Text, "unknown-field")
					case "oversized":
						require.Contains(t, text.Text, "expression size validation failed")
					case "uncached-import":
						require.Contains(t, text.Text, "not cached")
					}
					require.NoError(t, session.Close())
					require.Equal(t, before, snapshotMCPCompileFiles(t, dir), "read-only compilation must not create, delete, or modify files")
				})
			}
		}
	}
}

func snapshotMCPCompileFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			files[path] = info.Mode().String()
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = fmt.Sprintf("%s %s %x", info.Mode(), info.ModTime().UTC(), sha256.Sum256(content))
		return nil
	}))
	return files
}

func TestMCPServer_CompileTool(t *testing.T) {
	binaryPath := testutil.RequireGhAwBinary(t)

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

	// Initialize git repository using shared helper
	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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
	require.FileExists(t, filepath.Join(workflowsDir, "test-compile.lock.yml"), "write-enabled compilation must still emit its lock file")
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
// 	serverCmd := exec.Command(binaryPath, "mcp-server", "--cmd", binaryPath)
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

func TestMCPServer_CompileWithSpecificWorkflow(t *testing.T) {
	binaryPath := testutil.RequireGhAwBinary(t)

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

	// Initialize git repository using shared helper
	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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

func TestMCPServer_CompileToolWithErrors(t *testing.T) {
	binaryPath := testutil.RequireGhAwBinary(t)

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

	// Initialize git repository using shared helper
	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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
	binaryPath := testutil.RequireGhAwBinary(t)

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

	// Initialize git repository using shared helper
	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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
	binaryPath := testutil.RequireGhAwBinary(t)

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

	// Initialize git repository using shared helper
	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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

// TestMCPServer_CompileToolWithSpecificWorkflows tests compiling specific workflows by name

func TestMCPServer_CompileToolWithSpecificWorkflows(t *testing.T) {
	binaryPath := testutil.RequireGhAwBinary(t)

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

	// Initialize git repository using shared helper
	if err := initTestGitRepo(tmpDir); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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
	binaryPath := testutil.RequireGhAwBinary(t)

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(binaryPath, "mcp-server", "--allow-write-tools", "--cmd", binaryPath)
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
