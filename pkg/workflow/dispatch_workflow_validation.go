package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var dispatchWorkflowValidationLog = newValidationLogger("dispatch_workflow")

// validateDispatchWorkflow validates that the dispatch-workflow configuration is correct
func (c *Compiler) validateDispatchWorkflow(data *WorkflowData, workflowPath string) error {
	dispatchWorkflowValidationLog.Print("Starting dispatch-workflow validation")

	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		dispatchWorkflowValidationLog.Print("No dispatch-workflow configuration found")
		return nil
	}

	config := data.SafeOutputs.DispatchWorkflow

	if len(config.Workflows) == 0 {
		return errors.New("dispatch-workflow: must specify at least one workflow in the list\n\nExample configuration in workflow frontmatter:\nsafe-outputs:\n  dispatch-workflow:\n    workflows: [workflow-name-1, workflow-name-2]\n\nWorkflow names should match the filename without the .md extension")
	}

	// Get the current workflow name for self-reference check
	currentWorkflowName := getCurrentWorkflowName(workflowPath)
	dispatchWorkflowValidationLog.Printf("Current workflow name: %s", currentWorkflowName)

	// Collect all validation errors using ErrorCollector
	collector := NewErrorCollector(c.failFast)

	for _, workflowName := range config.Workflows {
		dispatchWorkflowValidationLog.Printf("Validating workflow: %s", workflowName)

		// Check for self-reference
		if workflowName == currentWorkflowName {
			selfRefErr := fmt.Errorf("dispatch-workflow: self-reference not allowed (workflow '%s' cannot dispatch itself)\n\nA workflow cannot trigger itself to prevent infinite loops.\nIf you need recurring execution, use a schedule trigger or workflow_dispatch instead", workflowName)
			if returnErr := collector.Add(selfRefErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
			continue // Skip further validation for this workflow
		}

		// Find the workflow file in multiple locations
		fileResult, err := findWorkflowFile(workflowName, workflowPath)
		if err != nil {
			findErr := fmt.Errorf("dispatch-workflow: error finding workflow '%s': %w", workflowName, err)
			if returnErr := collector.Add(findErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
			continue // Skip further validation for this workflow
		}

		// Check if any workflow file exists
		if !fileResult.mdExists && !fileResult.lockExists && !fileResult.ymlExists {
			// Provide helpful error message showing .github/workflows location
			currentDir := filepath.Dir(workflowPath)
			githubDir := filepath.Dir(currentDir)
			repoRoot := filepath.Dir(githubDir)
			workflowsDir := filepath.Join(repoRoot, ".github", "workflows")

			notFoundErr := fmt.Errorf("dispatch-workflow: workflow '%s' not found in %s\n\nChecked for: %s.md, %s.lock.yml, %s.yml\n\nTo fix:\n1. Verify the workflow file exists in .github/workflows/\n2. Ensure the filename matches exactly (case-sensitive)\n3. Use the filename without extension in your configuration", workflowName, workflowsDir, workflowName, workflowName, workflowName)
			if returnErr := collector.Add(notFoundErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
			continue // Skip further validation for this workflow
		}

		// Validate that the workflow supports workflow_dispatch
		// Priority: .lock.yml (compiled agentic workflow) > .yml (standard GitHub Actions) > .md (needs compilation)
		var workflowContent []byte // #nosec G304 -- All file paths are validated via isPathWithinDir() before use
		var workflowFile string
		var readErr error

		if fileResult.lockExists {
			workflowFile = fileResult.lockPath
			workflowContent, readErr = os.ReadFile(fileResult.lockPath) // #nosec G304 -- Path is validated via isPathWithinDir in findWorkflowFile
			if readErr != nil {
				fileReadErr := fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", fileResult.lockPath, readErr)
				if returnErr := collector.Add(fileReadErr); returnErr != nil {
					return returnErr // Fail-fast mode
				}
				continue // Skip further validation for this workflow
			}
		} else if fileResult.ymlExists {
			workflowFile = fileResult.ymlPath
			workflowContent, readErr = os.ReadFile(fileResult.ymlPath) // #nosec G304 -- Path is validated via isPathWithinDir in findWorkflowFile
			if readErr != nil {
				fileReadErr := fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", fileResult.ymlPath, readErr)
				if returnErr := collector.Add(fileReadErr); returnErr != nil {
					return returnErr // Fail-fast mode
				}
				continue // Skip further validation for this workflow
			}
		} else {
			// Only .md exists — it may be a same-batch compilation target.
			// Validate via the .md frontmatter so a second compile pass is not required.
			mdHasDispatch, checkErr := mdHasWorkflowDispatch(fileResult.mdPath)
			if checkErr != nil {
				readErr := fmt.Errorf("dispatch-workflow: failed to read workflow source %s: %w", fileResult.mdPath, checkErr)
				if returnErr := collector.Add(readErr); returnErr != nil {
					return returnErr // Fail-fast mode
				}
				continue // Skip further validation for this workflow
			}
			if !mdHasDispatch {
				dispatchErr := fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName)
				if returnErr := collector.Add(dispatchErr); returnErr != nil {
					return returnErr // Fail-fast mode
				}
				continue // Skip further validation for this workflow
			}
			// .md exists with workflow_dispatch — valid same-batch compilation target.
			dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found .md source at %s with workflow_dispatch trigger)", workflowName, fileResult.mdPath)
			continue // Trigger validated; skip YAML-specific checks below
		}

		// Parse the workflow YAML to check for workflow_dispatch trigger
		var workflow map[string]any
		if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
			parseErr := fmt.Errorf("dispatch-workflow: failed to parse workflow file %s: %w", workflowFile, err)
			if returnErr := collector.Add(parseErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
			continue // Skip further validation for this workflow
		}

		// Check if the workflow has an "on" section
		onSection, hasOn := workflow["on"]
		if !hasOn {
			onSectionErr := fmt.Errorf("dispatch-workflow: workflow '%s' does not have an 'on' trigger section", workflowName)
			if returnErr := collector.Add(onSectionErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
			continue // Skip further validation for this workflow
		}

		if !containsWorkflowDispatch(onSection) {
			dispatchErr := fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName)
			if returnErr := collector.Add(dispatchErr); returnErr != nil {
				return returnErr // Fail-fast mode
			}
			continue // Skip further validation for this workflow
		}

		dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found in %s)", workflowName, workflowFile)
	}

	dispatchWorkflowValidationLog.Printf("Dispatch workflow validation completed: error_count=%d, total_workflows=%d", collector.Count(), len(config.Workflows))

	// Return aggregated errors with formatted output
	return collector.FormattedError("dispatch-workflow")
}

// extractWorkflowDispatchInputs parses a workflow file and extracts the workflow_dispatch inputs schema
// Returns a map of input definitions that can be used to generate MCP tool schemas
func extractWorkflowDispatchInputs(workflowPath string) (map[string]any, error) {
	dispatchWorkflowValidationLog.Printf("Extracting workflow_dispatch inputs from: %s", workflowPath)
	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(workflowPath)
	if !filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("workflow path must be absolute: %s", workflowPath)
	}

	workflowContent, err := os.ReadFile(cleanPath) // #nosec G304 -- Path is sanitized above
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file %s: %w", workflowPath, err)
	}

	var workflow map[string]any
	if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow file %s: %w", workflowPath, err)
	}

	// Navigate to workflow_dispatch.inputs
	onSection, hasOn := workflow["on"]
	if !hasOn {
		return make(map[string]any), nil // No inputs
	}

	onMap, ok := onSection.(map[string]any)
	if !ok {
		return make(map[string]any), nil // No inputs
	}

	workflowDispatch, hasWorkflowDispatch := onMap["workflow_dispatch"]
	if !hasWorkflowDispatch {
		return make(map[string]any), nil // No inputs
	}

	workflowDispatchMap, ok := workflowDispatch.(map[string]any)
	if !ok {
		return make(map[string]any), nil // No inputs
	}

	inputs, hasInputs := workflowDispatchMap["inputs"]
	if !hasInputs {
		return make(map[string]any), nil // No inputs
	}

	inputsMap, ok := inputs.(map[string]any)
	if !ok {
		return make(map[string]any), nil // No inputs
	}

	return inputsMap, nil
}

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

// containsWorkflowDispatch reports whether the given 'on:' section value includes
// a workflow_dispatch trigger.  It handles the three GitHub Actions forms:
//   - string:     "on: workflow_dispatch"
//   - []any:      "on: [push, workflow_dispatch]"
//   - map[string]any: "on:\n  workflow_dispatch: ..."
func containsWorkflowDispatch(onSection any) bool {
	return containsTrigger(onSection, "workflow_dispatch")
}
