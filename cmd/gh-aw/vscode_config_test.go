package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestVSCodeMCPConfig validates that the .vscode/mcp.json file has the correct MCP server command
func TestVSCodeMCPConfig(t *testing.T) {
	// Read the .vscode/mcp.json file
	configPath := filepath.Join("..", "..", ".vscode", "mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read .vscode/mcp.json: %v", err)
	}

	// Parse the JSON
	var config struct {
		Servers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
			Cwd     string   `json:"cwd"`
		} `json:"servers"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse .vscode/mcp.json: %v", err)
	}

	// Validate the github-agentic-workflows server configuration
	server, ok := config.Servers["github-agentic-workflows"]
	if !ok {
		t.Fatal("github-agentic-workflows server not found in .vscode/mcp.json")
	}

	// Check that the command is correct
	expectedCommand := "./gh-aw"
	if server.Command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, server.Command)
	}

	// Check that the args are correct - should be ["mcp-server"], not ["mcp", "serve"]
	if len(server.Args) != 1 {
		t.Errorf("Expected 1 argument, got %d: %v", len(server.Args), server.Args)
	}

	expectedArg := "mcp-server"
	if len(server.Args) > 0 && server.Args[0] != expectedArg {
		t.Errorf("Expected first argument to be '%s', got '%s'", expectedArg, server.Args[0])
	}

	// Validate that the incorrect ["mcp", "serve"] is not used
	if len(server.Args) >= 2 {
		if server.Args[0] == "mcp" && server.Args[1] == "serve" {
			t.Error("Found incorrect command ['mcp', 'serve']. Should use ['mcp-server'] instead.")
		}
	}

	// Check that cwd is set
	expectedCwd := "${workspaceFolder}"
	if server.Cwd != expectedCwd {
		t.Errorf("Expected cwd '%s', got '%s'", expectedCwd, server.Cwd)
	}
}
