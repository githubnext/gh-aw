package workflow

// ResolvedEngineTarget holds the fully resolved engine configuration for a workflow compilation.
//
// The compiler resolves the raw frontmatter into a single ResolvedEngineTarget early in the
// compilation pipeline and passes this resolved result to all downstream consumers. This
// eliminates the pattern where each consumer re-reads the raw EngineConfig independently,
// which was fragile and led to inconsistencies (e.g., threat detection copying only a subset
// of engine fields).
//
// Usage pattern:
//
//	// Resolve once during compilation setup
//	target := NewResolvedEngineTarget(engineDef, agenticEngine, engineConfig)
//
//	// Consume everywhere downstream
//	steps := target.Runtime.GetInstallationSteps(workflowData)
//	secrets := target.Runtime.GetRequiredSecretNames(workflowData)
//	model := target.Config.Model
type ResolvedEngineTarget struct {
	// Definition is the engine's declarative definition from the catalog.
	// It contains the engine's identity, display metadata, and secret specifications.
	Definition EngineDefinition

	// Runtime is the engine's runtime adapter implementing CodingAgentEngine.
	// It provides the actual workflow step generation, log parsing, and MCP config rendering.
	Runtime CodingAgentEngine

	// Config is the parsed engine configuration from workflow frontmatter.
	// It holds workflow-specific overrides such as model, version, env vars, and args.
	Config *EngineConfig
}

// NewResolvedEngineTarget creates a ResolvedEngineTarget from the resolved components.
// This is called after the compiler has resolved the engine definition, runtime adapter,
// and frontmatter configuration.
func NewResolvedEngineTarget(definition EngineDefinition, runtime CodingAgentEngine, config *EngineConfig) ResolvedEngineTarget {
	return ResolvedEngineTarget{
		Definition: definition,
		Runtime:    runtime,
		Config:     config,
	}
}

// EngineID returns the resolved engine ID, sourced from the config if set,
// falling back to the definition ID. This handles the case where the engine
// was resolved via prefix matching (e.g., "codex-experimental" → "codex").
//
// If both Config.ID and Definition.ID are empty (which should not happen when
// NewResolvedEngineTarget is used correctly), an empty string is returned.
func (r *ResolvedEngineTarget) EngineID() string {
	if r.Config != nil && r.Config.ID != "" {
		return r.Config.ID
	}
	return r.Definition.ID
}
