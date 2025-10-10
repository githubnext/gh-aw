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
	expectedArgs := []string{"mcp-server"}
	if len(server.Args) != len(expectedArgs) {
		t.Errorf("Expected %d argument(s), got %d: %v", len(expectedArgs), len(server.Args), server.Args)
	} else if len(server.Args) > 0 && server.Args[0] != expectedArgs[0] {
		t.Errorf("Expected args %v, got %v", expectedArgs, server.Args)
	}

	// Check that cwd is set
	expectedCwd := "${workspaceFolder}"
	if server.Cwd != expectedCwd {
		t.Errorf("Expected cwd '%s', got '%s'", expectedCwd, server.Cwd)
	}
}
