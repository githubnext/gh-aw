package workflow

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed mcp-safe-outputs/safe_outputs_mcp_server.js
var safeOutputsMCPServerJS string

//go:embed mcp-safe-outputs/mcp_package.json
var mcpPackageJSON string

//go:embed mcp-safe-outputs/mcp_package-lock.json
var mcpPackageLockJSON string

// SafeOutputsMCPServerScript contains the compiled JavaScript MCP server
// that implements all safe output types as MCP tools
func SafeOutputsMCPServerScript() string {
	return safeOutputsMCPServerJS
}

// WriteSafeOutputsMCPServerToTemp writes the embedded MCP server JavaScript
// and npm dependencies to a temporary directory and returns the path
func WriteSafeOutputsMCPServerToTemp() (string, error) {
	// Create temp directory
	tempDir := "/tmp/safe-outputs-mcp"
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write package.json
	packageJSONPath := filepath.Join(tempDir, "package.json")
	err = os.WriteFile(packageJSONPath, []byte(mcpPackageJSON), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write package.json: %w", err)
	}

	// Write package-lock.json
	packageLockPath := filepath.Join(tempDir, "package-lock.json")
	err = os.WriteFile(packageLockPath, []byte(mcpPackageLockJSON), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write package-lock.json: %w", err)
	}

	// Run npm ci to install dependencies
	cmd := exec.Command("npm", "ci")
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to install npm dependencies: %w", err)
	}

	// Write the JavaScript file
	serverPath := filepath.Join(tempDir, "safe_outputs_mcp_server.js")
	err = os.WriteFile(serverPath, []byte(safeOutputsMCPServerJS), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write MCP server file: %w", err)
	}

	return serverPath, nil
}
