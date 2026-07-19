package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Configuration Generation
// ========================================
//
// This file generates the GH_AW_SAFE_OUTPUTS_CONFIG_PATH (config.json) consumed
// by the safe-outputs MCP server at startup and by the output ingestion step.
//
// Standard handler configuration is derived from the handlerRegistry defined in
// compiler_safe_outputs_config.go (the single source of truth for handler keys and
// field contracts). Non-handler global configuration (mentions, max_bot_mentions,
// safe_jobs, safe_scripts, push_repo_memory) is generated here because it is
// specific to config.json and not part of the handler registry.

// generateSafeOutputsConfig generates the JSON configuration for the safe-outputs
// MCP server. Standard handler configs are sourced from handlerRegistry to ensure
// they stay in sync with GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG.
func generateSafeOutputsConfig(data *WorkflowData) (string, error) {
	if data.SafeOutputs == nil {
		safeOutputsConfigLog.Print("No safe outputs configuration found, returning empty config")
		return "", nil
	}
	safeOutputsConfigLog.Print("Generating safe outputs configuration for workflow")

	safeOutputsConfig := make(map[string]any)
	engineManifestFiles, engineManifestPathPrefixes := getEngineAgentFileInfoFromWorkflowData(data)

	addStandardSafeOutputHandlerConfigs(safeOutputsConfig, data, engineManifestFiles, engineManifestPathPrefixes)
	addSafeOutputJobConfigs(safeOutputsConfig, data.SafeOutputs.Jobs)
	addSafeOutputScriptConfigs(safeOutputsConfig, data.SafeOutputs.Scripts)
	if err := addSafeOutputActionConfigs(safeOutputsConfig, data.SafeOutputs.Actions); err != nil {
		return "", err
	}

	if data.SafeOutputs.Mentions != nil {
		mentionsConfig := buildMentionsHandlerConfig(data.SafeOutputs.Mentions)
		if len(mentionsConfig) > 0 {
			safeOutputsConfig["mentions"] = mentionsConfig
		}
	}

	addMaxBotMentionsConfig(safeOutputsConfig, data.SafeOutputs)
	addPushRepoMemoryConfig(safeOutputsConfig, data)

	if len(safeOutputsConfig) == 0 {
		return "", nil
	}
	configJSON, err := json.Marshal(safeOutputsConfig)
	if err != nil {
		return "", fmt.Errorf("marshaling safe-outputs config: %w", err)
	}
	safeOutputsConfigLog.Printf("Safe outputs config generation complete: %d tool types configured", len(safeOutputsConfig))
	return string(configJSON), nil
}

func addStandardSafeOutputHandlerConfigs(safeOutputsConfig map[string]any, data *WorkflowData, engineManifestFiles []string, engineManifestPathPrefixes []string) {
	for handlerName, builder := range handlerRegistry {
		if handlerCfg := builder(data.SafeOutputs); handlerCfg != nil {
			injectCurrentCheckoutPatchWorkspacePath(handlerName, handlerCfg, data)
			injectCheckoutMapping(handlerName, handlerCfg, data)
			applyProtectedFileConfig(handlerCfg, engineManifestFiles, engineManifestPathPrefixes)
			safeOutputsConfig[handlerName] = handlerCfg
		}
	}
}

func applyProtectedFileConfig(handlerCfg map[string]any, engineManifestFiles []string, engineManifestPathPrefixes []string) {
	excludeFiles := ParseStringArrayFromConfig(handlerCfg, "_protected_files_exclude", nil)
	delete(handlerCfg, "_protected_files_exclude")
	if _, hasProtectedFiles := handlerCfg["protected_files"]; !hasProtectedFiles {
		return
	}
	fullManifestFiles := getAllManifestFiles(engineManifestFiles...)
	fullPathPrefixes := getProtectedPathPrefixes(engineManifestPathPrefixes...)
	handlerCfg["protected_files"] = sliceutil.Exclude(fullManifestFiles, excludeFiles...)
	filteredPrefixes := sliceutil.Exclude(fullPathPrefixes, excludeFiles...)
	if len(filteredPrefixes) > 0 {
		handlerCfg["protected_path_prefixes"] = filteredPrefixes
	} else {
		delete(handlerCfg, "protected_path_prefixes")
	}
	if dotFolderExcludes := getDotFolderExcludes(excludeFiles); len(dotFolderExcludes) > 0 {
		handlerCfg["protected_dot_folder_excludes"] = dotFolderExcludes
	}
}

func addSafeOutputJobConfigs(safeOutputsConfig map[string]any, jobs map[string]*SafeJobConfig) {
	if len(jobs) == 0 {
		return
	}
	safeOutputsConfigLog.Printf("Processing %d safe job configurations", len(jobs))
	for jobName, jobConfig := range jobs {
		safeOutputsConfigLog.Printf("Generating config for safe job: %s", jobName)
		safeJobConfig := map[string]any{}
		if jobConfig.Description != "" {
			safeJobConfig["description"] = jobConfig.Description
		}
		if jobConfig.Output != "" {
			safeJobConfig["output"] = jobConfig.Output
		}
		if jobConfig.Max > 0 {
			safeJobConfig["max"] = jobConfig.Max
		}
		if len(jobConfig.Inputs) > 0 {
			safeJobConfig["inputs"] = buildSafeOutputInputsConfig(jobConfig.Inputs)
		}
		safeOutputsConfig[jobName] = safeJobConfig
	}
}

func addSafeOutputScriptConfigs(safeOutputsConfig map[string]any, scripts map[string]*SafeScriptConfig) {
	if len(scripts) == 0 {
		return
	}
	safeOutputsConfigLog.Printf("Processing %d safe script configurations", len(scripts))
	for scriptName, scriptConfig := range scripts {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
		safeOutputsConfigLog.Printf("Generating config for safe script: %s (normalized: %s)", scriptName, normalizedName)
		safeScriptConfigMap := map[string]any{}
		if scriptConfig.Description != "" {
			safeScriptConfigMap["description"] = scriptConfig.Description
		}
		if len(scriptConfig.Inputs) > 0 {
			safeScriptConfigMap["inputs"] = buildSafeOutputInputsConfig(scriptConfig.Inputs)
		}
		safeOutputsConfig[normalizedName] = safeScriptConfigMap
	}
}

func buildSafeOutputInputsConfig(inputs map[string]*InputDefinition) map[string]any {
	inputsConfig := make(map[string]any)
	for inputName, inputDef := range inputs {
		inputConfig := map[string]any{
			"type":        inputDef.Type,
			"description": inputDef.Description,
			"required":    inputDef.Required,
		}
		if inputDef.Default != "" {
			inputConfig["default"] = inputDef.Default
		}
		if len(inputDef.Options) > 0 {
			inputConfig["options"] = inputDef.Options
		}
		inputsConfig[inputName] = inputConfig
	}
	return inputsConfig
}

func addSafeOutputActionConfigs(safeOutputsConfig map[string]any, actions map[string]*SafeOutputActionConfig) error {
	if len(actions) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Processing %d safe action configurations", len(actions))
	for actionName := range actions {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(actionName)
		if _, exists := safeOutputsConfig[normalizedName]; exists {
			return fmt.Errorf("safe-outputs action %q has a normalized name %q that conflicts with an existing safe outputs config entry; rename the action to avoid the conflict", actionName, normalizedName)
		}
		safeOutputsConfigLog.Printf("Adding safe action to config: %s (normalized: %s)", actionName, normalizedName)
		safeOutputsConfig[normalizedName] = true
	}
	return nil
}

func addMaxBotMentionsConfig(safeOutputsConfig map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs.MaxBotMentions == nil {
		return
	}
	v := *safeOutputs.MaxBotMentions
	if n := templatableIntValue(safeOutputs.MaxBotMentions); n > 0 {
		safeOutputsConfig["max_bot_mentions"] = n
	} else if strings.HasPrefix(v, "${{") {
		safeOutputsConfig["max_bot_mentions"] = v
	}
}

func addPushRepoMemoryConfig(safeOutputsConfig map[string]any, data *WorkflowData) {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return
	}
	var memories []map[string]any
	for _, memory := range data.RepoMemoryConfig.Memories {
		memories = append(memories, map[string]any{
			"id":             memory.ID,
			"dir":            constants.TmpRepoMemoryDir + memory.ID,
			"max_file_size":  memory.MaxFileSize,
			"max_patch_size": memory.MaxPatchSize,
			"max_file_count": memory.MaxFileCount,
		})
	}
	safeOutputsConfig["push_repo_memory"] = map[string]any{"memories": memories}
	safeOutputsConfigLog.Printf("Added push_repo_memory config with %d memory entries", len(data.RepoMemoryConfig.Memories))
}

func getEngineAgentFileInfoFromWorkflowData(data *WorkflowData) (manifestFiles []string, pathPrefixes []string) {
	if data == nil || data.EngineConfig == nil {
		return nil, nil
	}

	engineRegistry := GetGlobalEngineRegistry()
	engine, err := engineRegistry.GetEngine(data.EngineConfig.ID)
	if err != nil {
		safeOutputsConfigLog.Printf("Engine lookup failed for %q: %v — skipping agent manifest file injection", data.EngineConfig.ID, err)
		return nil, nil
	}
	if engine == nil {
		return nil, nil
	}

	provider, ok := engine.(AgentFileProvider)
	if !ok {
		return nil, nil
	}

	return provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes()
}

// generateCustomJobToolDefinition creates an MCP tool definition for a custom safe-output job.
// Returns a map representing the tool definition in MCP format with name, description, and inputSchema.
func generateCustomJobToolDefinition(jobName string, jobConfig *SafeJobConfig) map[string]any {
	safeOutputsConfigLog.Printf("Generating tool definition for custom job: %s", jobName)

	description := jobConfig.Description
	if description == "" {
		description = fmt.Sprintf("Execute the %s custom job", jobName)
	}

	inputSchema := map[string]any{
		"type":                 "object",
		"properties":           make(map[string]any),
		"additionalProperties": false,
	}

	var requiredFields []string
	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		properties = make(map[string]any)
		inputSchema["properties"] = properties
	}

	for inputName, inputDef := range jobConfig.Inputs {
		property := buildCustomJobInputProperty(inputDef)
		if inputDef.Required {
			requiredFields = append(requiredFields, inputName)
		}
		properties[inputName] = property
	}

	if len(requiredFields) > 0 {
		sort.Strings(requiredFields)
		inputSchema["required"] = requiredFields
	}

	safeOutputsConfigLog.Printf("Generated tool definition for %s with %d inputs, %d required",
		jobName, len(jobConfig.Inputs), len(requiredFields))

	return map[string]any{
		"name":        jobName,
		"description": description,
		"inputSchema": inputSchema,
	}
}

func buildCustomJobInputProperty(inputDef *InputDefinition) map[string]any {
	property := map[string]any{}
	if inputDef.Description != "" {
		property["description"] = inputDef.Description
	}
	switch inputDef.Type {
	case "choice":
		property["type"] = "string"
		if len(inputDef.Options) > 0 {
			property["enum"] = inputDef.Options
		}
	case "boolean":
		property["type"] = "boolean"
	case "number":
		property["type"] = "number"
	default:
		property["type"] = "string"
	}
	if inputDef.Default != nil {
		property["default"] = inputDef.Default
	}
	return property
}
