// This file contains MCP (Model Context Protocol) helper utilities.
// These utilities support binary path resolution used by both the MCP server
// and MCP validation logic.

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var mcpHelpersLog = logger.New("cli:mcp_helpers")

// GetBinaryPath returns the path to the currently running gh-aw binary.
// This is used by the MCP server to determine where the gh-aw binary is located
// when launching itself with different arguments.
//
// Returns the absolute path to the binary, or an error if the path cannot be determined.
func GetBinaryPath() (string, error) {
	// Get the path to the currently running executable
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks to get the actual binary path
	// This is important because gh extensions are typically symlinked
	// Note: EvalSymlinks already returns an absolute path
	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		mcpHelpersLog.Printf("Warning: failed to resolve symlinks for %s: %v", exePath, err)
		return exePath, nil
	}

	return resolvedPath, nil
}

// boolPtr returns a pointer to the given bool value, used for optional *bool fields.
func boolPtr(b bool) *bool { return typeutil.Ptr(b) }

// logAndValidateBinaryPath determines the binary path, logs it, and validates it exists.
// Returns the detected binary path and an error if the path cannot be determined or if the file doesn't exist.
// This is a helper used by both runMCPServer and validateMCPServerConfiguration.
// Diagnostics are emitted through the debug logger only.
func logAndValidateBinaryPath() (string, error) {
	binaryPath, err := GetBinaryPath()
	if err != nil {
		mcpHelpersLog.Printf("Warning: failed to get binary path: %v", err)
		return "", err
	}

	// Check if the binary file exists
	if _, err := os.Stat(binaryPath); err != nil {
		if os.IsNotExist(err) {
			mcpHelpersLog.Printf("ERROR: binary file does not exist at path: %s", binaryPath)
			return "", fmt.Errorf("binary file does not exist at path: %s", binaryPath)
		}
		mcpHelpersLog.Printf("Warning: failed to stat binary file at %s: %v", binaryPath, err)
		return "", err
	}

	// Log the binary path for debugging
	mcpHelpersLog.Printf("gh-aw binary path: %s", binaryPath)
	return binaryPath, nil
}
