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

	addStandardHandlerConfigs(safeOutputsConfig, data, engineManifestFiles, engineManifestPathPrefixes)

	// Safe-jobs configuration: custom output types that run as separate GitHub Actions jobs.
	// These are not standard handlers but must be in config.json so the ingestion step can
	// validate and route those output types.
	addSafeJobsConfig(safeOutputsConfig, data.SafeOutputs)

	// Safe-scripts configuration: script output types handled inline by the handler manager.
	addSafeScriptsConfig(safeOutputsConfig, data.SafeOutputs)

	// Safe-actions configuration: custom GitHub Actions exposed as safe output tools.
	// The normalized action names are added as config keys so both MCP server implementations
	// recognise them as enabled tools (the tool schema is already in tools.json via
	// tools_meta.json; the MCP server just needs to see the name in config.json).
	if err := addSafeActionsConfig(safeOutputsConfig, data.SafeOutputs); err != nil {
		return "", err
	}

	// Mentions configuration: controls which @mentions are allowed in AI output.
	// This is consumed by the ingestion step, not by standard handlers.
	addMentionsConfig(safeOutputsConfig, data.SafeOutputs)

	// Max bot mentions: limits bot trigger references (e.g. "fixes #123") in AI output.
	// Consumed by the ingestion step as a global config knob.
	// Store as integer when possible (matching original behavior), or as expression string.
	addMaxBotMentionsConfig(safeOutputsConfig, data.SafeOutputs)

	// Push-repo-memory configuration: enables the push_repo_memory MCP tool for early
	// size validation during the agent session.
	addPushRepoMemoryConfig(safeOutputsConfig, data.RepoMemoryConfig)

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

func addStandardHandlerConfigs(safeOutputsConfig map[string]any, data *WorkflowData, engineManifestFiles, engineManifestPathPrefixes []string) {
	for handlerName, builder := range handlerRegistry {
		handlerCfg := builder(data.SafeOutputs)
		if handlerCfg == nil {
			continue
		}
		injectCurrentCheckoutPatchWorkspacePath(handlerName, handlerCfg, data)
		injectCheckoutMapping(handlerName, handlerCfg, data)
		addProtectedFilesConfig(handlerCfg, engineManifestFiles, engineManifestPathPrefixes)
		addDataSchemaConfig(handlerName, handlerCfg, data.SafeOutputs)
		safeOutputsConfig[handlerName] = handlerCfg
	}
}

func addProtectedFilesConfig(handlerCfg map[string]any, engineManifestFiles, engineManifestPathPrefixes []string) {
	excludeFiles := ParseStringArrayFromConfig(handlerCfg, "_protected_files_exclude", nil)
	// Strip the internal sentinel key used by the handler manager for compile-time
	// exclusion processing — it must not be forwarded to the runtime config.json.
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
	// Compute which top-level dot-folder prefixes are excluded so the runtime
	// dot-folder check can skip them.
	if dotFolderExcludes := getDotFolderExcludes(excludeFiles); len(dotFolderExcludes) > 0 {
		handlerCfg["protected_dot_folder_excludes"] = dotFolderExcludes
	}
}

func addDataSchemaConfig(handlerName string, handlerCfg map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil || !safeOutputs.DataEnabled || !isDataSchemaEnabledType(handlerName) {
		return
	}
	handlerCfg["data_enabled"] = true
	if safeOutputs.NormalizedDataSchema != nil {
		handlerCfg["data_schema"] = safeOutputs.NormalizedDataSchema
	} else if strings.TrimSpace(safeOutputs.DataSchemaExpression) != "" {
		handlerCfg["data_schema"] = safeOutputs.DataSchemaExpression
	}
}

func addSafeJobsConfig(safeOutputsConfig map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}
	if len(safeOutputs.Jobs) == 0 {
		return
	}
	safeOutputsConfigLog.Printf("Processing %d safe job configurations", len(safeOutputs.Jobs))
	for jobName, jobConfig := range safeOutputs.Jobs {
		safeOutputsConfigLog.Printf("Generating config for safe job: %s", jobName)
		safeOutputsConfig[jobName] = buildSafeOutputInputsConfig(jobConfig.Description, jobConfig.Output, jobConfig.Max, jobConfig.Inputs)
	}
}

func addSafeScriptsConfig(safeOutputsConfig map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}
	if len(safeOutputs.Scripts) == 0 {
		return
	}
	safeOutputsConfigLog.Printf("Processing %d safe script configurations", len(safeOutputs.Scripts))
	for scriptName, scriptConfig := range safeOutputs.Scripts {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
		safeOutputsConfigLog.Printf("Generating config for safe script: %s (normalized: %s)", scriptName, normalizedName)
		safeOutputsConfig[normalizedName] = buildSafeOutputInputsConfig(scriptConfig.Description, "", 0, scriptConfig.Inputs)
	}
}

func addSafeActionsConfig(safeOutputsConfig map[string]any, safeOutputs *SafeOutputsConfig) error {
	if safeOutputs == nil {
		return nil
	}
	if len(safeOutputs.Actions) == 0 {
		return nil
	}
	safeOutputsConfigLog.Printf("Processing %d safe action configurations", len(safeOutputs.Actions))
	for actionName := range safeOutputs.Actions {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(actionName)
		if _, exists := safeOutputsConfig[normalizedName]; exists {
			return fmt.Errorf(
				"safe-outputs action %q has a normalized name %q that conflicts with an existing safe outputs config entry; rename the action to avoid the conflict",
				actionName,
				normalizedName,
			)
		}
		safeOutputsConfigLog.Printf("Adding safe action to config: %s (normalized: %s)", actionName, normalizedName)
		safeOutputsConfig[normalizedName] = true
	}
	return nil
}

func addMentionsConfig(safeOutputsConfig map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}
	if safeOutputs.Mentions == nil {
		return
	}
	mentionsConfig := buildMentionsHandlerConfig(safeOutputs.Mentions)
	if len(mentionsConfig) > 0 {
		safeOutputsConfig["mentions"] = mentionsConfig
	}
}

func addMaxBotMentionsConfig(safeOutputsConfig map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}
	if safeOutputs.MaxBotMentions == nil {
		return
	}

	value := *safeOutputs.MaxBotMentions
	if n := templatableIntValue(safeOutputs.MaxBotMentions); n > 0 {
		safeOutputsConfig["max_bot_mentions"] = n
	} else if strings.HasPrefix(value, "${{") {
		safeOutputsConfig["max_bot_mentions"] = value
	}
}

func addPushRepoMemoryConfig(safeOutputsConfig map[string]any, repoMemoryConfig *RepoMemoryConfig) {
	if repoMemoryConfig == nil || len(repoMemoryConfig.Memories) == 0 {
		return
	}

	var memories []map[string]any
	for _, memory := range repoMemoryConfig.Memories {
		memoryConfig := map[string]any{
			"id":             memory.ID,
			"dir":            constants.TmpRepoMemoryDir + memory.ID,
			"max_file_size":  memory.MaxFileSize,
			"max_patch_size": memory.MaxPatchSize,
			"max_file_count": memory.MaxFileCount,
		}
		if memory.FormatJSON {
			memoryConfig["format_json"] = true
		}
		if memory.Validation != nil {
			memoryConfig["validation"] = map[string]any{
				"script":  memory.Validation.Script,
				"timeout": memoryValidationTimeoutSeconds(memory.Validation),
			}
		}
		memories = append(memories, memoryConfig)
	}

	safeOutputsConfig["push_repo_memory"] = map[string]any{
		"memories": memories,
	}
	safeOutputsConfigLog.Printf("Added push_repo_memory config with %d memory entries", len(repoMemoryConfig.Memories))
}

func buildSafeOutputInputsConfig(description string, output string, max int, inputs map[string]*InputDefinition) map[string]any {
	config := map[string]any{}
	if description != "" {
		config["description"] = description
	}
	if output != "" {
		config["output"] = output
	}
	if max > 0 {
		config["max"] = max
	}
	if len(inputs) > 0 {
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
		config["inputs"] = inputsConfig
	}
	return config
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

	inputSchema, requiredCount := buildCustomJobInputSchema(jobConfig.Inputs)

	safeOutputsConfigLog.Printf("Generated tool definition for %s with %d inputs, %d required",
		jobName, len(jobConfig.Inputs), requiredCount)

	return map[string]any{
		"name":        jobName,
		"description": description,
		"inputSchema": inputSchema,
	}
}

func buildCustomJobInputSchema(inputs map[string]*InputDefinition) (map[string]any, int) {
	properties := make(map[string]any)
	inputSchema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}

	var requiredFields []string
	for inputName, inputDef := range inputs {
		properties[inputName] = buildCustomJobInputProperty(inputDef)
		if inputDef.Required {
			requiredFields = append(requiredFields, inputName)
		}
	}
	if len(requiredFields) > 0 {
		sort.Strings(requiredFields)
		inputSchema["required"] = requiredFields
	}

	return inputSchema, len(requiredFields)
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
