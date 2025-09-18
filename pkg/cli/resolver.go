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
// - Workflow name (adds .md extension and looks in .github/workflows)
// - Workflow name with .md extension
func ResolveWorkflowPath(workflowFile string) (string, error) {
	workflowsDir := ".github/workflows"
	var workflowPath string

	if strings.HasSuffix(workflowFile, ".md") {
		// If it's already a .md file, use it directly if it exists
		if _, err := os.Stat(workflowFile); err == nil {
			workflowPath = workflowFile
		} else {
			// Try in workflows directory
			workflowPath = filepath.Join(workflowsDir, workflowFile)
		}
	} else {
		// Add .md extension and look in workflows directory
		workflowPath = filepath.Join(workflowsDir, workflowFile+".md")
	}

	// Verify the workflow file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		return "", fmt.Errorf("workflow file not found: %s", workflowPath)
	}

	return workflowPath, nil
}

// NormalizeWorkflowFile normalizes a workflow file name by adding .md extension if missing
func NormalizeWorkflowFile(workflowFile string) string {
	if !strings.HasSuffix(workflowFile, ".md") {
		return workflowFile + ".md"
	}
	return workflowFile
}
