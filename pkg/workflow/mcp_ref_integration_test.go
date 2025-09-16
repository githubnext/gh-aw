package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPRefIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mcp-ref-integration-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .vscode directory and mcp.json file
	vscodeDirPath := filepath.Join(tmpDir, ".vscode")
	err = os.MkdirAll(vscodeDirPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	mcpJSON := `{
  "servers": {
    "my-tool": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
      "env": {
        "NODE_ENV": "production",
        "API_KEY": "secret"
      }
    }
  }
}`

	mcpJSONPath := filepath.Join(vscodeDirPath, "mcp.json")
	err = os.WriteFile(mcpJSONPath, []byte(mcpJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a workflow markdown file
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	workflowMD := `---
engine: claude
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  my-tool:
    mcp-ref: "vscode"
    allowed: [list_files, read_file]
---

# Weekly file analysis

Analyze the files in the repository and create a summary.
`

	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	err = os.WriteFile(workflowPath, []byte(workflowMD), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Change to the temporary directory so the compiler can find .vscode/mcp.json
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test the complete workflow compilation
	compiler := NewCompiler(false, "", "test")

	// Parse the workflow file
	workflowData, err := compiler.parseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Validate that the tools section was processed correctly
	if workflowData.Tools == nil {
		t.Fatal("Expected tools section to be populated")
	}

	myTool, exists := workflowData.Tools["my-tool"]
	if !exists {
		t.Fatal("Expected 'my-tool' to exist in tools")
	}

	toolConfig, ok := myTool.(map[string]any)
	if !ok {
		t.Fatal("Expected 'my-tool' to be a map")
	}

	// Check that mcp-ref validation passed
	if err := validateMCPRef("my-tool", toolConfig); err != nil {
		t.Fatalf("MCP ref validation failed: %v", err)
	}

	// Check that the tool is detected as having MCP config
	hasMCP, mcpType := hasMCPConfig(toolConfig)
	if !hasMCP {
		t.Fatal("Expected tool to have MCP configuration")
	}
	if mcpType != "stdio" {
		t.Fatalf("Expected MCP type 'stdio', got '%s'", mcpType)
	}

	// Test that getMCPConfig loads the VSCode configuration
	mcpConfig, err := getMCPConfig(toolConfig, "my-tool")
	if err != nil {
		t.Fatalf("Failed to get MCP config: %v", err)
	}

	// Verify the loaded configuration
	if mcpConfig["type"] != "stdio" {
		t.Errorf("Expected type 'stdio', got '%s'", mcpConfig["type"])
	}

	if mcpConfig["command"] != "npx" {
		t.Errorf("Expected command 'npx', got '%s'", mcpConfig["command"])
	}

	args, ok := mcpConfig["args"].([]any)
	if !ok {
		t.Fatal("Expected args to be []any")
	}
	if len(args) != 3 {
		t.Fatalf("Expected 3 args, got %d", len(args))
	}
	if args[0] != "-y" {
		t.Errorf("Expected first arg '-y', got '%s'", args[0])
	}

	env, ok := mcpConfig["env"].(map[string]any)
	if !ok {
		t.Fatal("Expected env to be map[string]any")
	}
	if env["NODE_ENV"] != "production" {
		t.Errorf("Expected NODE_ENV 'production', got '%s'", env["NODE_ENV"])
	}
	if env["API_KEY"] != "secret" {
		t.Errorf("Expected API_KEY 'secret', got '%s'", env["API_KEY"])
	}
}

func TestMCPRefWithInvalidInputs(t *testing.T) {
	// Test that inputs are rejected when using mcp-ref
	tools := map[string]any{
		"my-tool": map[string]any{
			"mcp-ref": "vscode",
			"inputs":  map[string]any{"key": "value"},
		},
	}

	err := ValidateMCPConfigs(tools)
	if err == nil {
		t.Fatal("Expected validation to fail when using mcp-ref with inputs")
	}

	if !strings.Contains(err.Error(), "cannot specify 'inputs'") {
		t.Errorf("Expected error about inputs, got: %v", err)
	}
}

func TestMCPRefWithRegularMCP(t *testing.T) {
	// Test that mcp-ref and mcp sections cannot coexist
	tools := map[string]any{
		"my-tool": map[string]any{
			"mcp-ref": "vscode",
			"mcp": map[string]any{
				"type": "stdio",
			},
		},
	}

	err := ValidateMCPConfigs(tools)
	if err == nil {
		t.Fatal("Expected validation to fail when using both mcp-ref and mcp")
	}

	if !strings.Contains(err.Error(), "cannot specify both 'mcp-ref' and 'mcp'") {
		t.Errorf("Expected error about conflicting configs, got: %v", err)
	}
}