package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Tools Generation
// ========================================
//
// This file handles tool JSON generation: it takes the full set of
// safe-output tool definitions (from safe-output-tools.json) and produces a
// filtered subset containing only those tools enabled by the workflow's
// SafeOutputsConfig. Dynamic tools (dispatch-workflow, custom jobs) are also
// generated here.
//
// generateToolsMetaJSON generates a small "meta" JSON (description suffixes,
// repo params, dynamic tools) to be written at compile time. At runtime,
// generate_safe_outputs_tools.cjs reads the source safe_outputs_tools.json
// from the actions folder, applies the meta overrides, and writes the final
// tools.json—avoiding inlining the entire file into the compiled workflow YAML.

// generateDynamicTools generates MCP tool definitions for dynamic tools:
// custom safe-jobs, dispatch_workflow targets, and call_workflow targets.
// These tools are not in safe_outputs_tools.json and must be generated from
// the workflow configuration at compile time.
func generateDynamicTools(data *WorkflowData, markdownPath string) ([]map[string]any, error) {
	var dynamicTools []map[string]any

	// Add custom job tools from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom job tools", len(data.SafeOutputs.Jobs))

		// Sort job names for deterministic output
		jobNames := make([]string, 0, len(data.SafeOutputs.Jobs))
		for jobName := range data.SafeOutputs.Jobs {
			jobNames = append(jobNames, jobName)
		}
		sort.Strings(jobNames)

		for _, jobName := range jobNames {
			jobConfig := data.SafeOutputs.Jobs[jobName]
			normalizedJobName := stringutil.NormalizeSafeOutputIdentifier(jobName)
			customTool := generateCustomJobToolDefinition(normalizedJobName, jobConfig)
			dynamicTools = append(dynamicTools, customTool)
		}
	}

	// Add custom script tools from SafeOutputs.Scripts
	if len(data.SafeOutputs.Scripts) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom script tools to dynamic tools", len(data.SafeOutputs.Scripts))

		scriptNames := make([]string, 0, len(data.SafeOutputs.Scripts))
		for scriptName := range data.SafeOutputs.Scripts {
			scriptNames = append(scriptNames, scriptName)
		}
		sort.Strings(scriptNames)

		for _, scriptName := range scriptNames {
			scriptConfig := data.SafeOutputs.Scripts[scriptName]
			normalizedScriptName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
			customTool := generateCustomScriptToolDefinition(normalizedScriptName, scriptConfig)
			dynamicTools = append(dynamicTools, customTool)
		}
	}

	// Add custom action tools from SafeOutputs.Actions
	// Each configured action is exposed as an MCP tool with schema derived from action.yml.
	// The compiler resolves action.yml at compile time; if resolution fails the tool is still
	// added with an empty inputSchema so the agent can still attempt to call it.
	if len(data.SafeOutputs.Actions) > 0 {
		safeOutputsConfigLog.Printf("Adding %d custom action tools to dynamic tools", len(data.SafeOutputs.Actions))

		actionNames := make([]string, 0, len(data.SafeOutputs.Actions))
		for actionName := range data.SafeOutputs.Actions {
			actionNames = append(actionNames, actionName)
		}
		sort.Strings(actionNames)

		for _, actionName := range actionNames {
			actionConfig := data.SafeOutputs.Actions[actionName]
			customTool := generateActionToolDefinition(actionName, actionConfig)
			dynamicTools = append(dynamicTools, customTool)
		}
	}

	// Add dynamic dispatch_workflow tools
	if data.SafeOutputs.DispatchWorkflow != nil && len(data.SafeOutputs.DispatchWorkflow.Workflows) > 0 {
		safeOutputsConfigLog.Printf("Adding %d dispatch_workflow tools", len(data.SafeOutputs.DispatchWorkflow.Workflows))

		if data.SafeOutputs.DispatchWorkflow.WorkflowFiles == nil {
			data.SafeOutputs.DispatchWorkflow.WorkflowFiles = make(map[string]string)
		}

		for _, workflowName := range data.SafeOutputs.DispatchWorkflow.Workflows {
			fileResult, err := findWorkflowFile(workflowName, markdownPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
				dynamicTools = append(dynamicTools, generateDispatchWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			var workflowPath string
			var extension string
			var useMD bool
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else if fileResult.mdExists {
				workflowPath = fileResult.mdPath
				extension = ".lock.yml"
				useMD = true
			} else {
				safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
				dynamicTools = append(dynamicTools, generateDispatchWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			data.SafeOutputs.DispatchWorkflow.WorkflowFiles[workflowName] = extension

			var workflowInputs map[string]any
			var inputsErr error
			if useMD {
				workflowInputs, inputsErr = extractMDWorkflowDispatchInputs(workflowPath)
			} else {
				workflowInputs, inputsErr = extractWorkflowDispatchInputs(workflowPath)
			}
			if inputsErr != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
				workflowInputs = make(map[string]any)
			}

			dynamicTools = append(dynamicTools, generateDispatchWorkflowTool(workflowName, workflowInputs))
		}
	}

	// Add dynamic dispatch_repository tools
	if data.SafeOutputs.DispatchRepository != nil && len(data.SafeOutputs.DispatchRepository.Tools) > 0 {
		safeOutputsConfigLog.Printf("Adding %d dispatch_repository tools to dynamic tools", len(data.SafeOutputs.DispatchRepository.Tools))

		// Sort tool keys for deterministic output
		toolKeys := make([]string, 0, len(data.SafeOutputs.DispatchRepository.Tools))
		for toolKey := range data.SafeOutputs.DispatchRepository.Tools {
			toolKeys = append(toolKeys, toolKey)
		}
		sort.Strings(toolKeys)

		for _, toolKey := range toolKeys {
			toolConfig := data.SafeOutputs.DispatchRepository.Tools[toolKey]
			dynamicTools = append(dynamicTools, generateDispatchRepositoryTool(toolKey, toolConfig))
		}
	}

	// Add dynamic call_workflow tools
	if data.SafeOutputs.CallWorkflow != nil && len(data.SafeOutputs.CallWorkflow.Workflows) > 0 {
		safeOutputsConfigLog.Printf("Adding %d call_workflow tools", len(data.SafeOutputs.CallWorkflow.Workflows))

		if data.SafeOutputs.CallWorkflow.WorkflowFiles == nil {
			data.SafeOutputs.CallWorkflow.WorkflowFiles = make(map[string]string)
		}

		for _, workflowName := range data.SafeOutputs.CallWorkflow.Workflows {
			fileResult, err := findWorkflowFile(workflowName, markdownPath)
			if err != nil {
				safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
				dynamicTools = append(dynamicTools, generateCallWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			var workflowPath string
			var extension string
			var useMD bool
			if fileResult.lockExists {
				workflowPath = fileResult.lockPath
				extension = ".lock.yml"
			} else if fileResult.ymlExists {
				workflowPath = fileResult.ymlPath
				extension = ".yml"
			} else if fileResult.mdExists {
				workflowPath = fileResult.mdPath
				extension = ".lock.yml"
				useMD = true
			} else {
				safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
				dynamicTools = append(dynamicTools, generateCallWorkflowTool(workflowName, make(map[string]any)))
				continue
			}

			relativePath := fmt.Sprintf("./.github/workflows/%s%s", workflowName, extension)
			data.SafeOutputs.CallWorkflow.WorkflowFiles[workflowName] = relativePath

			var workflowInputs map[string]any
			var inputsErr error
			if useMD {
				workflowInputs, inputsErr = extractMDWorkflowCallInputs(workflowPath)
			} else {
				workflowInputs, inputsErr = extractWorkflowCallInputs(workflowPath)
			}
			if inputsErr != nil {
				safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
				workflowInputs = make(map[string]any)
			}

			dynamicTools = append(dynamicTools, generateCallWorkflowTool(workflowName, workflowInputs))
		}
	}

	return dynamicTools, nil
}

// ToolsMeta is the structure written to tools_meta.json at compile time and read
// by generate_safe_outputs_tools.cjs at runtime. It avoids inlining the entire
// safe_outputs_tools.json into the compiled workflow YAML.
type ToolsMeta struct {
	// DescriptionSuffixes maps tool name → constraint text to append to the base description.
	// Example: " CONSTRAINTS: Maximum 5 issue(s) can be created."
	DescriptionSuffixes map[string]string `json:"description_suffixes"`
	// RepoParams maps tool name → "repo" inputSchema property definition, only present
	// when allowed-repos or a wildcard target-repo is configured for that tool.
	RepoParams map[string]map[string]any `json:"repo_params"`
	// DynamicTools contains tool definitions for custom safe-jobs, dispatch_workflow
	// targets, and call_workflow targets. These are workflow-specific and cannot be
	// derived from the static safe_outputs_tools.json at runtime.
	DynamicTools []map[string]any `json:"dynamic_tools"`
}

// generateToolsMetaJSON generates the content for tools_meta.json: a compact file
// that captures the workflow-specific customisations (description constraints,
// repo parameters, dynamic tools) without inlining the entire
// safe_outputs_tools.json into the compiled workflow YAML.
//
// At runtime, generate_safe_outputs_tools.cjs reads safe_outputs_tools.json from
// the actions folder, applies the meta overrides from tools_meta.json, and writes
// the final ${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json.
func generateToolsMetaJSON(data *WorkflowData, markdownPath string) (string, error) {
	if data.SafeOutputs == nil {
		empty := ToolsMeta{
			DescriptionSuffixes: map[string]string{},
			RepoParams:          map[string]map[string]any{},
			DynamicTools:        []map[string]any{},
		}
		result, err := json.Marshal(empty)
		if err != nil {
			return "", fmt.Errorf("failed to marshal empty tools meta: %w", err)
		}
		return string(result), nil
	}

	safeOutputsConfigLog.Print("Generating tools meta JSON for workflow")

	enabledTools := computeEnabledToolNames(data)

	// Compute description suffix for each enabled predefined tool.
	// enhanceToolDescription with an empty base returns just the constraint text
	// (e.g. " CONSTRAINTS: Maximum 5 issue(s).") so JavaScript can append it.
	descriptionSuffixes := make(map[string]string)
	for toolName := range enabledTools {
		suffix := enhanceToolDescription(toolName, "", data.SafeOutputs)
		if suffix != "" {
			descriptionSuffixes[toolName] = suffix
		}
	}

	// Compute repo parameter definition for each tool that needs it.
	repoParams := make(map[string]map[string]any)
	for toolName := range enabledTools {
		if param := computeRepoParamForTool(toolName, data.SafeOutputs); param != nil {
			repoParams[toolName] = param
		}
	}

	// Generate dynamic tool definitions (custom jobs + dispatch/call workflow tools).
	dynamicTools, err := generateDynamicTools(data, markdownPath)
	if err != nil {
		safeOutputsConfigLog.Printf("Error generating dynamic tools: %v", err)
		dynamicTools = []map[string]any{}
	}
	if dynamicTools == nil {
		dynamicTools = []map[string]any{}
	}

	meta := ToolsMeta{
		DescriptionSuffixes: descriptionSuffixes,
		RepoParams:          repoParams,
		DynamicTools:        dynamicTools,
	}

	result, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		safeOutputsConfigLog.Printf("Failed to marshal tools meta: %v", err)
		return "", fmt.Errorf("failed to marshal tools meta: %w", err)
	}

	safeOutputsConfigLog.Printf("Successfully generated tools meta JSON: %d description suffixes, %d repo params, %d dynamic tools",
		len(descriptionSuffixes), len(repoParams), len(dynamicTools))
	return string(result), nil
}
