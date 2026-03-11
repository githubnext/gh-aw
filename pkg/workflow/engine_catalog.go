package workflow

import (
	"fmt"
	"sort"
	"sync"

	"github.com/github/gh-aw/pkg/constants"
)

// EngineSecretSpec describes the secret requirements for an engine.
// It captures all authentication-related metadata needed to set up and validate
// the secrets an engine requires.
type EngineSecretSpec struct {
	// Primary is the main secret name required for this engine (e.g., "COPILOT_GITHUB_TOKEN")
	Primary string

	// Alternatives are alternative secret names that can also satisfy authentication
	// (e.g., Codex accepts either "OPENAI_API_KEY" or "CODEX_API_KEY")
	Alternatives []string

	// EnvVarName is an alternative environment variable name if different from Primary (optional)
	EnvVarName string

	// KeyURL is the URL where users can obtain the API key/token for this engine
	KeyURL string

	// WhenNeeded is a human-readable description of when this secret is needed
	WhenNeeded string
}

// EngineDefinition is the declarative definition of an agentic engine.
//
// It represents engine metadata independent of the runtime adapter implementation,
// making the engine catalog the single source of truth for engine identity, secrets,
// and display information. Runtime behavior (install steps, execution steps, etc.)
// remains in the CodingAgentEngine adapter implementations.
//
// Built-in engines (copilot, claude, codex, gemini) are represented as EngineDefinition
// entries in the built-in catalog. External or organization-defined engines can be added
// to the catalog without modifying Go code.
//
// Example usage:
//
//	def, err := GetGlobalEngineCatalog().GetDefinition("copilot")
//	fmt.Println(def.DisplayName) // "GitHub Copilot CLI"
//	fmt.Println(def.Secrets.Primary) // "COPILOT_GITHUB_TOKEN"
type EngineDefinition struct {
	// ID is the unique identifier used in workflow frontmatter (e.g., "copilot", "claude")
	ID string

	// DisplayName is the human-readable name (e.g., "GitHub Copilot CLI")
	DisplayName string

	// Description describes the engine's capabilities
	Description string

	// Secrets describes the secret requirements for this engine
	Secrets EngineSecretSpec

	// IsExperimental marks the engine as experimental/preview
	IsExperimental bool
}

// EngineCatalog is the single source of truth for engine definitions.
//
// It provides methods to query engine metadata and derive the lists used by
// the compiler, CLI prompts, schema validation, and documentation. By centralizing
// engine definitions, the catalog eliminates the duplicated engine metadata that
// previously existed across constants, schema enums, and CLI helpers.
//
// The built-in catalog (returned by GetGlobalEngineCatalog) contains definitions
// for all built-in engines. The EngineRegistry remains separate and manages the
// runtime adapter implementations (CodingAgentEngine).
//
// Engine metadata is defined here once; downstream consumers derive their lists:
//   - AgenticEngines: use GetAllEngineIDs()
//   - CLI selection options: use GetEngineOptions()
//   - Schema enums: derived from GetAllEngineIDs()
type EngineCatalog struct {
	definitions map[string]EngineDefinition
	// ordered preserves insertion order for deterministic iteration
	ordered []string
}

var (
	globalEngineCatalog     *EngineCatalog
	globalEngineCatalogOnce sync.Once
)

// GetGlobalEngineCatalog returns the global engine catalog containing all built-in engine definitions.
// This is the single source of truth for engine metadata.
// The catalog is initialized lazily using sync.Once to ensure thread-safe access.
func GetGlobalEngineCatalog() *EngineCatalog {
	globalEngineCatalogOnce.Do(func() {
		globalEngineCatalog = newBuiltInEngineCatalog()
	})
	return globalEngineCatalog
}

// newBuiltInEngineCatalog creates the default catalog containing all built-in engine definitions.
// This function is the single place where built-in engine metadata is defined.
func newBuiltInEngineCatalog() *EngineCatalog {
	catalog := &EngineCatalog{
		definitions: make(map[string]EngineDefinition),
	}

	// Register built-in engine definitions in a consistent order.
	// This order is used for CLI selection menus and documentation.
	catalog.register(EngineDefinition{
		ID:          string(constants.CopilotEngine),
		DisplayName: "GitHub Copilot CLI",
		Description: "GitHub Copilot CLI with agent support",
		Secrets: EngineSecretSpec{
			Primary:    "COPILOT_GITHUB_TOKEN",
			KeyURL:     "https://github.com/settings/personal-access-tokens/new",
			WhenNeeded: "Copilot workflows (CLI, engine, agent tasks, etc.)",
		},
	})

	catalog.register(EngineDefinition{
		ID:          string(constants.ClaudeEngine),
		DisplayName: "Claude Code",
		Description: "Anthropic Claude Code coding agent",
		Secrets: EngineSecretSpec{
			Primary:    "ANTHROPIC_API_KEY",
			KeyURL:     "https://console.anthropic.com/settings/keys",
			WhenNeeded: "Claude engine workflows",
		},
	})

	catalog.register(EngineDefinition{
		ID:          string(constants.CodexEngine),
		DisplayName: "Codex",
		Description: "OpenAI Codex/GPT engine",
		Secrets: EngineSecretSpec{
			Primary:      "OPENAI_API_KEY",
			Alternatives: []string{"CODEX_API_KEY"},
			KeyURL:       "https://platform.openai.com/api-keys",
			WhenNeeded:   "Codex/OpenAI engine workflows",
		},
	})

	catalog.register(EngineDefinition{
		ID:          string(constants.GeminiEngine),
		DisplayName: "Google Gemini CLI",
		Description: "Google Gemini CLI with headless mode and LLM gateway support",
		Secrets: EngineSecretSpec{
			Primary:    "GEMINI_API_KEY",
			KeyURL:     "https://aistudio.google.com/apikey",
			WhenNeeded: "Gemini engine workflows",
		},
	})

	return catalog
}

// register adds an engine definition to the catalog.
func (c *EngineCatalog) register(def EngineDefinition) {
	if _, exists := c.definitions[def.ID]; !exists {
		c.ordered = append(c.ordered, def.ID)
	}
	c.definitions[def.ID] = def
}

// GetDefinition returns the EngineDefinition for the given engine ID.
// Returns an error if the engine ID is not found in the catalog.
func (c *EngineCatalog) GetDefinition(id string) (EngineDefinition, error) {
	def, exists := c.definitions[id]
	if !exists {
		return EngineDefinition{}, fmt.Errorf("unknown engine: %s", id)
	}
	return def, nil
}

// GetAllEngineIDs returns the IDs of all engines in the catalog, in registration order.
// This replaces the hard-coded constants.AgenticEngines list and includes all engines
// (including gemini, which was previously missing from that list).
func (c *EngineCatalog) GetAllEngineIDs() []string {
	ids := make([]string, len(c.ordered))
	copy(ids, c.ordered)
	return ids
}

// GetEngineOptions returns the engine options for CLI selection menus.
// The returned slice replaces constants.EngineOptions and is derived from the catalog,
// ensuring CLI options stay in sync with the catalog definitions.
func (c *EngineCatalog) GetEngineOptions() []constants.EngineOption {
	opts := make([]constants.EngineOption, 0, len(c.ordered))
	for _, id := range c.ordered {
		def := c.definitions[id]
		opts = append(opts, constants.EngineOption{
			Value:              def.ID,
			Label:              def.DisplayName,
			Description:        def.Description,
			SecretName:         def.Secrets.Primary,
			AlternativeSecrets: def.Secrets.Alternatives,
			EnvVarName:         def.Secrets.EnvVarName,
			KeyURL:             def.Secrets.KeyURL,
			WhenNeeded:         def.Secrets.WhenNeeded,
		})
	}
	return opts
}

// GetAllSecretNames returns all unique primary and alternative secret names for all
// engines in the catalog. This is used for secret enumeration and validation.
func (c *EngineCatalog) GetAllSecretNames() []string {
	seen := make(map[string]bool)
	var secrets []string

	for _, id := range c.ordered {
		def := c.definitions[id]
		if def.Secrets.Primary != "" && !seen[def.Secrets.Primary] {
			seen[def.Secrets.Primary] = true
			secrets = append(secrets, def.Secrets.Primary)
		}
		for _, alt := range def.Secrets.Alternatives {
			if alt != "" && !seen[alt] {
				seen[alt] = true
				secrets = append(secrets, alt)
			}
		}
	}

	sort.Strings(secrets)
	return secrets
}

// IsKnownEngine returns true if the given engine ID is registered in the catalog.
func (c *EngineCatalog) IsKnownEngine(id string) bool {
	_, exists := c.definitions[id]
	return exists
}
