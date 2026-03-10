package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Dispatch Workflow Handling
// ========================================
//
// This file contains functions for managing dispatch-workflow safe output
// configurations: mapping workflow names to their file extensions so the
// runtime handler knows which file to use when dispatching a workflow.

// populateDispatchWorkflowFiles resolves the file extension for each dispatch
// workflow listed in SafeOutputsConfig.DispatchWorkflow.Workflows. The resolved
// extension is stored in WorkflowFiles for later use by the runtime handler.
//
// Priority order: .lock.yml > .yml > .md (same-batch compilation target)
func populateDispatchWorkflowFiles(data *WorkflowData, markdownPath string) {
	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return
	}

	if len(data.SafeOutputs.DispatchWorkflow.Workflows) == 0 {
		return
	}

	safeOutputsConfigLog.Printf("Populating workflow files for %d dispatch workflows", len(data.SafeOutputs.DispatchWorkflow.Workflows))

	// Initialize WorkflowFiles map if not already initialized
	if data.SafeOutputs.DispatchWorkflow.WorkflowFiles == nil {
		data.SafeOutputs.DispatchWorkflow.WorkflowFiles = make(map[string]string)
	}

	for _, workflowName := range data.SafeOutputs.DispatchWorkflow.Workflows {
		// Find the workflow file
		fileResult, err := findWorkflowFile(workflowName, markdownPath)
		if err != nil {
			safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
			continue
		}

		// Determine which file to use - priority: .lock.yml > .yml > .md (batch target)
		var extension string
		if fileResult.lockExists {
			extension = ".lock.yml"
		} else if fileResult.ymlExists {
			extension = ".yml"
		} else if fileResult.mdExists {
			// .md-only: the workflow is a same-batch compilation target that will produce a .lock.yml
			extension = ".lock.yml"
		} else {
			safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
			continue
		}

		// Store the file extension for runtime use
		data.SafeOutputs.DispatchWorkflow.WorkflowFiles[workflowName] = extension
		safeOutputsConfigLog.Printf("Mapped workflow %s to extension %s", workflowName, extension)
	}
}

// generateDispatchWorkflowTool generates an MCP tool definition for a specific workflow.
// The tool will be named after the workflow (normalized to underscores) and accept
// the workflow's defined workflow_dispatch inputs as parameters.
func generateDispatchWorkflowTool(workflowName string, workflowInputs map[string]any) map[string]any {
	// Normalize workflow name to use underscores for tool name
	toolName := stringutil.NormalizeSafeOutputIdentifier(workflowName)

	// Build the description
	description := fmt.Sprintf("Dispatch the '%s' workflow with workflow_dispatch trigger. This workflow must support workflow_dispatch and be in .github/workflows/ directory in the same repository.", workflowName)

	// Build input schema properties
	properties := make(map[string]any)
	required := []string{} // No required fields by default

	// Convert GitHub Actions workflow_dispatch inputs to MCP tool schema
	for inputName, inputDef := range workflowInputs {
		inputDefMap, ok := inputDef.(map[string]any)
		if !ok {
			continue
		}

		// Extract input properties
		inputType := "string" // Default type
		inputDescription := fmt.Sprintf("Input parameter '%s' for workflow %s", inputName, workflowName)
		inputRequired := false

		if desc, ok := inputDefMap["description"].(string); ok && desc != "" {
			inputDescription = desc
		}

		if req, ok := inputDefMap["required"].(bool); ok {
			inputRequired = req
		}

		// GitHub Actions workflow_dispatch supports: string, number, boolean, choice, environment
		// Map these to JSON schema types
		if typeStr, ok := inputDefMap["type"].(string); ok {
			switch typeStr {
			case "number":
				inputType = "number"
			case "boolean":
				inputType = "boolean"
			case "choice":
				inputType = "string"
				// Add enum if options are provided
				if options, ok := inputDefMap["options"].([]any); ok && len(options) > 0 {
					properties[inputName] = map[string]any{
						"type":        inputType,
						"description": inputDescription,
						"enum":        options,
					}
					if inputRequired {
						required = append(required, inputName)
					}
					continue
				}
			case "environment":
				inputType = "string"
			}
		}

		properties[inputName] = map[string]any{
			"type":        inputType,
			"description": inputDescription,
		}

		// Add default value if provided
		if defaultVal, ok := inputDefMap["default"]; ok {
			properties[inputName].(map[string]any)["default"] = defaultVal
		}

		if inputRequired {
			required = append(required, inputName)
		}
	}

	// Add internal workflow_name parameter (hidden from description but used internally)
	// This will be injected by the safe output handler

	// Build the complete tool definition
	tool := map[string]any{
		"name":           toolName,
		"description":    description,
		"_workflow_name": workflowName, // Internal metadata for handler routing
		"inputSchema": map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		},
	}

	if len(required) > 0 {
		sort.Strings(required)
		tool["inputSchema"].(map[string]any)["required"] = required
	}

	return tool
}
