package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var safeOutputsWorkflowHelpersLog = logger.New("workflow:safe_outputs_workflow_helpers")

type workflowToolDefinitionOptions struct {
	workflowName      string
	workflowInputs    map[string]any
	descriptionFormat string
	metadataKey       string
}

func generateWorkflowToolDefinition(opts workflowToolDefinitionOptions) map[string]any {
	toolName := stringutil.NormalizeSafeOutputIdentifier(opts.workflowName)
	safeOutputsWorkflowHelpersLog.Printf("Generating workflow tool definition: workflow=%s, tool=%s, inputs=%d", opts.workflowName, toolName, len(opts.workflowInputs))
	description := fmt.Sprintf(opts.descriptionFormat, opts.workflowName)
	properties, required := buildInputSchema(opts.workflowInputs, func(inputName string) string {
		return fmt.Sprintf("Input parameter '%s' for workflow %s", inputName, opts.workflowName)
	})

	tool := map[string]any{
		"name":           toolName,
		"description":    description,
		opts.metadataKey: opts.workflowName,
		"inputSchema": map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		},
	}

	if len(required) > 0 {
		sort.Strings(required)
		tool["inputSchema"].(map[string]any)["required"] = required
		safeOutputsWorkflowHelpersLog.Printf("Workflow tool %s has %d required inputs", toolName, len(required))
	}

	return tool
}

func resolveWorkflowExtension(fileResult *findWorkflowFileResult) (string, bool) {
	if fileResult.lockExists {
		safeOutputsWorkflowHelpersLog.Print("Resolved workflow extension: .lock.yml (lock file exists)")
		return ".lock.yml", true
	}
	if fileResult.ymlExists {
		safeOutputsWorkflowHelpersLog.Print("Resolved workflow extension: .yml (yml file exists)")
		return ".yml", true
	}
	if fileResult.mdExists {
		// .md-only: the workflow is a same-batch compilation target that will produce a .lock.yml
		safeOutputsWorkflowHelpersLog.Print("Resolved workflow extension: .lock.yml (md-only, same-batch target)")
		return ".lock.yml", true
	}

	safeOutputsWorkflowHelpersLog.Print("Failed to resolve workflow extension: no candidate files exist")
	return "", false
}
