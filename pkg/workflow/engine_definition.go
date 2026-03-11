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
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

// AuthStrategy identifies how an engine authenticates with its provider.
type AuthStrategy string

const (
	// AuthStrategyAPIKey uses a direct API key sent via a header (default when Secret is set).
	AuthStrategyAPIKey AuthStrategy = "api-key"
	// AuthStrategyOAuthClientCreds exchanges client credentials for a bearer token before each call.
	AuthStrategyOAuthClientCreds AuthStrategy = "oauth-client-credentials"
	// AuthStrategyBearer sends a pre-obtained token as a standard Authorization: Bearer header.
	AuthStrategyBearer AuthStrategy = "bearer"
)

// AuthDefinition describes how the engine authenticates with a provider backend.
// It extends the simple AuthBinding model to support OAuth client-credentials flows,
// custom header injection, and template-based secret references.
//
// For backwards compatibility, a plain auth.secret field without a strategy is treated as
// AuthStrategyAPIKey.
type AuthDefinition struct {
	// Strategy selects the authentication flow (api-key, oauth-client-credentials, bearer).
	// Defaults to api-key when Secret is non-empty and Strategy is unset.
	Strategy AuthStrategy `yaml:"strategy,omitempty"`

	// Secret is the env-var / GitHub Actions secret name that holds the raw API key or token.
	// Required for api-key and bearer strategies.
	Secret string `yaml:"secret,omitempty"`

	// TokenURL is the OAuth token endpoint (e.g. "https://auth.example.com/oauth/token").
	// Required for oauth-client-credentials strategy.
	TokenURL string `yaml:"token-url,omitempty"`

	// ClientIDRef is the secret name that holds the OAuth client ID.
	// Required for oauth-client-credentials strategy.
	ClientIDRef string `yaml:"client-id-ref,omitempty"`

	// ClientSecretRef is the secret name that holds the OAuth client secret.
	// Required for oauth-client-credentials strategy.
	ClientSecretRef string `yaml:"client-secret-ref,omitempty"`

	// TokenField is the JSON field name in the token response that contains the access token.
	// Defaults to "access_token" when empty.
	TokenField string `yaml:"token-field,omitempty"`

	// HeaderName is the HTTP header to inject the token into (e.g. "api-key").
	// Required when strategy is not bearer (bearer always uses Authorization header).
	HeaderName string `yaml:"header-name,omitempty"`
}

// RequestShape describes non-standard URL and body transformations applied to each
// API call before it is sent to the provider backend.
type RequestShape struct {
	// PathTemplate is a URL path template with {model} and other variable placeholders
	// (e.g. "/openai/deployments/{model}/chat/completions").
	PathTemplate string `yaml:"path-template,omitempty"`

	// Query holds static or template query-parameter values appended to every request
	// (e.g. {"api-version": "2024-10-01-preview"}).
	Query map[string]string `yaml:"query,omitempty"`

	// BodyInject holds key/value pairs injected into the JSON request body before sending
	// (e.g. {"appKey": "{APP_KEY_SECRET}"}).
	BodyInject map[string]string `yaml:"body-inject,omitempty"`
}

// ProviderSelection identifies the AI provider for an engine (e.g. "anthropic", "openai").
// It optionally carries advanced authentication and request-shaping configuration for
// non-standard backends.
type ProviderSelection struct {
	Name    string          `yaml:"name,omitempty"`
	Auth    *AuthDefinition `yaml:"auth,omitempty"`
	Request *RequestShape   `yaml:"request,omitempty"`
}

// ModelSelection specifies the default and supported models for an engine.
type ModelSelection struct {
	Default   string   `yaml:"default,omitempty"`
	Supported []string `yaml:"supported,omitempty"`
}

// AuthBinding maps a logical authentication role to a secret name.
type AuthBinding struct {
	Role   string `yaml:"role"`
	Secret string `yaml:"secret"`
}

// RequiredSecretNames returns the env-var names that must be provided at runtime for
// this AuthDefinition. Returns an empty slice when Auth is nil.
func (a *AuthDefinition) RequiredSecretNames() []string {
	if a == nil {
		return []string{}
	}
	var secrets []string
	switch a.Strategy {
	case AuthStrategyOAuthClientCreds:
		if a.ClientIDRef != "" {
			secrets = append(secrets, a.ClientIDRef)
		}
		if a.ClientSecretRef != "" {
			secrets = append(secrets, a.ClientSecretRef)
		}
	default:
		// api-key, bearer, or unset strategy – Secret is the raw credential.
		if a.Secret != "" {
			secrets = append(secrets, a.Secret)
		}
	}
	return secrets
}

// EngineDefinition holds the declarative metadata for an AI engine.
// It is separate from the runtime adapter (CodingAgentEngine) to allow the catalog
// layer to carry identity and provider information without coupling to implementation.
type EngineDefinition struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display-name,omitempty"`
	Description string `yaml:"description,omitempty"`
	// RuntimeID maps to the CodingAgentEngine registered in EngineRegistry.
	// Defaults to ID when omitted.
	RuntimeID string            `yaml:"runtime-id,omitempty"`
	Provider  ProviderSelection `yaml:"provider,omitempty"`
	Models    ModelSelection    `yaml:"models,omitempty"`
	Auth      []AuthBinding     `yaml:"auth,omitempty"`
	Options   map[string]any    `yaml:"options,omitempty"`
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
// pre-registers the four built-in engine definitions (claude, codex, copilot, gemini)
// loaded from the embedded Markdown files in data/engines/*.md.
func NewEngineCatalog(registry *EngineRegistry) *EngineCatalog {
	catalog := &EngineCatalog{
		definitions: make(map[string]*EngineDefinition),
		registry:    registry,
	}

	for _, def := range loadBuiltinEngineDefinitions() {
		catalog.Register(def)
	}

	return catalog
}

// Register adds or replaces an EngineDefinition in the catalog.
func (c *EngineCatalog) Register(def *EngineDefinition) {
	c.definitions[def.ID] = def
}

// Get returns the EngineDefinition for the given ID, or nil if not found.
func (c *EngineCatalog) Get(id string) *EngineDefinition {
	return c.definitions[id]
}

// IDs returns a sorted list of all engine IDs in the catalog.
func (c *EngineCatalog) IDs() []string {
	ids := make([]string, 0, len(c.definitions))
	for id := range c.definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// DisplayNames returns a list of engine display names in sorted ID order.
func (c *EngineCatalog) DisplayNames() []string {
	ids := c.IDs()
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, c.definitions[id].DisplayName)
	}
	return names
}

// All returns all engine definitions in sorted ID order.
func (c *EngineCatalog) All() []*EngineDefinition {
	ids := c.IDs()
	defs := make([]*EngineDefinition, 0, len(ids))
	for _, id := range ids {
		defs = append(defs, c.definitions[id])
	}
	return defs
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
