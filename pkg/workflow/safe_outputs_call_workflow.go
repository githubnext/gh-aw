package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var safeOutputsCallWorkflowLog = logger.New("workflow:safe_outputs_call_workflow")

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
	safeOutputsCallWorkflowLog.Printf("Generating call-workflow tool: workflow=%s, inputs=%d", workflowName, len(workflowInputs))

	// Normalize workflow name to use underscores for tool name
	toolName := stringutil.NormalizeSafeOutputIdentifier(workflowName)

	// Build the description
	description := fmt.Sprintf("Call the '%s' reusable workflow via workflow_call. This workflow must support workflow_call and be in .github/workflows/ directory in the same repository.", workflowName)

	// Build input schema properties from workflow_call inputs
	properties, required := buildInputSchema(workflowInputs, func(inputName string) string {
		return fmt.Sprintf("Input parameter '%s' for workflow %s", inputName, workflowName)
	})

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

	safeOutputsCallWorkflowLog.Printf("Generated call-workflow tool: name=%s, properties=%d, required=%d", toolName, len(properties), len(required))
	return tool
}
