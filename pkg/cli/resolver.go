package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var resolverLog = logger.New("cli:resolver")

// ResolveWorkflowPath resolves a workflow file path from various formats:
// - Absolute path to .md file
// - Relative path to .md file
// - Workflow name or subpath (e.g., "a.md" -> ".github/workflows/a.md", "shared/b.md" -> ".github/workflows/shared/b.md")
func ResolveWorkflowPath(workflowFile string) (string, error) {
	resolverLog.Printf("Resolving workflow path: %s", workflowFile)
	workflowsDir := ".github/workflows"

	// Add .md extension if not present
	searchPath := workflowFile
	if !strings.HasSuffix(searchPath, ".md") {
		searchPath += ".md"
	}

	// 1. If it's a path that exists as-is (absolute or relative), use it
	if _, err := os.Stat(searchPath); err == nil {
		resolverLog.Printf("Found workflow at direct path: %s", searchPath)
		return searchPath, nil
	}

	// 2. Try exact relative path under .github/workflows
	workflowPath := filepath.Join(workflowsDir, searchPath)
	if _, err := os.Stat(workflowPath); err == nil {
		resolverLog.Printf("Found workflow at: %s", workflowPath)
		return workflowPath, nil
	}

	// No matches found
	resolverLog.Printf("Workflow file not found: %s", workflowPath)
	return "", fmt.Errorf("workflow file not found: %s", workflowPath)
}

// NormalizeWorkflowFile normalizes a workflow file name by adding .md extension if missing
func NormalizeWorkflowFile(workflowFile string) string {
	if !strings.HasSuffix(workflowFile, ".md") {
		return workflowFile + ".md"
	}
	return workflowFile
}
