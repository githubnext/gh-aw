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
// - Workflow name with subpath (e.g., "shared/serena" or "shared/mcp/serena")
// - Workflow name (searches recursively in .github/workflows)
//
// Resolution order:
// 1. If path exists as-is, use it
// 2. Try exact relative path match under .github/workflows (e.g., "shared/b.md" -> ".github/workflows/shared/b.md")
// 3. Search recursively for files ending with the input path (subpath matching)
// 4. Search recursively for files with matching basename
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
	exactPath := filepath.Join(workflowsDir, searchPath)
	if _, err := os.Stat(exactPath); err == nil {
		return exactPath, nil
	}

	// 3 & 4. Search recursively through .github/workflows
	var matches []string
	var exactSubpathMatches []string

	err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only consider .md files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Get relative path from workflows directory
		relPath, err := filepath.Rel(workflowsDir, path)
		if err != nil {
			return nil
		}

		// Check for exact subpath match (e.g., "shared/mcp/serena.md" matches "shared/mcp/serena.md")
		if relPath == searchPath {
			exactSubpathMatches = append(exactSubpathMatches, path)
			return nil
		}

		// Check for suffix match (e.g., "serena.md" matches ".../shared/mcp/serena.md")
		if strings.HasSuffix(relPath, searchPath) {
			matches = append(matches, path)
			return nil
		}

		// Check for basename match (e.g., "serena.md" matches any "serena.md" in subdirs)
		if filepath.Base(path) == filepath.Base(searchPath) {
			matches = append(matches, path)
			return nil
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching for workflow file: %w", err)
	}

	// Return exact subpath match if found (highest priority)
	if len(exactSubpathMatches) > 0 {
		return exactSubpathMatches[0], nil
	}

	// Return first match if any found
	if len(matches) > 0 {
		return matches[0], nil
	}

	// No matches found
	return "", fmt.Errorf("workflow file not found: %s", searchPath)
}

// NormalizeWorkflowFile normalizes a workflow file name by adding .md extension if missing
func NormalizeWorkflowFile(workflowFile string) string {
	if !strings.HasSuffix(workflowFile, ".md") {
		return workflowFile + ".md"
	}
	return workflowFile
}
