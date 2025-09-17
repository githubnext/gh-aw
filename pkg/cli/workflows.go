package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getPackagesDir returns the packages directory path based on local flag
func getPackagesDir(local bool) (string, error) {
	if local {
		return ".aw/packages", nil
	}

	// Use global directory under user's home
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".aw", "packages"), nil
}

func getWorkflowsDir(customDir ...string) string {
	if len(customDir) > 0 && customDir[0] != "" {
		return customDir[0]
	}
	return ".github/workflows"
}

// readWorkflowFile reads a workflow file from either filesystem
func readWorkflowFile(filePath string, workflowsDir string) ([]byte, string, error) {
	// Using local filesystem
	fullPath := filepath.Join(workflowsDir, filePath)
	if !strings.HasPrefix(fullPath, workflowsDir) {
		// If filePath is already absolute
		fullPath = filePath
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read workflow file %s: %w", fullPath, err)
	}
	return content, fullPath, nil
}
