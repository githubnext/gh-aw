package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Call Workflow Handling
// ========================================
//
// This file contains functions for managing call-workflow safe output
// configurations: mapping workflow names to their relative file paths so the
// compiler can generate the correct `uses:` declarations.

// populateCallWorkflowFiles resolves the relative file path for each call-workflow
// listed in SafeOutputsConfig.CallWorkflow.Workflows. The resolved path is stored
// in WorkflowFiles for use by the compiler when generating conditional `uses:` jobs.
//
// Priority order: .lock.yml > .yml > .md (same-batch compilation target → .lock.yml)
func populateCallWorkflowFiles(data *WorkflowData, markdownPath string) {
	if data.SafeOutputs == nil || data.SafeOutputs.CallWorkflow == nil {
		return
	}

	if len(data.SafeOutputs.CallWorkflow.Workflows) == 0 {
		return
	}

	callWorkflowLog.Printf("Populating workflow files for %d call workflows", len(data.SafeOutputs.CallWorkflow.Workflows))

	// Initialize WorkflowFiles map if not already initialized
	if data.SafeOutputs.CallWorkflow.WorkflowFiles == nil {
		data.SafeOutputs.CallWorkflow.WorkflowFiles = make(map[string]string)
	}

	for _, workflowName := range data.SafeOutputs.CallWorkflow.Workflows {
		fileResult, err := findWorkflowFile(workflowName, markdownPath)
		if err != nil {
			callWorkflowLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
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
			callWorkflowLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
			continue
		}

		// Store the relative path for runtime/compile-time use (e.g. ./.github/workflows/worker.lock.yml)
		relativePath := fmt.Sprintf("./.github/workflows/%s%s", workflowName, extension)
		data.SafeOutputs.CallWorkflow.WorkflowFiles[workflowName] = relativePath
		callWorkflowLog.Printf("Mapped workflow %s to path %s", workflowName, relativePath)
	}
}

// generateCallWorkflowTool generates an MCP tool definition for a specific call-workflow target.
// The tool will be named after the workflow (normalized to underscores) and accept
// the workflow's defined workflow_call inputs as parameters.
// The agent calls this tool to select which worker to activate; the handler writes
// call_workflow_name and call_workflow_payload outputs for the conditional `uses:` jobs.
func generateCallWorkflowTool(workflowName string, workflowInputs map[string]any) map[string]any {
	// Normalize workflow name to use underscores for tool name
	toolName := stringutil.NormalizeSafeOutputIdentifier(workflowName)

	// Build the description
	description := fmt.Sprintf("Call the '%s' reusable workflow via workflow_call. This workflow must support workflow_call and be in .github/workflows/ directory in the same repository.", workflowName)

	// Build input schema properties
	properties := make(map[string]any)
	required := []string{}

	// Convert GitHub Actions workflow_call inputs to MCP tool schema
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

		// GitHub Actions workflow_call supports: string, number, boolean, choice, environment
		if typeStr, ok := inputDefMap["type"].(string); ok {
			switch typeStr {
			case "number":
				inputType = "number"
			case "boolean":
				inputType = "boolean"
			case "choice":
				inputType = "string"
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

		prop := map[string]any{
			"type":        inputType,
			"description": inputDescription,
		}

		if defaultVal, ok := inputDefMap["default"]; ok {
			prop["default"] = defaultVal
		}

		properties[inputName] = prop

		if inputRequired {
			required = append(required, inputName)
		}
	}

	// Build the complete tool definition
	tool := map[string]any{
		"name":                toolName,
		"description":         description,
		"_call_workflow_name": workflowName, // Internal metadata for handler routing
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
