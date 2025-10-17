package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveWorkflowPath resolves a workflow file path from various formats:
// - Absolute path to .md file
// - Relative path to .md file
// - Workflow name or subpath (e.g., "a.md" -> ".github/workflows/a.md", "shared/b.md" -> ".github/workflows/shared/b.md")
func ResolveWorkflowPath(workflowFile string) (string, error) {
	workflowsDir := ".github/workflows"

	// Add .md extension if not present
	searchPath := workflowFile
	if !strings.HasSuffix(searchPath, ".md") {
		searchPath += ".md"
	}

	// 1. If it's a path that exists as-is (absolute or relative), use it
	if _, err := os.Stat(searchPath); err == nil {
		return searchPath, nil
	}

	// 2. Try exact relative path under .github/workflows
	workflowPath := filepath.Join(workflowsDir, searchPath)
	if _, err := os.Stat(workflowPath); err == nil {
		return workflowPath, nil
	}

	// No matches found
	return "", fmt.Errorf("workflow file not found: %s", workflowPath)
}

// NormalizeWorkflowFile normalizes a workflow file name by adding .md extension if missing
func NormalizeWorkflowFile(workflowFile string) string {
	if !strings.HasSuffix(workflowFile, ".md") {
		return workflowFile + ".md"
	}
	return workflowFile
}
