package workflow

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed js/safe_outputs_mcp_server.js
var safeOutputsMCPServerJS string

// SafeOutputsMCPServerScript contains the compiled JavaScript MCP server
// that implements all safe output types as MCP tools
func SafeOutputsMCPServerScript() string {
	return safeOutputsMCPServerJS
}

// WriteSafeOutputsMCPServerToTemp writes the embedded MCP server JavaScript
// to a temporary file and returns the path
func WriteSafeOutputsMCPServerToTemp() (string, error) {
	// Create temp directory
	tempDir := "/tmp/safe-outputs-mcp"
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write the JavaScript file
	serverPath := filepath.Join(tempDir, "safe_outputs_mcp_server.js")
	err = os.WriteFile(serverPath, []byte(safeOutputsMCPServerJS), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write MCP server file: %w", err)
	}

	return serverPath, nil
}
