package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/stringutil"
)

type workflowToolDefinitionOptions struct {
	workflowName      string
	workflowInputs    map[string]any
	descriptionFormat string
	metadataKey       string
}

func generateWorkflowToolDefinition(opts workflowToolDefinitionOptions) map[string]any {
	toolName := stringutil.NormalizeSafeOutputIdentifier(opts.workflowName)
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
	}

	return tool
}

func resolveWorkflowExtension(fileResult *findWorkflowFileResult) (string, bool) {
	if fileResult.lockExists {
		return ".lock.yml", true
	}
	if fileResult.ymlExists {
		return ".yml", true
	}
	if fileResult.mdExists {
		// .md-only: the workflow is a same-batch compilation target that will produce a .lock.yml
		return ".lock.yml", true
	}

	return "", false
}
