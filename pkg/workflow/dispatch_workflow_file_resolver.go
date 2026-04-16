package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/parser"
)

// getCurrentWorkflowName extracts the workflow name from the file path
func getCurrentWorkflowName(workflowPath string) string {
	filename := filepath.Base(workflowPath)
	// Remove .md or .lock.yml extension
	filename = strings.TrimSuffix(filename, ".md")
	filename = strings.TrimSuffix(filename, ".lock.yml")
	return filename
}

// isPathWithinDir checks if a path is within a given directory (prevents path traversal)
func isPathWithinDir(path, dir string) bool {
	// Get absolute paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	// Get the relative path from dir to path
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}

	// Check if the relative path tries to go outside the directory
	// If it starts with "..", it's trying to escape
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

// findWorkflowFileResult holds the result of finding a workflow file
type findWorkflowFileResult struct {
	mdPath     string
	lockPath   string
	ymlPath    string
	mdExists   bool
	lockExists bool
	ymlExists  bool
}

// findWorkflowFile searches for a workflow file in .github/workflows directory only
// Returns paths and existence flags for .md, .lock.yml, and .yml files
func findWorkflowFile(workflowName string, currentWorkflowPath string) (*findWorkflowFileResult, error) {
	dispatchWorkflowValidationLog.Printf("Finding workflow file: name=%s, current_path=%s", workflowName, currentWorkflowPath)
	result := &findWorkflowFileResult{}

	// Get the current workflow's directory
	currentDir := filepath.Dir(currentWorkflowPath)

	// Get repo root by going up from current directory
	// Assume structure: <repo-root>/.github/workflows/file.md or <repo-root>/.github/aw/file.md
	githubDir := filepath.Dir(currentDir) // .github
	repoRoot := filepath.Dir(githubDir)   // repo root

	// Only search in .github/workflows (standard GitHub Actions location)
	searchDir := filepath.Join(repoRoot, ".github", "workflows")

	// Build paths for the workflows directory
	mdPath := filepath.Clean(filepath.Join(searchDir, workflowName+".md"))
	lockPath := filepath.Clean(filepath.Join(searchDir, workflowName+".lock.yml"))
	ymlPath := filepath.Clean(filepath.Join(searchDir, workflowName+".yml"))

	// Validate paths are within the search directory (prevent path traversal)
	if !isPathWithinDir(mdPath, searchDir) || !isPathWithinDir(lockPath, searchDir) || !isPathWithinDir(ymlPath, searchDir) {
		return result, fmt.Errorf("invalid workflow name '%s' (path traversal not allowed)", workflowName)
	}

	// Check which files exist
	result.mdPath = mdPath
	result.lockPath = lockPath
	result.ymlPath = ymlPath
	result.mdExists = fileutil.FileExists(mdPath)
	result.lockExists = fileutil.FileExists(lockPath)
	result.ymlExists = fileutil.FileExists(ymlPath)

	dispatchWorkflowValidationLog.Printf("Workflow file search results: md_exists=%v, lock_exists=%v, yml_exists=%v", result.mdExists, result.lockExists, result.ymlExists)
	return result, nil
}

// mdHasWorkflowDispatch reads a .md workflow file's frontmatter and reports whether
// the workflow includes a workflow_dispatch trigger in its 'on:' section.
// This is used to validate same-batch dispatch-workflow targets whose .lock.yml has
// not yet been generated.
func mdHasWorkflowDispatch(mdPath string) (bool, error) {
	dispatchWorkflowValidationLog.Printf("Checking for workflow_dispatch trigger in: %s", mdPath)
	content, err := os.ReadFile(mdPath) // #nosec G304 -- mdPath is validated via isPathWithinDir in findWorkflowFile
	if err != nil {
		return false, err
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result == nil {
		return false, err
	}
	onSection, hasOn := result.Frontmatter["on"]
	if !hasOn {
		return false, nil
	}
	return containsWorkflowDispatch(onSection), nil
}

// extractMDWorkflowDispatchInputs reads a .md workflow file's frontmatter and extracts
// the workflow_dispatch inputs schema, mirroring extractWorkflowDispatchInputs for .md sources.
func extractMDWorkflowDispatchInputs(mdPath string) (map[string]any, error) {
	content, err := os.ReadFile(mdPath) // #nosec G304 -- mdPath is validated via isPathWithinDir in findWorkflowFile
	if err != nil {
		return nil, err
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result == nil {
		return make(map[string]any), nil
	}
	onSection, hasOn := result.Frontmatter["on"]
	if !hasOn {
		return make(map[string]any), nil
	}
	onMap, ok := onSection.(map[string]any)
	if !ok {
		return make(map[string]any), nil
	}
	workflowDispatch, hasWorkflowDispatch := onMap["workflow_dispatch"]
	if !hasWorkflowDispatch {
		return make(map[string]any), nil
	}
	workflowDispatchMap, ok := workflowDispatch.(map[string]any)
	if !ok {
		return make(map[string]any), nil
	}
	inputs, hasInputs := workflowDispatchMap["inputs"]
	if !hasInputs {
		return make(map[string]any), nil
	}
	inputsMap, ok := inputs.(map[string]any)
	if !ok {
		return make(map[string]any), nil
	}
	return inputsMap, nil
}
