package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
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

// findWorkflowFile searches for a workflow file in the configured workflows directory only.
// Returns paths and existence flags for .md, .lock.yml, and .yml files
func findWorkflowFile(workflowName string, currentWorkflowPath string) (*findWorkflowFileResult, error) {
	dispatchWorkflowValidationLog.Printf("Finding workflow file: name=%s, current_path=%s", workflowName, currentWorkflowPath)
	result := &findWorkflowFileResult{}

	// Get the current workflow's directory
	currentDir := filepath.Dir(currentWorkflowPath)

	// Get repo root by going up from the current workflow directory.
	// Assume structure: <repo-root>/<configured-workflows-dir>/file.md or <repo-root>/.github/aw/file.md.
	githubDir := filepath.Dir(currentDir) // .github
	repoRoot := filepath.Dir(githubDir)   // repo root

	// Only search in the configured workflows directory.
	searchDir := filepath.Join(repoRoot, constants.GetWorkflowDir())

	// Build paths for the workflows directory
	mdPath := filepath.Clean(filepath.Join(searchDir, workflowName+".md"))
	lockPath := filepath.Clean(filepath.Join(searchDir, workflowName+".lock.yml"))
	ymlPath := filepath.Clean(filepath.Join(searchDir, workflowName+".yml"))

	// Validate paths are within the search directory (prevent path traversal)
	if !isPathWithinDir(mdPath, searchDir) || !isPathWithinDir(lockPath, searchDir) || !isPathWithinDir(ymlPath, searchDir) {
		dispatchWorkflowValidationLog.Printf("Rejecting workflow name '%s': resolved paths escape search dir %s", workflowName, searchDir)
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
		dispatchWorkflowValidationLog.Printf("Failed to read %s: %v", mdPath, err)
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
	dispatchWorkflowValidationLog.Printf("Extracting workflow_dispatch inputs from: %s", mdPath)
	inputs, err := extractInputsFromMarkdown(mdPath, "workflow_dispatch")
	if err != nil {
		return nil, err
	}
	dispatchWorkflowValidationLog.Printf("Extracted %d workflow_dispatch input(s) from: %s", len(inputs), mdPath)
	return inputs, nil
}

// workflowFileValidation holds the result of readAndValidateWorkflowFileTrigger.
type workflowFileValidation struct {
	// filePath is the compiled file that was read (.lock.yml or .yml).
	// Empty when neither file exists (md-only case).
	filePath string
	// readErr is set when the file could not be read.
	readErr error
	// parseErr is set when the file content could not be parsed as YAML.
	parseErr error
	// noOnSection is true when the workflow has no 'on:' section at all.
	noOnSection bool
	// hasTrigger is true when the workflow's 'on:' section includes triggerName.
	hasTrigger bool
}

// readAndValidateWorkflowFileTrigger reads the compiled workflow file (preferring
// .lock.yml over .yml) from fileResult and checks whether it declares triggerName.
// Returns an empty workflowFileValidation (filePath == "") when neither compiled
// file exists; callers handle the .md-only case separately.
func readAndValidateWorkflowFileTrigger(fileResult *findWorkflowFileResult, triggerName string) *workflowFileValidation {
	result := &workflowFileValidation{}

	if !fileResult.lockExists && !fileResult.ymlExists {
		return result // md-only case — caller handles it
	}

	if fileResult.lockExists {
		result.filePath = fileResult.lockPath
	} else {
		result.filePath = fileResult.ymlPath
	}

	content, err := os.ReadFile(result.filePath) // #nosec G304 -- paths are validated via isPathWithinDir() in findWorkflowFile()
	if err != nil {
		result.readErr = err
		return result
	}

	var workflow map[string]any
	if err = yaml.Unmarshal(content, &workflow); err != nil {
		result.parseErr = err
		return result
	}

	onSection, hasOn := workflow["on"]
	if !hasOn {
		result.noOnSection = true
		return result
	}

	result.hasTrigger = containsTrigger(onSection, triggerName)
	return result
}

