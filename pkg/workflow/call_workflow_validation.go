package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var callWorkflowValidationLog = newValidationLogger("call_workflow")

// validateCallWorkflow validates that the call-workflow configuration is correct.
// It checks that each workflow exists, declares a workflow_call trigger, and is not
// a self-reference.
func (c *Compiler) validateCallWorkflow(data *WorkflowData, workflowPath string) error {
	callWorkflowValidationLog.Print("Starting call-workflow validation")

	if data.SafeOutputs == nil || data.SafeOutputs.CallWorkflow == nil {
		callWorkflowValidationLog.Print("No call-workflow configuration found")
		return nil
	}

	config := data.SafeOutputs.CallWorkflow

	if len(config.Workflows) == 0 {
		return errors.New("call-workflow: must specify at least one workflow in the list\n\nExample configuration in workflow frontmatter:\nsafe-outputs:\n  call-workflow:\n    workflows: [workflow-name-1, workflow-name-2]\n\nWorkflow names should match the filename without the .md extension")
	}

	// Check for duplicate workflow names — each name must appear exactly once
	seen := make(map[string]int, len(config.Workflows))
	for i, name := range config.Workflows {
		if prev, exists := seen[name]; exists {
			return fmt.Errorf("call-workflow: duplicate workflow name '%s' at index %d (first seen at index %d)\n\nEach workflow may appear only once in the list", name, i, prev)
		}
		seen[name] = i
	}

	// Get the current workflow name for self-reference check
	currentWorkflowName := getCurrentWorkflowName(workflowPath)
	callWorkflowValidationLog.Printf("Current workflow name: %s", currentWorkflowName)

	// Collect all validation errors using ErrorCollector
	collector := NewErrorCollector(c.failFast)

	for _, workflowName := range config.Workflows {
		callWorkflowValidationLog.Printf("Validating workflow: %s", workflowName)

		// Check for self-reference
		if workflowName == currentWorkflowName {
			selfRefErr := fmt.Errorf("call-workflow: self-reference not allowed (workflow '%s' cannot call itself)\n\nA workflow cannot call itself to prevent infinite loops.\nUse a separate worker workflow for the task instead", workflowName)
			if returnErr := collector.Add(selfRefErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		// Find the workflow file in multiple locations
		fileResult, err := findWorkflowFile(workflowName, workflowPath)
		if err != nil {
			findErr := fmt.Errorf("call-workflow: error finding workflow '%s': %w", workflowName, err)
			if returnErr := collector.Add(findErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		// Check if any workflow file exists
		if !fileResult.mdExists && !fileResult.lockExists && !fileResult.ymlExists {
			currentDir := filepath.Dir(workflowPath)
			githubDir := filepath.Dir(currentDir)
			repoRoot := filepath.Dir(githubDir)
			workflowsDir := filepath.Join(repoRoot, ".github", "workflows")

			notFoundErr := fmt.Errorf("call-workflow: workflow '%s' not found in %s\n\nChecked for: %s.md, %s.lock.yml, %s.yml\n\nTo fix:\n1. Verify the workflow file exists in .github/workflows/\n2. Ensure the filename matches exactly (case-sensitive)\n3. Use the filename without extension in your configuration", workflowName, workflowsDir, workflowName, workflowName, workflowName)
			if returnErr := collector.Add(notFoundErr); returnErr != nil {
				return returnErr
			}
			continue
		}

		// Validate that the workflow supports workflow_call.
		// Priority: .lock.yml > .yml > .md (same-batch compilation target)
		if fileResult.lockExists {
			workflowContent, readErr := os.ReadFile(fileResult.lockPath) // #nosec G304 -- lockPath is validated via isPathWithinDir() in findWorkflowFile() before being returned
			if readErr != nil {
				fileReadErr := fmt.Errorf("call-workflow: failed to read workflow file %s: %w", fileResult.lockPath, readErr)
				if returnErr := collector.Add(fileReadErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			var workflow map[string]any
			if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
				parseErr := fmt.Errorf("call-workflow: failed to parse workflow file %s: %w", fileResult.lockPath, err)
				if returnErr := collector.Add(parseErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			onSection, hasOn := workflow["on"]
			if !hasOn {
				onErr := fmt.Errorf("call-workflow: workflow '%s' does not have an 'on' trigger section", workflowName)
				if returnErr := collector.Add(onErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			if !containsWorkflowCall(onSection) {
				callErr := fmt.Errorf("call-workflow: workflow '%s' does not support workflow_call trigger (must include 'workflow_call' in the 'on' section)", workflowName)
				if returnErr := collector.Add(callErr); returnErr != nil {
					return returnErr
				}
				continue
			}
		} else if fileResult.ymlExists {
			workflowContent, readErr := os.ReadFile(fileResult.ymlPath) // #nosec G304 -- ymlPath is validated via isPathWithinDir() in findWorkflowFile() before being returned
			if readErr != nil {
				fileReadErr := fmt.Errorf("call-workflow: failed to read workflow file %s: %w", fileResult.ymlPath, readErr)
				if returnErr := collector.Add(fileReadErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			var workflow map[string]any
			if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
				parseErr := fmt.Errorf("call-workflow: failed to parse workflow file %s: %w", fileResult.ymlPath, err)
				if returnErr := collector.Add(parseErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			onSection, hasOn := workflow["on"]
			if !hasOn {
				onErr := fmt.Errorf("call-workflow: workflow '%s' does not have an 'on' trigger section", workflowName)
				if returnErr := collector.Add(onErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			if !containsWorkflowCall(onSection) {
				callErr := fmt.Errorf("call-workflow: workflow '%s' does not support workflow_call trigger (must include 'workflow_call' in the 'on' section)", workflowName)
				if returnErr := collector.Add(callErr); returnErr != nil {
					return returnErr
				}
				continue
			}
		} else {
			// Only .md exists — it may be a same-batch compilation target.
			// Validate via the .md frontmatter so a second compile pass is not required.
			mdHasCall, checkErr := mdHasWorkflowCall(fileResult.mdPath)
			if checkErr != nil {
				readErr := fmt.Errorf("call-workflow: failed to read workflow source %s: %w", fileResult.mdPath, checkErr)
				if returnErr := collector.Add(readErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			if !mdHasCall {
				callErr := fmt.Errorf("call-workflow: workflow '%s' does not support workflow_call trigger (must include 'workflow_call' in the 'on' section)", workflowName)
				if returnErr := collector.Add(callErr); returnErr != nil {
					return returnErr
				}
				continue
			}
			// .md exists with workflow_call — valid same-batch compilation target.
			callWorkflowValidationLog.Printf("Workflow '%s' is valid for call-workflow (found .md source at %s with workflow_call trigger)", workflowName, fileResult.mdPath)
			continue
		}

		callWorkflowValidationLog.Printf("Workflow '%s' is valid for call-workflow", workflowName)
	}

	callWorkflowValidationLog.Printf("Call workflow validation completed: error_count=%d, total_workflows=%d", collector.Count(), len(config.Workflows))

	return collector.FormattedError("call-workflow")
}

// extractWorkflowCallInputs parses a workflow file and extracts the workflow_call inputs schema.
// Returns a map of input definitions that can be used to generate MCP tool schemas.
func extractWorkflowCallInputs(workflowPath string) (map[string]any, error) {
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

	return extractWorkflowCallInputsFromParsed(workflow), nil
}

// extractMDWorkflowCallInputs reads a .md workflow file's frontmatter and extracts
// the workflow_call inputs schema, mirroring extractWorkflowCallInputs for .md sources.
func extractMDWorkflowCallInputs(mdPath string) (map[string]any, error) {
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
	workflowCall, hasWorkflowCall := onMap["workflow_call"]
	if !hasWorkflowCall {
		return make(map[string]any), nil
	}
	workflowCallMap, ok := workflowCall.(map[string]any)
	if !ok {
		return make(map[string]any), nil
	}
	inputs, hasInputs := workflowCallMap["inputs"]
	if !hasInputs {
		return make(map[string]any), nil
	}
	inputsMap, ok := inputs.(map[string]any)
	if !ok {
		return make(map[string]any), nil
	}
	return inputsMap, nil
}

// extractWorkflowCallInputsFromParsed extracts workflow_call inputs from an already-parsed
// workflow map (used for both .lock.yml and .yml files).
func extractWorkflowCallInputsFromParsed(workflow map[string]any) map[string]any {
	onSection, hasOn := workflow["on"]
	if !hasOn {
		return make(map[string]any)
	}
	onMap, ok := onSection.(map[string]any)
	if !ok {
		return make(map[string]any)
	}
	workflowCall, hasWorkflowCall := onMap["workflow_call"]
	if !hasWorkflowCall {
		return make(map[string]any)
	}
	workflowCallMap, ok := workflowCall.(map[string]any)
	if !ok {
		return make(map[string]any)
	}
	inputs, hasInputs := workflowCallMap["inputs"]
	if !hasInputs {
		return make(map[string]any)
	}
	inputsMap, ok := inputs.(map[string]any)
	if !ok {
		return make(map[string]any)
	}
	return inputsMap
}

// mdHasWorkflowCall reads a .md workflow file's frontmatter and reports whether
// the workflow includes a workflow_call trigger in its 'on:' section.
func mdHasWorkflowCall(mdPath string) (bool, error) {
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
	return containsWorkflowCall(onSection), nil
}

// containsWorkflowCall reports whether the given 'on:' section value includes
// a workflow_call trigger. It handles the three GitHub Actions forms:
//   - string:          "on: workflow_call"
//   - []any:           "on: [push, workflow_call]"
//   - map[string]any:  "on:\n  workflow_call: ..."
func containsWorkflowCall(onSection any) bool {
	switch on := onSection.(type) {
	case string:
		return on == "workflow_call"
	case []any:
		for _, trigger := range on {
			if triggerStr, ok := trigger.(string); ok && triggerStr == "workflow_call" {
				return true
			}
		}
	case map[string]any:
		_, ok := on["workflow_call"]
		return ok
	}
	return false
}
