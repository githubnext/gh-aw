package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

// ========================================
// Handler Manager Config Generation
// ========================================
//
// This file produces the GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG env var consumed
// by the handler manager at runtime, using the handlerRegistry and the fluent
// handlerConfigBuilder API.
//
// The handlerRegistry is the single source of truth for handler keys and field
// contracts. generateSafeOutputsConfig() in safe_outputs_config_generation.go
// derives config.json from this same registry so both consumers stay in sync
// without a separate generation path.

var compilerSafeOutputsConfigLog = logger.New("workflow:compiler_safe_outputs_config")

// getEffectiveFooterForTemplatable returns the effective footer as a templatable string.
// If the local string footer is set, use it; otherwise convert the global bool footer.
// Returns nil if neither is set (default to true in JavaScript).
func getEffectiveFooterForTemplatable(localFooter *string, globalFooter *bool) *string {
	if localFooter != nil {
		return localFooter
	}
	if globalFooter != nil {
		var s string
		if *globalFooter {
			s = "true"
		} else {
			s = "false"
		}
		return &s
	}
	return nil
}

// getEffectiveFooterString returns the effective footer string value for a config.
// If the local string footer is set, use it; otherwise convert the global bool footer.
// Returns nil if neither is set (default to "always" in JavaScript).
func getEffectiveFooterString(localFooter *string, globalFooter *bool) *string {
	if localFooter != nil {
		return localFooter
	}
	if globalFooter != nil {
		var s string
		if *globalFooter {
			s = "always"
		} else {
			s = "none"
		}
		return &s
	}
	return nil
}

// handlerConfigBuilder provides a fluent API for building handler configurations
type handlerConfigBuilder struct {
	config map[string]any
}

// newHandlerConfigBuilder creates a new handler config builder
func newHandlerConfigBuilder() *handlerConfigBuilder {
	return &handlerConfigBuilder{
		config: map[string]any{},
	}
}

// AddIfPositive adds an integer field only if the value is greater than 0
func (b *handlerConfigBuilder) AddIfPositive(key string, value int) *handlerConfigBuilder {
	if value > 0 {
		b.config[key] = value
	}
	return b
}

// AddIfNotEmpty adds a string field only if the value is not empty
func (b *handlerConfigBuilder) AddIfNotEmpty(key string, value string) *handlerConfigBuilder {
	if value != "" {
		b.config[key] = value
	}
	return b
}

// AddStringSlice adds a string slice field only if the slice is not empty
func (b *handlerConfigBuilder) AddStringSlice(key string, value []string) *handlerConfigBuilder {
	if len(value) > 0 {
		b.config[key] = value
	}
	return b
}

// AddBoolPtr adds a boolean pointer field only if the pointer is not nil
func (b *handlerConfigBuilder) AddBoolPtr(key string, value *bool) *handlerConfigBuilder {
	if value != nil {
		b.config[key] = *value
	}
	return b
}

// AddBoolPtrOrDefault adds a boolean field, using default if pointer is nil
func (b *handlerConfigBuilder) AddBoolPtrOrDefault(key string, value *bool, defaultValue bool) *handlerConfigBuilder {
	if value != nil {
		b.config[key] = *value
	} else {
		b.config[key] = defaultValue
	}
	return b
}

// AddStringPtr adds a string pointer field only if the pointer is not nil
func (b *handlerConfigBuilder) AddStringPtr(key string, value *string) *handlerConfigBuilder {
	if value != nil {
		b.config[key] = *value
	}
	return b
}

// AddDefault adds a field with a default value unconditionally
func (b *handlerConfigBuilder) AddDefault(key string, value any) *handlerConfigBuilder {
	b.config[key] = value
	return b
}

// AddIfTrue adds a boolean field only if the value is true
func (b *handlerConfigBuilder) AddIfTrue(key string, value bool) *handlerConfigBuilder {
	if value {
		b.config[key] = true
	}
	return b
}

// Build returns the built configuration map
func (b *handlerConfigBuilder) Build() map[string]any {
	return b.config
}

// handlerBuilder is a function that builds a handler config from SafeOutputsConfig
type handlerBuilder func(*SafeOutputsConfig) map[string]any

// handlerRegistry maps handler names to their builder functions.
// Entries are registered via init() in domain-specific files:
//   - compiler_safe_outputs_handlers_issues.go
//   - compiler_safe_outputs_handlers_prs.go
//   - compiler_safe_outputs_handlers_discussions.go
//   - compiler_safe_outputs_handlers_misc.go
var handlerRegistry = map[string]handlerBuilder{}

func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		compilerSafeOutputsConfigLog.Print("No safe-outputs configuration, skipping handler manager config")
		return
	}

	compilerSafeOutputsConfigLog.Print("Building handler manager configuration for safe-outputs")
	config := make(map[string]map[string]any)

	// Collect engine-specific manifest files and path prefixes (AgentFileProvider interface).
	// These are merged with the global runtime-derived lists so that engine-specific
	// instruction files (e.g. CLAUDE.md, .claude/, AGENTS.md) are automatically protected.
	extraManifestFiles, extraPathPrefixes := c.getEngineAgentFileInfo(data)
	fullManifestFiles := getAllManifestFiles(extraManifestFiles...)
	fullPathPrefixes := getProtectedPathPrefixes(extraPathPrefixes...)

	// For workflow_call relay workflows, inject the resolved platform repo and ref into the
	// dispatch_workflow handler config so dispatch targets the host repo, not the caller's.
	safeOutputs := data.SafeOutputs
	if hasWorkflowCallTrigger(data.On) && safeOutputs.DispatchWorkflow != nil {
		if safeOutputs.DispatchWorkflow.TargetRepoSlug == "" {
			safeOutputs = safeOutputsWithDispatchTargetRepo(safeOutputs, "${{ needs.activation.outputs.target_repo }}")
			compilerSafeOutputsConfigLog.Print("Injecting target_repo into dispatch_workflow config for workflow_call relay")
		}
		if safeOutputs.DispatchWorkflow.TargetRef == "" {
			safeOutputs = safeOutputsWithDispatchTargetRef(safeOutputs, "${{ needs.activation.outputs.target_ref }}")
			compilerSafeOutputsConfigLog.Print("Injecting target_ref into dispatch_workflow config for workflow_call relay")
		}
	}

	// Build configuration for each handler using the registry
	for handlerName, builder := range handlerRegistry {
		handlerConfig := builder(safeOutputs)
		// Include handler if:
		// 1. It returns a non-nil config (explicitly enabled, even if empty)
		// 2. For auto-enabled handlers, include even with empty config
		if handlerConfig != nil {
			// Augment protected-files protection with engine-specific files for handlers that use it.
			if _, hasProtected := handlerConfig["protected_files"]; hasProtected {
				handlerConfig["protected_files"] = fullManifestFiles
				handlerConfig["protected_path_prefixes"] = fullPathPrefixes
			}
			compilerSafeOutputsConfigLog.Printf("Adding %s handler configuration", handlerName)
			config[handlerName] = handlerConfig
		}
	}

	// Only add the env var if there are handlers to configure
	if len(config) > 0 {
		compilerSafeOutputsConfigLog.Printf("Marshaling handler config with %d handlers", len(config))
		configJSON, err := json.Marshal(config)
		if err != nil {
			consolidatedSafeOutputsLog.Printf("Failed to marshal handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
		compilerSafeOutputsConfigLog.Printf("Added handler config env var: size=%d bytes", len(configStr))
	} else {
		compilerSafeOutputsConfigLog.Print("No handlers configured, skipping config env var")
	}
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
		compilerSafeOutputsConfigLog.Printf("Engine lookup failed for %q: %v — skipping agent manifest file injection", data.EngineConfig.ID, err)
		return nil, nil
	}
	if engine == nil {
		return nil, nil
	}
	provider, ok := engine.(AgentFileProvider)
	if !ok {
		return nil, nil
	}
	compilerSafeOutputsConfigLog.Printf("Engine %s provides AgentFileProvider: files=%v, prefixes=%v",
		data.EngineConfig.ID, provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes())
	return provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes()
}
