package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var safeOutputsDispatchWorkflowLog = logger.New("workflow:safe_outputs_dispatch")

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
// It also detects which workflows declare aw_context in their workflow_dispatch.inputs
// and stores those names in AwContextWorkflows, so the runtime handler only injects
// aw_context metadata for workflows that explicitly support it.
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

		// Check if the target workflow declares aw_context in its workflow_dispatch.inputs.
		// We check the lock file first (compiled YAML), falling back to the markdown frontmatter.
		if workflowHasAwContextInput(fileResult, workflowName) {
			data.SafeOutputs.DispatchWorkflow.AwContextWorkflows = append(
				data.SafeOutputs.DispatchWorkflow.AwContextWorkflows, workflowName,
			)
			safeOutputsConfigLog.Printf("Workflow %s declares aw_context input", workflowName)
		}
	}
}

// workflowHasAwContextInput reports whether the workflow identified by fileResult
// has aw_context declared in its workflow_dispatch.inputs.
func workflowHasAwContextInput(fileResult *findWorkflowFileResult, workflowName string) bool {
	var inputs map[string]any
	var err error

	if fileResult.lockExists {
		inputs, err = extractWorkflowDispatchInputs(fileResult.lockPath)
	} else if fileResult.ymlExists {
		inputs, err = extractWorkflowDispatchInputs(fileResult.ymlPath)
	} else if fileResult.mdExists {
		inputs, err = extractMDWorkflowDispatchInputs(fileResult.mdPath)
	} else {
		return false
	}
	if err != nil {
		safeOutputsConfigLog.Printf("Warning: error extracting inputs for %s: %v", workflowName, err)
		return false
	}
	_, hasAwContext := inputs["aw_context"]
	return hasAwContext
}

// generateDispatchWorkflowTool generates an MCP tool definition for a specific workflow.
// The tool will be named after the workflow (normalized to underscores) and accept
// the workflow's defined workflow_dispatch inputs as parameters.
func generateDispatchWorkflowTool(workflowName string, workflowInputs map[string]any) map[string]any {
	safeOutputsDispatchWorkflowLog.Printf("Generating dispatch-workflow tool: workflow=%s, inputs=%d", workflowName, len(workflowInputs))

	// Normalize workflow name to use underscores for tool name
	toolName := stringutil.NormalizeSafeOutputIdentifier(workflowName)

	// Build the description
	description := fmt.Sprintf("Dispatch the '%s' workflow with workflow_dispatch trigger. This workflow must support workflow_dispatch and be in .github/workflows/ directory in the same repository.", workflowName)

	// Build input schema properties from workflow_dispatch inputs
	properties, required := buildInputSchema(workflowInputs, func(inputName string) string {
		return fmt.Sprintf("Input parameter '%s' for workflow %s", inputName, workflowName)
	})

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

	safeOutputsDispatchWorkflowLog.Printf("Generated dispatch-workflow tool: name=%s, properties=%d, required=%d", toolName, len(properties), len(required))
	return tool
}
