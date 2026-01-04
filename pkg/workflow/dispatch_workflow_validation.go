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

		// Check if the workflow file exists - support .md, .lock.yml, or .yml files
		workflowFilePath := filepath.Join(workflowsDir, workflowName+".md")
		lockFilePath := filepath.Join(workflowsDir, workflowName+".lock.yml")
		ymlFilePath := filepath.Join(workflowsDir, workflowName+".yml")

		// Check if any workflow file exists
		mdExists := fileExists(workflowFilePath)
		lockExists := fileExists(lockFilePath)
		ymlExists := fileExists(ymlFilePath)

		if !mdExists && !lockExists && !ymlExists {
			return fmt.Errorf("dispatch-workflow: workflow '%s' not found (expected %s, %s, or %s)", workflowName, workflowFilePath, lockFilePath, ymlFilePath)
		}

		// Validate that the workflow supports workflow_dispatch
		// Priority: .lock.yml (compiled agentic workflow) > .yml (standard GitHub Actions) > .md (needs compilation)
		var workflowContent []byte
		var err error
		var workflowFile string

		if lockExists {
			workflowFile = lockFilePath
			workflowContent, err = os.ReadFile(lockFilePath)
			if err != nil {
				return fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", lockFilePath, err)
			}
		} else if ymlExists {
			workflowFile = ymlFilePath
			workflowContent, err = os.ReadFile(ymlFilePath)
			if err != nil {
				return fmt.Errorf("dispatch-workflow: failed to read workflow file %s: %w", ymlFilePath, err)
			}
		} else {
			// Only .md exists - needs to be compiled first
			return fmt.Errorf("dispatch-workflow: workflow '%s' must be compiled first (run 'gh aw compile %s')", workflowName, workflowFilePath)
		}

		// Parse the workflow YAML to check for workflow_dispatch trigger
		var workflow map[string]any
		if err := yaml.Unmarshal(workflowContent, &workflow); err != nil {
			return fmt.Errorf("dispatch-workflow: failed to parse workflow file %s: %w", workflowFile, err)
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

		dispatchWorkflowValidationLog.Printf("Workflow '%s' is valid for dispatch (found in %s)", workflowName, workflowFile)
	}

	dispatchWorkflowValidationLog.Printf("All %d workflows validated successfully", len(config.Workflows))
	return nil
}

// extractWorkflowDispatchInputs parses a workflow file and extracts the workflow_dispatch inputs schema
// Returns a map of input definitions that can be used to generate MCP tool schemas
func extractWorkflowDispatchInputs(workflowPath string) (map[string]any, error) {
	workflowContent, err := os.ReadFile(workflowPath)
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

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
