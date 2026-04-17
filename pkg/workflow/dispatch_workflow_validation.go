package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

	currentWorkflowName := getCurrentWorkflowName(workflowPath)
	dispatchWorkflowValidationLog.Printf("Current workflow name: %s", currentWorkflowName)
	collector := NewErrorCollector(c.failFast)

	for _, workflowName := range config.Workflows {
		dispatchWorkflowValidationLog.Printf("Validating workflow: %s", workflowName)
		if workflowName == currentWorkflowName {
			selfRefErr := fmt.Errorf("dispatch-workflow: self-reference not allowed (workflow '%s' cannot dispatch itself)\n\nA workflow cannot trigger itself to prevent infinite loops.\nIf you need recurring execution, use a schedule trigger or workflow_dispatch instead", workflowName)
			if returnErr := collector.Add(selfRefErr); returnErr != nil {
				return returnErr
			}
			continue
		}
		fileResult, err := findWorkflowFile(workflowName, workflowPath)
		if err != nil {
			findErr := fmt.Errorf("dispatch-workflow: error finding workflow '%s': %w", workflowName, err)
			if returnErr := collector.Add(findErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		if !fileResult.mdExists && !fileResult.lockExists && !fileResult.ymlExists {
			currentDir := filepath.Dir(workflowPath)
			githubDir := filepath.Dir(currentDir)
			repoRoot := filepath.Dir(githubDir)
			workflowsDir := filepath.Join(repoRoot, ".github", "workflows")
			notFoundErr := fmt.Errorf("dispatch-workflow: workflow '%s' not found in %s\n\nChecked for: %s.md, %s.lock.yml, %s.yml\n\nTo fix:\n1. Verify the workflow file exists in .github/workflows/\n2. Ensure the filename matches exactly (case-sensitive)\n3. Use the filename without extension in your configuration", workflowName, workflowsDir, workflowName, workflowName, workflowName)
			if returnErr := collector.Add(notFoundErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		var workflowContent []byte // #nosec G304 -- All file paths are validated via isPathWithinDir before use
		var workflowFile string
		var readErr error

		if fileResult.lockExists {
			workflowFile = fileResult.lockPath
			workflowContent, readErr = os.ReadFile(fileResult.lockPath) // #nosec G304 -- Path is validated via isPathWithinDir in findWorkflowFile
			if readErr != nil {
				fileReadErr := fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", fileResult.lockPath, readErr)
				if returnErr := collector.Add(fileReadErr); returnErr != nil {
					return returnErr
				}
				continue
			}
		} else if fileResult.ymlExists {
			workflowFile = fileResult.ymlPath
			workflowContent, readErr = os.ReadFile(fileResult.ymlPath) // #nosec G304 -- Path is validated via isPathWithinDir in findWorkflowFile
			if readErr != nil {
				fileReadErr := fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", fileResult.ymlPath, readErr)
				if returnErr := collector.Add(fileReadErr); returnErr != nil {
					return returnErr
				}
				continue
			}
		} else {
			mdHasDispatch, checkErr := mdHasWorkflowDispatch(fileResult.mdPath)
			if checkErr != nil {
				readErr := fmt.Errorf("dispatch-workflow: failed to read workflow source %s: %w", fileResult.mdPath, checkErr)
				if returnErr := collector.Add(readErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			if !mdHasDispatch {
				dispatchErr := fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName)
				if returnErr := collector.Add(dispatchErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found .md source at %s with workflow_dispatch trigger)", workflowName, fileResult.mdPath)
			continue
		}

		var workflow map[string]any
		if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
			parseErr := fmt.Errorf("dispatch-workflow: failed to parse workflow file %s: %w", workflowFile, err)
			if returnErr := collector.Add(parseErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		onSection, hasOn := workflow["on"]
		if !hasOn {
			onSectionErr := fmt.Errorf("dispatch-workflow: workflow '%s' does not have an 'on' trigger section", workflowName)
			if returnErr := collector.Add(onSectionErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		if !containsWorkflowDispatch(onSection) {
			dispatchErr := fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName)
			if returnErr := collector.Add(dispatchErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found in %s)", workflowName, workflowFile)
	}

	dispatchWorkflowValidationLog.Printf("Dispatch workflow validation completed: error_count=%d, total_workflows=%d", collector.Count(), len(config.Workflows))

	return collector.FormattedError("dispatch-workflow")
}

// extractWorkflowDispatchInputs parses a workflow file and extracts the workflow_dispatch inputs schema
// Returns a map of input definitions that can be used to generate MCP tool schemas
func extractWorkflowDispatchInputs(workflowPath string) (map[string]any, error) {
	dispatchWorkflowValidationLog.Printf("Extracting workflow_dispatch inputs from: %s", workflowPath)
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

	onSection, hasOn := workflow["on"]
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
