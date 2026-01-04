package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var dispatchWorkflowValidationLog = logger.New("workflow:dispatch_workflow_validation")

// validateDispatchWorkflow validates that the dispatch-workflow configuration is correct
func (c *Compiler) validateDispatchWorkflow(data *WorkflowData, workflowPath string) error {
	dispatchWorkflowValidationLog.Print("Starting dispatch-workflow validation")

	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		dispatchWorkflowValidationLog.Print("No dispatch-workflow configuration found")
		return nil
	}

	config := data.SafeOutputs.DispatchWorkflow

	if len(config.Workflows) == 0 {
		return fmt.Errorf("dispatch-workflow: must specify at least one workflow in the list")
	}

	// Get the current workflow name for self-reference check
	currentWorkflowName := getCurrentWorkflowName(workflowPath)
	dispatchWorkflowValidationLog.Printf("Current workflow name: %s", currentWorkflowName)

	// Get the workflows directory
	workflowsDir := filepath.Dir(workflowPath)

	for _, workflowName := range config.Workflows {
		dispatchWorkflowValidationLog.Printf("Validating workflow: %s", workflowName)

		// Check for self-reference
		if workflowName == currentWorkflowName {
			return fmt.Errorf("dispatch-workflow: self-reference not allowed (workflow '%s' cannot dispatch itself)", workflowName)
		}

		// Check if the workflow file exists
		workflowFilePath := filepath.Join(workflowsDir, workflowName+".md")
		lockFilePath := filepath.Join(workflowsDir, workflowName+".lock.yml")

		// Check if either .md or .lock.yml exists
		mdExists := fileExists(workflowFilePath)
		lockExists := fileExists(lockFilePath)

		if !mdExists && !lockExists {
			return fmt.Errorf("dispatch-workflow: workflow '%s' not found (expected %s or %s)", workflowName, workflowFilePath, lockFilePath)
		}

		// Validate that the workflow supports workflow_dispatch
		// We check the .lock.yml file if it exists, otherwise parse the .md file
		var workflowContent []byte
		var err error

		if lockExists {
			workflowContent, err = os.ReadFile(lockFilePath)
			if err != nil {
				return fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", lockFilePath, err)
			}
		} else {
			// For .md files, we need to compile them first or just check the frontmatter
			// For simplicity, we'll just check if the .lock.yml exists
			return fmt.Errorf("dispatch-workflow: workflow '%s' must be compiled first (run 'gh aw compile %s')", workflowName, workflowFilePath)
		}

		// Parse the workflow YAML to check for workflow_dispatch trigger
		var workflow map[string]any
		if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
			return fmt.Errorf("dispatch-workflow: failed to parse workflow file %s: %w", lockFilePath, err)
		}

		// Check if the workflow has an "on" section
		onSection, hasOn := workflow["on"]
		if !hasOn {
			return fmt.Errorf("dispatch-workflow: workflow '%s' does not have an 'on' trigger section", workflowName)
		}

		// Check if workflow_dispatch is in the "on" section
		hasWorkflowDispatch := false
		switch on := onSection.(type) {
		case string:
			// Simple trigger like "on: push"
			if on == "workflow_dispatch" {
				hasWorkflowDispatch = true
			}
		case []any:
			// Array of triggers like "on: [push, workflow_dispatch]"
			for _, trigger := range on {
				if triggerStr, ok := trigger.(string); ok && triggerStr == "workflow_dispatch" {
					hasWorkflowDispatch = true
					break
				}
			}
		case map[string]any:
			// Map of triggers like "on: { push: {}, workflow_dispatch: {} }"
			_, hasWorkflowDispatch = on["workflow_dispatch"]
		}

		if !hasWorkflowDispatch {
			return fmt.Errorf("dispatch-workflow: workflow '%s' does not support workflow_dispatch trigger (must include 'workflow_dispatch' in the 'on' section)", workflowName)
		}

		dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch", workflowName)
	}

	dispatchWorkflowValidationLog.Printf("All %d workflows validated successfully", len(config.Workflows))
	return nil
}

// getCurrentWorkflowName extracts the workflow name from the file path
func getCurrentWorkflowName(workflowPath string) string {
	filename := filepath.Base(workflowPath)
	// Remove .md or .lock.yml extension
	filename = strings.TrimSuffix(filename, ".md")
	filename = strings.TrimSuffix(filename, ".lock.yml")
	return filename
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
