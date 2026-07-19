package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/sliceutil"
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
// generateToolsMetaJSON generates the content for tools_meta.json: a compact file
// that captures the workflow-specific customisations without inlining all tools.
func generateToolsMetaJSON(data *WorkflowData, markdownPath string) (string, error) {
	if data.SafeOutputs == nil {
		return marshalEmptyToolsMeta()
	}
	safeOutputsConfigLog.Print("Generating tools meta JSON for workflow")
	enabledTools := computeEnabledToolNames(data)
	descriptionSuffixes := computeDescriptionSuffixes(enabledTools, data.SafeOutputs)
	repoParams := computeRepoParams(enabledTools, data.SafeOutputs)
	dynamicTools := safeGenerateDynamicTools(data, markdownPath)
	meta := ToolsMeta{
		DescriptionSuffixes:    descriptionSuffixes,
		RepoParams:             repoParams,
		DynamicTools:           dynamicTools,
		RequiredFieldRemovals:  computeRequiredFieldRemovals(data.SafeOutputs),
		RequiredFieldAdditions: computeRequiredFieldAdditions(data.SafeOutputs),
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

func marshalEmptyToolsMeta() (string, error) {
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

func computeDescriptionSuffixes(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) map[string]string {
	descriptionSuffixes := make(map[string]string)
	for toolName := range enabledTools {
		suffix := enhanceToolDescription(toolName, "", safeOutputs)
		if suffix != "" {
			descriptionSuffixes[toolName] = suffix
		}
	}
	return descriptionSuffixes
}

func computeRepoParams(enabledTools map[string]struct{}, safeOutputs *SafeOutputsConfig) map[string]map[string]any {
	repoParams := make(map[string]map[string]any)
	for toolName := range enabledTools {
		if param := computeRepoParamForTool(toolName, safeOutputs); param != nil {
			repoParams[toolName] = param
		}
	}
	return repoParams
}

func safeGenerateDynamicTools(data *WorkflowData, markdownPath string) []map[string]any {
	dynamicTools, err := generateDynamicTools(data, markdownPath)
	if err != nil {
		safeOutputsConfigLog.Printf("Error generating dynamic tools: %v", err)
		dynamicTools = []map[string]any{}
	}
	if dynamicTools == nil {
		dynamicTools = []map[string]any{}
	}
	return dynamicTools
}

// generateDynamicTools generates MCP tool definitions for dynamic tools:
// custom safe-jobs, dispatch_workflow targets, and call_workflow targets.
func generateDynamicTools(data *WorkflowData, markdownPath string) ([]map[string]any, error) {
	var dynamicTools []map[string]any
	dynamicTools = append(dynamicTools, generateCustomJobTools(data)...)
	dynamicTools = append(dynamicTools, generateCustomScriptTools(data)...)
	dynamicTools = append(dynamicTools, generateCustomActionTools(data)...)
	dynamicTools = append(dynamicTools, generateDispatchWorkflowTools(data, markdownPath)...)
	dynamicTools = append(dynamicTools, generateDispatchRepositoryTools(data)...)
	dynamicTools = append(dynamicTools, generateCallWorkflowTools(data, markdownPath)...)
	return dynamicTools, nil
}

func generateCustomJobTools(data *WorkflowData) []map[string]any {
	if len(data.SafeOutputs.Jobs) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Adding %d custom job tools", len(data.SafeOutputs.Jobs))
	tools := make([]map[string]any, 0, len(data.SafeOutputs.Jobs))
	for _, jobName := range sliceutil.SortedKeys(data.SafeOutputs.Jobs) {
		jobConfig := data.SafeOutputs.Jobs[jobName]
		normalizedJobName := stringutil.NormalizeSafeOutputIdentifier(jobName)
		tools = append(tools, generateCustomJobToolDefinition(normalizedJobName, jobConfig))
	}
	return tools
}

func generateCustomScriptTools(data *WorkflowData) []map[string]any {
	if len(data.SafeOutputs.Scripts) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Adding %d custom script tools to dynamic tools", len(data.SafeOutputs.Scripts))
	tools := make([]map[string]any, 0, len(data.SafeOutputs.Scripts))
	for _, scriptName := range sliceutil.SortedKeys(data.SafeOutputs.Scripts) {
		scriptConfig := data.SafeOutputs.Scripts[scriptName]
		normalizedScriptName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
		tools = append(tools, generateCustomScriptToolDefinition(normalizedScriptName, scriptConfig))
	}
	return tools
}

func generateCustomActionTools(data *WorkflowData) []map[string]any {
	if len(data.SafeOutputs.Actions) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Adding %d custom action tools to dynamic tools", len(data.SafeOutputs.Actions))
	tools := make([]map[string]any, 0, len(data.SafeOutputs.Actions))
	for _, actionName := range sliceutil.SortedKeys(data.SafeOutputs.Actions) {
		tools = append(tools, generateActionToolDefinition(actionName, data.SafeOutputs.Actions[actionName]))
	}
	return tools
}

func generateDispatchWorkflowTools(data *WorkflowData, markdownPath string) []map[string]any {
	config := data.SafeOutputs.DispatchWorkflow
	if config == nil || len(config.Workflows) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Adding %d dispatch_workflow tools", len(config.Workflows))
	if config.WorkflowFiles == nil {
		config.WorkflowFiles = make(map[string]string)
	}
	tools := make([]map[string]any, 0, len(config.Workflows))
	for _, workflowName := range config.Workflows {
		inputs, extension := loadDynamicWorkflowInputs(workflowName, markdownPath, extractWorkflowDispatchInputs, extractMDWorkflowDispatchInputs)
		if extension != "" {
			config.WorkflowFiles[workflowName] = extension
		}
		tools = append(tools, generateDispatchWorkflowTool(workflowName, inputs))
	}
	return tools
}

func generateDispatchRepositoryTools(data *WorkflowData) []map[string]any {
	config := data.SafeOutputs.DispatchRepository
	if config == nil || len(config.Tools) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Adding %d dispatch_repository tools to dynamic tools", len(config.Tools))
	tools := make([]map[string]any, 0, len(config.Tools))
	for _, toolKey := range sliceutil.SortedKeys(config.Tools) {
		tools = append(tools, generateDispatchRepositoryTool(toolKey, config.Tools[toolKey]))
	}
	return tools
}

func generateCallWorkflowTools(data *WorkflowData, markdownPath string) []map[string]any {
	config := data.SafeOutputs.CallWorkflow
	if config == nil || len(config.Workflows) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Adding %d call_workflow tools", len(config.Workflows))
	if config.WorkflowFiles == nil {
		config.WorkflowFiles = make(map[string]string)
	}
	tools := make([]map[string]any, 0, len(config.Workflows))
	for _, workflowName := range config.Workflows {
		inputs, extension := loadDynamicWorkflowInputs(workflowName, markdownPath, extractWorkflowCallInputs, extractMDWorkflowCallInputs)
		if extension != "" {
			config.WorkflowFiles[workflowName] = fmt.Sprintf("./.github/workflows/%s%s", workflowName, extension)
		}
		tools = append(tools, generateCallWorkflowTool(workflowName, inputs))
	}
	return tools
}

type workflowInputExtractor func(string) (map[string]any, error)

func loadDynamicWorkflowInputs(workflowName, markdownPath string, ymlExtractor, mdExtractor workflowInputExtractor) (map[string]any, string) {
	workflowPath, extension, useMD, ok := resolveDynamicWorkflowFile(workflowName, markdownPath)
	if !ok {
		return make(map[string]any), ""
	}
	extractor := ymlExtractor
	if useMD {
		extractor = mdExtractor
	}
	workflowInputs, inputsErr := extractor(workflowPath)
	if inputsErr != nil {
		safeOutputsConfigLog.Printf("Warning: failed to extract inputs for workflow %s from %s: %v", workflowName, workflowPath, inputsErr)
		workflowInputs = make(map[string]any)
	}
	return workflowInputs, extension
}

func resolveDynamicWorkflowFile(workflowName, markdownPath string) (string, string, bool, bool) {
	fileResult, err := findWorkflowFile(workflowName, markdownPath)
	if err != nil {
		safeOutputsConfigLog.Printf("Warning: error finding workflow %s: %v", workflowName, err)
		return "", "", false, false
	}
	if fileResult.lockExists {
		return fileResult.lockPath, ".lock.yml", false, true
	}
	if fileResult.ymlExists {
		return fileResult.ymlPath, ".yml", false, true
	}
	if fileResult.mdExists {
		return fileResult.mdPath, ".lock.yml", true, true
	}
	safeOutputsConfigLog.Printf("Warning: no workflow file found for %s (checked .lock.yml, .yml, .md)", workflowName)
	return "", "", false, false
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
	// RequiredFieldRemovals maps tool name → list of field names to remove from the
	// inputSchema.required array. Used when a field that is required in the static
	// safe_outputs_tools.json should be optional for this specific workflow (e.g. when
	// allow-body: false is configured for close_discussion or close_issue).
	RequiredFieldRemovals map[string][]string `json:"required_field_removals,omitempty"`
	// RequiredFieldAdditions maps tool name → list of field names to add to the
	// inputSchema.required array. Used when a field that is optional in the static
	// safe_outputs_tools.json should be required for this specific workflow.
	RequiredFieldAdditions map[string][]string `json:"required_field_additions,omitempty"`
}

// computeRequiredFieldRemovals returns a map of tool name → required fields to remove
// based on the safe-outputs configuration. Currently handles allow-body: false for
// close_discussion and close_issue.
func computeRequiredFieldRemovals(safeOutputs *SafeOutputsConfig) map[string][]string {
	removals := make(map[string][]string)
	if safeOutputs == nil {
		return removals
	}
	if safeOutputs.CloseDiscussions != nil && safeOutputs.CloseDiscussions.AllowBody != nil && !*safeOutputs.CloseDiscussions.AllowBody {
		removals["close_discussion"] = []string{"body"}
	}
	if safeOutputs.CloseIssues != nil && safeOutputs.CloseIssues.AllowBody != nil && !*safeOutputs.CloseIssues.AllowBody {
		removals["close_issue"] = []string{"body"}
	}
	return removals
}

// computeRequiredFieldAdditions returns a map of tool name → required fields to add
// based on the safe-outputs configuration.
func computeRequiredFieldAdditions(safeOutputs *SafeOutputsConfig) map[string][]string {
	additions := make(map[string][]string)
	if safeOutputs == nil {
		return additions
	}
	if safeOutputs.CreateIssues != nil && safeOutputs.CreateIssues.RequireTemporaryID {
		additions["create_issue"] = []string{"temporary_id"}
	}
	if safeOutputs.CreatePullRequests != nil && safeOutputs.CreatePullRequests.RequireTemporaryID {
		additions["create_pull_request"] = []string{"temporary_id"}
	}
	issueIntentRequiredFields := []string{"rationale", "confidence"}
	if safeOutputs.SetIssueType != nil && issueIntentRequired(safeOutputs.SetIssueType.IssueIntent) {
		additions["set_issue_type"] = issueIntentRequiredFields
	}
	if safeOutputs.SetIssueField != nil && issueIntentRequired(safeOutputs.SetIssueField.IssueIntent) {
		additions["set_issue_field"] = issueIntentRequiredFields
	}
	if safeOutputs.CloseIssues != nil && issueIntentRequired(safeOutputs.CloseIssues.IssueIntent) {
		additions["close_issue"] = issueIntentRequiredFields
	}
	if safeOutputs.AssignToUser != nil && issueIntentRequired(safeOutputs.AssignToUser.IssueIntent) {
		additions["assign_to_user"] = issueIntentRequiredFields
	}
	if safeOutputs.AssignToAgent != nil && issueIntentRequired(safeOutputs.AssignToAgent.IssueIntent) {
		additions["assign_to_agent"] = issueIntentRequiredFields
	}
	return additions
}

func issueIntentRequired(issueIntent *bool) bool {
	return issueIntent != nil && *issueIntent
}
