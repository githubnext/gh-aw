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
// - Workflow name (adds .md extension and looks in .github/workflows, shared/, and shared/mcp/)
// - Workflow name with .md extension
func ResolveWorkflowPath(workflowFile string) (string, error) {
	workflowsDir := ".github/workflows"
	sharedDir := filepath.Join(workflowsDir, "shared")
	sharedMCPDir := filepath.Join(sharedDir, "mcp")

	var workflowPath string

	if strings.HasSuffix(workflowFile, ".md") {
		// If it's already a .md file, use it directly if it exists
		if _, err := os.Stat(workflowFile); err == nil {
			workflowPath = workflowFile
		} else {
			// Try in workflows directory first
			workflowPath = filepath.Join(workflowsDir, workflowFile)

			// If not found, try shared directories
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				// Try shared directory
				sharedPath := filepath.Join(sharedDir, workflowFile)
				if _, err := os.Stat(sharedPath); err == nil {
					workflowPath = sharedPath
				} else {
					// Try shared/mcp directory
					sharedMCPPath := filepath.Join(sharedMCPDir, workflowFile)
					if _, err := os.Stat(sharedMCPPath); err == nil {
						workflowPath = sharedMCPPath
					}
				}
			}
		}
	} else {
		// Add .md extension and look in workflows directory first
		workflowPath = filepath.Join(workflowsDir, workflowFile+".md")

		// If not found, try shared directories
		if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
			// Try shared directory
			sharedPath := filepath.Join(sharedDir, workflowFile+".md")
			if _, err := os.Stat(sharedPath); err == nil {
				workflowPath = sharedPath
			} else {
				// Try shared/mcp directory
				sharedMCPPath := filepath.Join(sharedMCPDir, workflowFile+".md")
				if _, err := os.Stat(sharedMCPPath); err == nil {
					workflowPath = sharedMCPPath
				}
			}
		}
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
