// This file defines the engine definition layer: declarative metadata types for AI engines,
// a catalog of registered definitions, and a resolved target that combines definition,
// config, and runtime adapter.
//
// # Architecture
//
// The engine definition layer sits on top of the existing EngineRegistry runtime layer:
//
//	EngineDefinition  – declarative metadata for a single engine entry
//	EngineCatalog     – registry of EngineDefinition entries with a Resolve() method
//	ResolvedEngineTarget – result of resolving an engine ID: definition + config + runtime
//
// The existing EngineRegistry and CodingAgentEngine interfaces are unchanged; the catalog
// is an additional layer that maps logical engine IDs to runtime adapters.
//
// # Built-in Engines
//
// NewEngineCatalog registers the four built-in engines: claude, codex, copilot, gemini.
// Each EngineDefinition carries the engine's RuntimeID which maps to the corresponding
// CodingAgentEngine registered in the EngineRegistry.
//
// # Resolve()
//
// EngineCatalog.Resolve() performs:
//  1. Exact catalog ID lookup
//  2. Runtime-ID prefix fallback (for backward compat, e.g. "codex-experimental")
//  3. Formatted validation error when engine is unknown
package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

// ProviderSelection identifies the AI provider for an engine (e.g. "anthropic", "openai").
type ProviderSelection struct {
	Name string
}

// ModelSelection specifies the default and supported models for an engine.
type ModelSelection struct {
	Default   string
	Supported []string
}

// AuthBinding maps a logical authentication role to a secret name.
type AuthBinding struct {
	Role   string
	Secret string
}

// EngineDefinition holds the declarative metadata for an AI engine.
// It is separate from the runtime adapter (CodingAgentEngine) to allow the catalog
// layer to carry identity and provider information without coupling to implementation.
type EngineDefinition struct {
	ID          string
	DisplayName string
	Description string
	RuntimeID   string // maps to the CodingAgentEngine registered in EngineRegistry
	Provider    ProviderSelection
	Models      ModelSelection
	Auth        []AuthBinding
	Options     map[string]any
}

// EngineCatalog is a collection of EngineDefinition entries backed by an EngineRegistry
// for runtime adapter resolution.
type EngineCatalog struct {
	definitions map[string]*EngineDefinition
	registry    *EngineRegistry
}

// ResolvedEngineTarget is the result of resolving an engine ID through the catalog.
// It combines the EngineDefinition, the caller-supplied EngineConfig, and the resolved
// CodingAgentEngine runtime adapter.
type ResolvedEngineTarget struct {
	Definition *EngineDefinition
	Config     *EngineConfig     // resolved merged config supplied by the caller
	Runtime    CodingAgentEngine // resolved adapter from the EngineRegistry
}

// NewEngineCatalog creates an EngineCatalog that wraps the given EngineRegistry and
// pre-registers the four built-in engine definitions (claude, codex, copilot, gemini).
func NewEngineCatalog(registry *EngineRegistry) *EngineCatalog {
	catalog := &EngineCatalog{
		definitions: make(map[string]*EngineDefinition),
		registry:    registry,
	}

	catalog.Register(&EngineDefinition{
		ID:          "claude",
		DisplayName: "Claude Code",
		Description: "Uses Claude Code with full MCP tool support and allow-listing",
		RuntimeID:   "claude",
		Provider:    ProviderSelection{Name: "anthropic"},
		Auth: []AuthBinding{
			{Role: "api-key", Secret: "ANTHROPIC_API_KEY"},
		},
	})

	catalog.Register(&EngineDefinition{
		ID:          "codex",
		DisplayName: "Codex",
		Description: "Uses OpenAI Codex CLI with MCP server support",
		RuntimeID:   "codex",
		Provider:    ProviderSelection{Name: "openai"},
		Auth: []AuthBinding{
			{Role: "api-key", Secret: "CODEX_API_KEY"},
		},
	})

	catalog.Register(&EngineDefinition{
		ID:          "copilot",
		DisplayName: "GitHub Copilot CLI",
		Description: "Uses GitHub Copilot CLI with MCP server support",
		RuntimeID:   "copilot",
		Provider:    ProviderSelection{Name: "github"},
	})

	catalog.Register(&EngineDefinition{
		ID:          "gemini",
		DisplayName: "Google Gemini CLI",
		Description: "Google Gemini CLI with headless mode and LLM gateway support",
		RuntimeID:   "gemini",
		Provider:    ProviderSelection{Name: "google"},
	})

	return catalog
}

// Register adds or replaces an EngineDefinition in the catalog.
func (c *EngineCatalog) Register(def *EngineDefinition) {
	c.definitions[def.ID] = def
}

// Resolve returns a ResolvedEngineTarget for the given engine ID and config.
// Resolution order:
//  1. Exact match in the catalog by ID
//  2. Prefix match in the underlying EngineRegistry (backward compat, e.g. "codex-experimental")
//  3. Returns a formatted validation error when no match is found
func (c *EngineCatalog) Resolve(id string, config *EngineConfig) (*ResolvedEngineTarget, error) {
	// Exact catalog lookup
	if def, ok := c.definitions[id]; ok {
		runtime, err := c.registry.GetEngine(def.RuntimeID)
		if err != nil {
			return nil, fmt.Errorf("engine %q definition references unknown runtime %q: %w", id, def.RuntimeID, err)
		}
		return &ResolvedEngineTarget{Definition: def, Config: config, Runtime: runtime}, nil
	}

	// Fall back to runtime-ID prefix lookup for backward compat (e.g. "codex-experimental")
	runtime, err := c.registry.GetEngineByPrefix(id)
	if err == nil {
		def := &EngineDefinition{
			ID:          id,
			DisplayName: runtime.GetDisplayName(),
			Description: runtime.GetDescription(),
			RuntimeID:   runtime.GetID(),
		}
		return &ResolvedEngineTarget{Definition: def, Config: config, Runtime: runtime}, nil
	}

	// Engine not found — produce a helpful validation error matching the existing format
	validEngines := c.registry.GetSupportedEngines()
	suggestions := parser.FindClosestMatches(id, validEngines, 1)
	enginesStr := strings.Join(validEngines, ", ")

	errMsg := fmt.Sprintf("invalid engine: %s. Valid engines are: %s.\n\nExample:\nengine: copilot\n\nSee: %s",
		id,
		enginesStr,
		constants.DocsEnginesURL)

	if len(suggestions) > 0 {
		errMsg = fmt.Sprintf("invalid engine: %s. Valid engines are: %s.\n\nDid you mean: %s?\n\nExample:\nengine: copilot\n\nSee: %s",
			id,
			enginesStr,
			suggestions[0],
			constants.DocsEnginesURL)
	}

	return nil, fmt.Errorf("%s", errMsg)
}
