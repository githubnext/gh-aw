package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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

	// Standard handler configs — sourced from handlerRegistry (same as GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG)
	for handlerName, builder := range handlerRegistry {
		if handlerCfg := builder(data.SafeOutputs); handlerCfg != nil {
			safeOutputsConfig[handlerName] = handlerCfg
		}
	}

	// Safe-jobs configuration: custom output types that run as separate GitHub Actions jobs.
	// These are not standard handlers but must be in config.json so the ingestion step can
	// validate and route those output types.
	if len(data.SafeOutputs.Jobs) > 0 {
		safeOutputsConfigLog.Printf("Processing %d safe job configurations", len(data.SafeOutputs.Jobs))
		for jobName, jobConfig := range data.SafeOutputs.Jobs {
			safeOutputsConfigLog.Printf("Generating config for safe job: %s", jobName)
			safeJobConfig := map[string]any{}
			if jobConfig.Description != "" {
				safeJobConfig["description"] = jobConfig.Description
			}
			if jobConfig.Output != "" {
				safeJobConfig["output"] = jobConfig.Output
			}
			if len(jobConfig.Inputs) > 0 {
				inputsConfig := make(map[string]any)
				for inputName, inputDef := range jobConfig.Inputs {
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
				safeJobConfig["inputs"] = inputsConfig
			}
			safeOutputsConfig[jobName] = safeJobConfig
		}
	}

	// Safe-scripts configuration: script output types handled inline by the handler manager.
	if len(data.SafeOutputs.Scripts) > 0 {
		safeOutputsConfigLog.Printf("Processing %d safe script configurations", len(data.SafeOutputs.Scripts))
		for scriptName, scriptConfig := range data.SafeOutputs.Scripts {
			normalizedName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
			safeOutputsConfigLog.Printf("Generating config for safe script: %s (normalized: %s)", scriptName, normalizedName)
			safeScriptConfigMap := map[string]any{}
			if scriptConfig.Description != "" {
				safeScriptConfigMap["description"] = scriptConfig.Description
			}
			if len(scriptConfig.Inputs) > 0 {
				inputsConfig := make(map[string]any)
				for inputName, inputDef := range scriptConfig.Inputs {
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
				safeScriptConfigMap["inputs"] = inputsConfig
			}
			safeOutputsConfig[normalizedName] = safeScriptConfigMap
		}
	}

	// Safe-actions configuration: custom GitHub Actions exposed as safe output tools.
	// The normalized action names are added as config keys so both MCP server implementations
	// recognise them as enabled tools (the tool schema is already in tools.json via
	// tools_meta.json; the MCP server just needs to see the name in config.json).
	if len(data.SafeOutputs.Actions) > 0 {
		safeOutputsConfigLog.Printf("Processing %d safe action configurations", len(data.SafeOutputs.Actions))
		for actionName := range data.SafeOutputs.Actions {
			normalizedName := stringutil.NormalizeSafeOutputIdentifier(actionName)
			if _, exists := safeOutputsConfig[normalizedName]; exists {
				return "", fmt.Errorf(
					"safe-outputs action %q has a normalized name %q that conflicts with an existing safe outputs config entry; rename the action to avoid the conflict",
					actionName,
					normalizedName,
				)
			}
			safeOutputsConfigLog.Printf("Adding safe action to config: %s (normalized: %s)", actionName, normalizedName)
			safeOutputsConfig[normalizedName] = true
		}
	}

	// Mentions configuration: controls which @mentions are allowed in AI output.
	// This is consumed by the ingestion step, not by standard handlers.
	if data.SafeOutputs.Mentions != nil {
		mentionsConfig := make(map[string]any)
		if data.SafeOutputs.Mentions.Enabled != nil {
			mentionsConfig["enabled"] = *data.SafeOutputs.Mentions.Enabled
		}
		if data.SafeOutputs.Mentions.AllowTeamMembers != nil {
			mentionsConfig["allowTeamMembers"] = *data.SafeOutputs.Mentions.AllowTeamMembers
		}
		if data.SafeOutputs.Mentions.AllowContext != nil {
			mentionsConfig["allowContext"] = *data.SafeOutputs.Mentions.AllowContext
		}
		if len(data.SafeOutputs.Mentions.Allowed) > 0 {
			mentionsConfig["allowed"] = data.SafeOutputs.Mentions.Allowed
		}
		if data.SafeOutputs.Mentions.Max != nil {
			mentionsConfig["max"] = *data.SafeOutputs.Mentions.Max
		}
		if len(mentionsConfig) > 0 {
			safeOutputsConfig["mentions"] = mentionsConfig
		}
	}

	// Max bot mentions: limits bot trigger references (e.g. "fixes #123") in AI output.
	// Consumed by the ingestion step as a global config knob.
	// Store as integer when possible (matching original behavior), or as expression string.
	if data.SafeOutputs.MaxBotMentions != nil {
		v := *data.SafeOutputs.MaxBotMentions
		if n := templatableIntValue(data.SafeOutputs.MaxBotMentions); n > 0 {
			safeOutputsConfig["max_bot_mentions"] = n
		} else if strings.HasPrefix(v, "${{") {
			safeOutputsConfig["max_bot_mentions"] = v
		}
	}

	// Push-repo-memory configuration: enables the push_repo_memory MCP tool for early
	// size validation during the agent session.
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		var memories []map[string]any
		for _, memory := range data.RepoMemoryConfig.Memories {
			memories = append(memories, map[string]any{
				"id":             memory.ID,
				"dir":            "/tmp/gh-aw/repo-memory/" + memory.ID,
				"max_file_size":  memory.MaxFileSize,
				"max_patch_size": memory.MaxPatchSize,
				"max_file_count": memory.MaxFileCount,
			})
		}
		safeOutputsConfig["push_repo_memory"] = map[string]any{
			"memories": memories,
		}
		safeOutputsConfigLog.Printf("Added push_repo_memory config with %d memory entries", len(data.RepoMemoryConfig.Memories))
	}

	if len(safeOutputsConfig) == 0 {
		return "", nil
	}
	configJSON, _ := json.Marshal(safeOutputsConfig)
	safeOutputsConfigLog.Printf("Safe outputs config generation complete: %d tool types configured", len(safeOutputsConfig))
	return string(configJSON), nil
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
	properties := inputSchema["properties"].(map[string]any)

	for inputName, inputDef := range jobConfig.Inputs {
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
