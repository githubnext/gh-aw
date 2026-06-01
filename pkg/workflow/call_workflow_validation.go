package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
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
	collector := NewErrorCollector(c.failFast)
	for _, workflowName := range config.Workflows {
		callWorkflowValidationLog.Printf("Validating workflow: %s", workflowName)
		if err := validateNotSelfReference(workflowName, currentWorkflowName); err != nil {
			if returnErr := collector.Add(err); returnErr != nil {
				return returnErr
			}
			continue
		}
		fileResult, err := findWorkflowFile(workflowName, workflowPath)
		if err != nil {
			findErr := fmt.Errorf("call-workflow: error finding workflow '%s': %w", workflowName, err)
			if returnErr := collector.Add(findErr); returnErr != nil {
				return returnErr
			}
			continue
		}
		if err := validateWorkflowFileExists(fileResult, workflowName, workflowPath); err != nil {
			if returnErr := collector.Add(err); returnErr != nil {
				return returnErr
			}
			continue
		}
		if err := validateWorkflowSupportsCallTrigger(workflowName, fileResult); err != nil {
			if returnErr := collector.Add(err); returnErr != nil {
				return returnErr
			}
			continue
		}
		callWorkflowValidationLog.Printf("Workflow '%s' is valid for call-workflow", workflowName)
	}
	callWorkflowValidationLog.Printf("Call workflow validation completed: error_count=%d, total_workflows=%d", collector.Count(), len(config.Workflows))
	return collector.FormattedError("call-workflow")
}

func validateNotSelfReference(workflowName, currentWorkflowName string) error {
	if workflowName == currentWorkflowName {
		return fmt.Errorf("call-workflow: self-reference not allowed (workflow '%s' cannot call itself)\n\nA workflow cannot call itself to prevent infinite loops.\nUse a separate worker workflow for the task instead", workflowName)
	}
	return nil
}

func validateWorkflowFileExists(fileResult *findWorkflowFileResult, workflowName, workflowPath string) error {
	if fileResult.mdExists || fileResult.lockExists || fileResult.ymlExists {
		return nil
	}
	currentDir := filepath.Dir(workflowPath)
	githubDir := filepath.Dir(currentDir)
	repoRoot := filepath.Dir(githubDir)
	workflowsDir := filepath.Join(repoRoot, constants.GetWorkflowDir())
	return fmt.Errorf("call-workflow: workflow '%s' not found in %s\n\nChecked for: %s.md, %s.lock.yml, %s.yml\n\nTo fix:\n1. Verify the workflow file exists in %s/\n2. Ensure the filename matches exactly (case-sensitive)\n3. Use the filename without extension in your configuration", workflowName, workflowsDir, workflowName, workflowName, workflowName, workflowsDir)
}

func validateWorkflowSupportsCallTrigger(workflowName string, fileResult *findWorkflowFileResult) error {
	if fileResult.lockExists {
		return validateYAMLWorkflowHasCallTrigger(fileResult.lockPath, workflowName)
	}
	if fileResult.ymlExists {
		return validateYAMLWorkflowHasCallTrigger(fileResult.ymlPath, workflowName)
	}
	return validateMarkdownWorkflowHasCallTrigger(fileResult.mdPath, workflowName)
}

func validateYAMLWorkflowHasCallTrigger(path, workflowName string) error {
	workflowContent, err := os.ReadFile(path) // #nosec G304 -- path is validated via isPathWithinDir() in findWorkflowFile() before being returned
	if err != nil {
		return fmt.Errorf("call-workflow: failed to read workflow file %s: %w", path, err)
	}
	var workflow map[string]any
	if err = yaml.Unmarshal(workflowContent, &workflow); err != nil {
		return fmt.Errorf("call-workflow: failed to parse workflow file %s: %w", path, err)
	}
	onSection, hasOn := workflow["on"]
	if !hasOn {
		return fmt.Errorf("call-workflow: workflow '%s' does not have an 'on' trigger section", workflowName)
	}
	if containsWorkflowCall(onSection) {
		return nil
	}
	return fmt.Errorf("call-workflow: workflow '%s' does not support workflow_call trigger (must include 'workflow_call' in the 'on' section)", workflowName)
}

func validateMarkdownWorkflowHasCallTrigger(path, workflowName string) error {
	mdHasCall, checkErr := mdHasWorkflowCall(path)
	if checkErr != nil {
		return fmt.Errorf("call-workflow: failed to read workflow source %s: %w", path, checkErr)
	}
	if !mdHasCall {
		return fmt.Errorf("call-workflow: workflow '%s' does not support workflow_call trigger (must include 'workflow_call' in the 'on' section)", workflowName)
	}
	callWorkflowValidationLog.Printf("Workflow '%s' is valid for call-workflow (found .md source at %s with workflow_call trigger)", workflowName, path)
	return nil
}

// extractWorkflowCallInputs parses a workflow file and extracts the workflow_call inputs schema.
// Returns a map of input definitions that can be used to generate MCP tool schemas.
func extractWorkflowCallInputs(workflowPath string) (map[string]any, error) {
	return extractInputsFromYAML(workflowPath, "workflow_call")
}

// extractMDWorkflowCallInputs reads a .md workflow file's frontmatter and extracts
// the workflow_call inputs schema, mirroring extractWorkflowCallInputs for .md sources.
func extractMDWorkflowCallInputs(mdPath string) (map[string]any, error) {
	return extractInputsFromMarkdown(mdPath, "workflow_call")
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
	return containsTrigger(onSection, "workflow_call")
}
