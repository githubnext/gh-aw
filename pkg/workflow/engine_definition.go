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
// NewEngineCatalog registers the built-in engines: claude, codex, copilot, gemini, opencode, crush.
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
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var engineCatalogLog = logger.New("workflow:engine_definition")

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
// It supports OAuth client-credentials flows, custom header injection, and
// template-based secret references.
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
	// The "Ref" suffix indicates this is a reference to a GitHub Actions secret name,
	// not the secret value itself. Required for oauth-client-credentials strategy.
	ClientIDRef string `yaml:"client-id,omitempty"`

	// ClientSecretRef is the secret name that holds the OAuth client secret.
	// The "Ref" suffix indicates this is a reference to a GitHub Actions secret name,
	// not the secret value itself. Required for oauth-client-credentials strategy.
	ClientSecretRef string `yaml:"client-secret,omitempty"`

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

// EngineCapabilitiesDefinition captures declarative engine capabilities loaded from
// engine definition frontmatter.
type EngineCapabilitiesDefinition struct {
	ToolsAllowlist   bool `yaml:"tools-allowlist,omitempty"`
	MaxTurns         bool `yaml:"max-turns,omitempty"`
	WebSearch        bool `yaml:"web-search,omitempty"`
	MaxContinuations bool `yaml:"max-continuations,omitempty"`
	NativeAgentFile  bool `yaml:"native-agent-file,omitempty"`
	BareMode         bool `yaml:"bare-mode,omitempty"`
}

// ToRuntimeCapabilities converts the declarative capabilities definition into the
// runtime EngineCapabilities struct used by CodingAgentEngine implementations.
func (d EngineCapabilitiesDefinition) ToRuntimeCapabilities() EngineCapabilities {
	return EngineCapabilities(d)
}

// EngineManifestDefinition describes engine-specific files and folders that alter
// agent behaviour and must be protected from untrusted pull requests.
type EngineManifestDefinition struct {
	Files        []string `yaml:"files,omitempty"`
	PathPrefixes []string `yaml:"path-prefixes,omitempty"`
}

// EngineInstallationDefinition describes how an engine CLI is installed.
type EngineInstallationDefinition struct {
	PackageManager     string `yaml:"package-manager,omitempty"`
	PackageName        string `yaml:"package-name,omitempty"`
	Version            string `yaml:"version,omitempty"`
	StepName           string `yaml:"step-name,omitempty"`
	BinaryName         string `yaml:"binary-name,omitempty"`
	IncludeNodeSetup   bool   `yaml:"include-node-setup,omitempty"`
	PostInstallScripts bool   `yaml:"post-install-scripts,omitempty"`
	Cooldown           bool   `yaml:"cooldown,omitempty"`
	VerifyCommand      string `yaml:"verify-command,omitempty"`
	VerifyStepName     string `yaml:"verify-step-name,omitempty"`
	DocumentationURL   string `yaml:"docs-url,omitempty"`
}

// EngineConfigFileDefinition describes a configuration file that should be written
// before executing the engine CLI.
type EngineConfigFileDefinition struct {
	Path          string `yaml:"path,omitempty"`
	StepName      string `yaml:"step-name,omitempty"`
	Content       string `yaml:"content,omitempty"`
	MergeStrategy string `yaml:"merge-strategy,omitempty"`
}

// EngineExecutionDefinition describes the common CLI execution pattern used by
// behavior-defined engines.
type EngineExecutionDefinition struct {
	CommandName            string   `yaml:"command-name,omitempty"`
	Args                   []string `yaml:"args,omitempty"`
	StepName               string   `yaml:"step-name,omitempty"`
	ModelEnvVarName        string   `yaml:"model-env-var,omitempty"`
	ModelEnvProviderPrefix string   `yaml:"model-env-provider-prefix,omitempty"`
	ModelFlag              string   `yaml:"model-flag,omitempty"`
	MCPConfigEnvVar        string   `yaml:"mcp-config-env-var,omitempty"`
	MCPConfigFlag          string   `yaml:"mcp-config-flag,omitempty"`
	WriteTimestamp         bool     `yaml:"write-timestamp,omitempty"`
	ProviderEnvMode        string   `yaml:"provider-env-mode,omitempty"`
	// Env holds additional static environment variables to inject into the
	// execution step.  Values are rendered verbatim and are not filtered
	// through the secrets allowlist, so they must not contain secret values.
	Env map[string]string `yaml:"env,omitempty"`
}

// EngineMCPDefinition describes how to render MCP configuration for a
// behavior-defined engine.
type EngineMCPDefinition struct {
	ConfigPath string `yaml:"config-path,omitempty"`
}

// EngineBehaviorDefinition captures declarative runtime behaviour for a custom
// engine definition.
type EngineBehaviorDefinition struct {
	SecretStrategy      string                        `yaml:"secret-strategy,omitempty"`
	SupportedEnvVarKeys []string                      `yaml:"supported-env-var-keys,omitempty"`
	Capabilities        EngineCapabilitiesDefinition  `yaml:"capabilities,omitempty"`
	Manifest            *EngineManifestDefinition     `yaml:"manifest,omitempty"`
	Installation        *EngineInstallationDefinition `yaml:"installation,omitempty"`
	ConfigFile          *EngineConfigFileDefinition   `yaml:"config-file,omitempty"`
	Execution           *EngineExecutionDefinition    `yaml:"execution,omitempty"`
	MCP                 *EngineMCPDefinition          `yaml:"mcp,omitempty"`
	// HarnessScript is the JavaScript source of a Node.js harness that spawns the
	// engine CLI.  When non-empty the script is written to
	// ${RUNNER_TEMP}/gh-aw/actions/<engine-id>_harness.cjs before execution and the
	// engine is launched via:
	//   node <harness-path> <command-name> [args...]
	// The harness can read process.env.GH_AW_PROMPT for the prompt-file path and
	// process.env.AWF_REFLECT_ENABLED / the AWF reflect JSON file to dynamically
	// configure the engine CLI at runtime.
	HarnessScript string `yaml:"harness-script,omitempty"`
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
	ID               string `yaml:"id"`
	DisplayName      string `yaml:"display-name,omitempty"`
	Description      string `yaml:"description,omitempty"`
	Experimental     bool   `yaml:"experimental,omitempty"`
	GHSkillAgentName string `yaml:"gh-skill-agent-name,omitempty"`
	// RuntimeID maps to the CodingAgentEngine registered in EngineRegistry.
	// Defaults to ID when omitted.
	RuntimeID string                    `yaml:"runtime-id,omitempty"`
	Provider  ProviderSelection         `yaml:"provider,omitempty"`
	Models    ModelSelection            `yaml:"models,omitempty"`
	Auth      []AuthBinding             `yaml:"auth,omitempty"`
	Options   map[string]any            `yaml:"options,omitempty"`
	Behaviors *EngineBehaviorDefinition `yaml:"behaviors,omitempty"`
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
// pre-registers the built-in engine definitions (claude, codex, copilot, gemini, opencode, crush)
// loaded from the embedded Markdown files in data/engines/*.md.
func NewEngineCatalog(registry *EngineRegistry) *EngineCatalog {
	catalog := &EngineCatalog{
		definitions: make(map[string]*EngineDefinition),
		registry:    registry,
	}

	for _, def := range loadBuiltinEngineDefinitions() {
		catalog.Register(def)
	}

	engineCatalogLog.Printf("Engine catalog initialized with %d built-in definitions", len(catalog.definitions))
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
	ids := sliceutil.SortedKeys(c.definitions)
	return ids
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
	engineCatalogLog.Printf("Resolving engine: %s", id)

	// Exact catalog lookup
	if def, ok := c.definitions[id]; ok {
		engineCatalogLog.Printf("Exact catalog match found for engine: %s (runtimeID=%s)", id, def.RuntimeID)
		runtime, err := c.registry.GetEngine(def.RuntimeID)
		if err != nil {
			return nil, fmt.Errorf("engine %q definition references unknown runtime %q: %w", id, def.RuntimeID, err)
		}
		return &ResolvedEngineTarget{Definition: def, Config: config, Runtime: runtime}, nil
	}

	// Fall back to runtime-ID prefix lookup for backward compat (e.g. "codex-experimental")
	runtime, err := c.registry.GetEngineByPrefix(id)
	if err == nil {
		engineCatalogLog.Printf("Engine %q resolved via runtime-ID prefix fallback to %q", id, runtime.GetID())
		def := &EngineDefinition{
			ID:          id,
			DisplayName: runtime.GetDisplayName(),
			Description: runtime.GetDescription(),
			RuntimeID:   runtime.GetID(),
		}
		return &ResolvedEngineTarget{Definition: def, Config: config, Runtime: runtime}, nil
	}

	// Engine not found — produce a helpful validation error matching the existing format
	engineCatalogLog.Printf("Engine not found: %s", id)
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

	return nil, errors.New(errMsg)
}
