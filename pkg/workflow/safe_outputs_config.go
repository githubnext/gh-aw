package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsConfigLog = logger.New("workflow:safe_outputs_config")

// ========================================
// Safe Output Configuration Extraction
// ========================================
//
// ## Schema Generation Architecture
//
// MCP tool schemas for Safe Outputs are managed through a hybrid approach:
//
// ### Static Schemas (30+ built-in safe output types)
// Defined in: pkg/workflow/js/safe_outputs_tools.json
// - Embedded at compile time via //go:embed directive in pkg/workflow/js.go
// - Contains complete MCP tool definitions with inputSchema for all built-in types
// - Examples: create_issue, create_pull_request, add_comment, update_project, etc.
// - Accessed via GetSafeOutputsToolsJSON() function
//
// ### Dynamic Schema Generation (custom safe-jobs)
// Implemented in: pkg/workflow/safe_outputs_config_generation.go
// - generateCustomJobToolDefinition() builds MCP tool schemas from SafeJobConfig
// - Converts job input definitions to JSON Schema format
// - Supports type mapping (string, boolean, number, choice/enum)
// - Enforces required fields and additionalProperties: false
// - Custom job tools are merged with static tools at runtime
//
// ### Schema Filtering
// Implemented in: pkg/workflow/safe_outputs_config_generation.go
// - generateFilteredToolsJSON() filters tools based on enabled safe-outputs
// - Only includes tools that are configured in the workflow frontmatter
// - Reduces MCP gateway overhead by exposing only necessary tools
//
// ### Validation
// Implemented in: pkg/workflow/safe_outputs_tools_schema_test.go
// - TestSafeOutputsToolsJSONCompliesWithMCPSchema validates against MCP spec
// - TestEachToolHasRequiredMCPFields checks name, description, inputSchema
// - TestNoTopLevelOneOfAllOfAnyOf prevents unsupported schema constructs
//
// This architecture ensures schema consistency by:
// 1. Using embedded JSON for static schemas (single source of truth)
// 2. Programmatic generation for dynamic schemas (type-safe)
// 3. Automated validation in CI (regression prevention)
//

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
	safeOutputsConfigLog.Print("Extracting safe-outputs configuration from frontmatter")
	var config *SafeOutputsConfig

	if output, exists := frontmatter["safe-outputs"]; exists {
		if outputMap, ok := output.(map[string]any); ok {
			safeOutputsConfigLog.Printf("Processing safe-outputs configuration with %d top-level keys", len(outputMap))
			config = &SafeOutputsConfig{}
			c.applyIssueHandlers(outputMap, config)
			c.applyPRHandlers(outputMap, config)
			c.applySecurityAndUploadHandlers(outputMap, config)
			c.applyLabelAndAssignmentHandlers(outputMap, config)
			c.applyDefaultFallbackHandlers(outputMap, config)
			c.parseSafeOutputsGlobalConfig(outputMap, config)
			c.parsePatchAndTimeoutSettings(outputMap, config)
			parseSafeOutputsMessageSettings(outputMap, config)
			parseSafeOutputsExtensionSettings(outputMap, config)
			c.parseSafeOutputsJobsAndActions(outputMap, config)
			if c.forceStaged {
				v := TemplatableBool("true")
				config.Staged = &v
			}
		}
	}

	c.applyDefaultThreatDetection(frontmatter, config)

	if config != nil {
		safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	} else {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
	}
	return config
}

// parseBaseSafeOutputConfig parses common fields (max, github-token, github-app, staged) from a config map.
// If defaultMax is provided (> 0), it will be set as the default value for config.Max
// before parsing the max field from configMap. Supports both integer values and GitHub
// Actions expression strings (e.g. "${{ inputs.max }}").
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	// Parse max (uses shared helper that sets default and handles expressions/integers)
	parseMaxField(configMap, config, defaultMax)

	// Parse github-token
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			safeOutputsConfigLog.Print("Parsed custom github-token from config")
			config.GitHubToken = githubTokenStr
		}
	}

	// Parse github-app (per-handler GitHub App credentials for token minting)
	if app, exists := configMap["github-app"]; exists {
		if appMap, ok := app.(map[string]any); ok {
			safeOutputsConfigLog.Print("Parsed custom github-app from config")
			config.GitHubApp = parseAppConfig(appMap)
		}
	}

	// Parse staged flag (per-handler staged mode)
	if err := preprocessBoolFieldAsString(configMap, "staged", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("Invalid staged value: %v", err)
	} else if staged, exists := configMap["staged"]; exists {
		if stagedStr, ok := staged.(string); ok && stagedStr != "" {
			safeOutputsConfigLog.Printf("Parsed staged flag: %s", stagedStr)
			value := TemplatableBool(stagedStr)
			config.Staged = &value
		}
	}

	// Parse samples list (hidden feature: deterministic replay samples for --use-samples).
	if samples, exists := configMap["samples"]; exists {
		parsed := parseSamplesValue(samples)
		if len(parsed) > 0 {
			safeOutputsConfigLog.Printf("Parsed %d samples entries", len(parsed))
			config.Samples = parsed
		}
	}
}

// parseSamplesValue normalizes a `samples` frontmatter value into a list of
// objects. Accepted shapes:
//   - YAML list of mappings: returned as-is
//   - single YAML mapping: wrapped into a one-element list
//
// Any other shape returns an empty slice — schema validation rejects those
// shapes upstream and we keep this parser strict to match.
func parseSamplesValue(samples any) []map[string]any {
	switch v := samples.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			} else if mStr, ok := item.(map[string]string); ok {
				converted := make(map[string]any, len(mStr))
				for k, s := range mStr {
					converted[k] = s
				}
				out = append(out, converted)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{v}
	default:
		return nil
	}
}

// SafeOutputStepConfig holds configuration for building a single safe output step
// within the consolidated safe-outputs job
type SafeOutputStepConfig struct {
	StepName                   string            // Human-readable step name (e.g., "Create Issue")
	StepID                     string            // Step ID for referencing outputs (e.g., "create_issue")
	Script                     string            // JavaScript script to execute (for inline mode)
	ScriptName                 string            // Name of the script in the registry (for file mode)
	CustomEnvVars              []string          // Environment variables specific to this step
	Condition                  ConditionNode     // Step-level condition (if clause)
	Token                      string            // GitHub token for this step
	UseCopilotRequestsToken    bool              // Whether to use Copilot requests token preference chain
	UseCopilotCodingAgentToken bool              // Whether to use Copilot coding agent token preference chain
	PreSteps                   []string          // Optional steps to run before the script step
	PostSteps                  []string          // Optional steps to run after the script step
	Outputs                    map[string]string // Outputs from this step
	ContinueOnError            bool              // Whether to continue the job even if this step fails (continue-on-error: true)
}

func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		safeOutputsConfigLog.Print("No safe-outputs configuration, skipping handler manager config")
		return
	}

	safeOutputsConfigLog.Print("Building handler manager configuration for safe-outputs")

	// Collect engine-specific manifest files and path prefixes (AgentFileProvider interface).
	extraManifestFiles, extraPathPrefixes := c.getEngineAgentFileInfo(data)
	fullManifestFiles := getAllManifestFiles(extraManifestFiles...)
	fullPathPrefixes := getProtectedPathPrefixes(extraPathPrefixes...)

	// For workflow_call relay workflows, inject the resolved platform repo and ref into the
	// dispatch_workflow handler config so dispatch targets the host repo, not the caller's.
	safeOutputs := resolveDispatchWorkflowSafeOutputs(data.SafeOutputs, data)

	// Build per-handler config map using the registry.
	config := populateHandlerManagerConfig(safeOutputs, data, fullManifestFiles, fullPathPrefixes)

	// Include top-level mentions configuration so the handler manager can pass it to
	// markdown-producing handlers that call sanitizeContent with allowed aliases.
	if safeOutputs.Mentions != nil {
		mentionsCfg := buildMentionsHandlerConfig(safeOutputs.Mentions)
		if len(mentionsCfg) > 0 {
			config["mentions"] = mentionsCfg
		}
	}

	if len(config) == 0 {
		safeOutputsConfigLog.Print("No handlers configured, skipping config env var")
		return
	}

	safeOutputsConfigLog.Printf("Marshaling handler config with %d handlers", len(config))
	configJSON, err := json.Marshal(config)
	if err != nil {
		safeOutputsConfigLog.Printf("Failed to marshal handler config: %v", err)
		return
	}
	configStr := string(configJSON)
	*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
	safeOutputsConfigLog.Printf("Added handler config env var: size=%d bytes", len(configStr))
}

// buildMentionsHandlerConfig converts a MentionsConfig into the map format used by
// GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG so safe_output_handler_manager.cjs can pass
// the top-level mentions policy through to mention-aware handlers.
func buildMentionsHandlerConfig(m *MentionsConfig) map[string]any {
	cfg := make(map[string]any)
	if m.Enabled != nil {
		cfg["enabled"] = *m.Enabled
	}
	if m.AllowedCollaborators != nil {
		cfg["allowedCollaborators"] = *m.AllowedCollaborators
	}
	if m.AllowContext != nil {
		cfg["allowContext"] = *m.AllowContext
	}
	if len(m.Allowed) > 0 {
		cfg["allowed"] = m.Allowed
	}
	if len(m.AllowedTeams) > 0 {
		cfg["allowedTeams"] = m.AllowedTeams
	}
	if m.Max != nil {
		cfg["max"] = *m.Max
	}
	return cfg
}

// safeOutputsWithDispatchTargetRepo returns a shallow copy of cfg with the dispatch_workflow
// TargetRepoSlug overridden to targetRepo. Only DispatchWorkflow is deep-copied; all other
// pointer fields remain shared. This avoids mutating the original config.
func safeOutputsWithDispatchTargetRepo(cfg *SafeOutputsConfig, targetRepo string) *SafeOutputsConfig {
	dispatchCopy := *cfg.DispatchWorkflow
	dispatchCopy.TargetRepoSlug = targetRepo
	configCopy := *cfg
	configCopy.DispatchWorkflow = &dispatchCopy
	return &configCopy
}

// safeOutputsWithDispatchTargetRef returns a shallow copy of cfg with the dispatch_workflow
// TargetRef overridden to targetRef. Only DispatchWorkflow is deep-copied; all other
// pointer fields remain shared. This avoids mutating the original config.
func safeOutputsWithDispatchTargetRef(cfg *SafeOutputsConfig, targetRef string) *SafeOutputsConfig {
	dispatchCopy := *cfg.DispatchWorkflow
	dispatchCopy.TargetRef = targetRef
	configCopy := *cfg
	configCopy.DispatchWorkflow = &dispatchCopy
	return &configCopy
}

// getEngineAgentFileInfo returns the engine-specific manifest filenames and path prefixes
// by type-asserting the active engine to AgentFileProvider.  Returns empty slices when
// the engine is not set or does not implement the interface.
func (c *Compiler) getEngineAgentFileInfo(data *WorkflowData) (manifestFiles []string, pathPrefixes []string) {
	if data == nil || data.EngineConfig == nil {
		return nil, nil
	}
	engine, err := c.engineRegistry.GetEngine(data.EngineConfig.ID)
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
	safeOutputsConfigLog.Printf("Engine %s provides AgentFileProvider: files=%v, prefixes=%v",
		data.EngineConfig.ID, provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes())
	return provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes()
}
