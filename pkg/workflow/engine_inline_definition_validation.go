// This file provides inline engine definition and auth validation for agentic workflows.
//
// # Inline Engine Definition Validation
//
// This file validates inline engine definitions declared via engine.runtime and
// engine.provider in the workflow frontmatter. It also handles registration of
// validated definitions into the session engine catalog.
//
// # Validation Functions
//
//   - validateEngineInlineDefinition() - Validates inline engine runtime.id against the registry
//   - registerInlineEngineDefinition() - Registers a validated inline engine definition
//   - validateEngineAuthDefinition() - Validates auth strategy fields for inline provider auth
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates engine.runtime or engine.provider fields
//   - It validates inline engine auth strategy configurations
//   - It registers or modifies the engine catalog for inline definitions
//   - It checks provider, auth, or request shape fields
//
// For engine driver and script validation, see engine_driver_validation.go.
// For engine version and MCP timeout validation, see engine_validation.go.
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var engineInlineDefinitionLog = logger.New("workflow:engine_inline_definition_validation")

// validateEngineInlineDefinition validates an inline engine definition parsed from
// engine.runtime + optional engine.provider in the workflow frontmatter.
// Returns an error if:
//   - The required runtime.id field is missing
//   - The runtime.id does not match a known runtime adapter
func (c *Compiler) validateEngineInlineDefinition(config *EngineConfig) error {
	if !config.IsInlineDefinition {
		return nil
	}

	engineInlineDefinitionLog.Printf("Validating inline engine definition: runtimeID=%s", config.ID)

	if config.ID == "" {
		return fmt.Errorf("inline engine definition is missing required 'runtime.id' field.\n\nExample:\nengine:\n  runtime:\n    id: codex\n\nSee: %s", constants.DocsEnginesURL)
	}

	// Validate that runtime.id maps to a known runtime adapter.
	if !c.engineRegistry.IsValidEngine(config.ID) {
		// Try prefix match for backward compatibility (e.g. "codex-experimental")
		if matched, err := c.engineRegistry.GetEngineByPrefix(config.ID); err == nil {
			engineInlineDefinitionLog.Printf("Inline engine runtime.id %q matched via prefix to runtime %q", config.ID, matched.GetID())
		} else {
			validEngines := c.engineRegistry.GetSupportedEngines()
			suggestions := parser.FindClosestMatches(config.ID, validEngines, 1)
			enginesStr := strings.Join(validEngines, ", ")

			errMsg := fmt.Sprintf("inline engine definition references unknown runtime.id: %s. Known runtime IDs are: %s.\n\nExample:\nengine:\n  runtime:\n    id: codex\n\nSee: %s",
				config.ID, enginesStr, constants.DocsEnginesURL)
			if len(suggestions) > 0 {
				errMsg = fmt.Sprintf("inline engine definition references unknown runtime.id: %s. Known runtime IDs are: %s.\n\nDid you mean: %s?\n\nExample:\nengine:\n  runtime:\n    id: codex\n\nSee: %s",
					config.ID, enginesStr, suggestions[0], constants.DocsEnginesURL)
			}
			return errors.New(errMsg)
		}
	}

	return nil
}

// registerInlineEngineDefinition registers an inline engine definition in the session
// catalog. If the runtime ID already exists in the catalog (e.g. a built-in), the
// existing display name and description are preserved while provider overrides are applied.
func (c *Compiler) registerInlineEngineDefinition(config *EngineConfig) {
	def := &EngineDefinition{
		ID:          config.ID,
		RuntimeID:   config.ID,
		DisplayName: config.ID,
		Description: "Inline engine definition from workflow frontmatter",
	}

	// Preserve display name and description from existing built-in entry if available.
	if existing := c.engineCatalog.Get(config.ID); existing != nil {
		def.DisplayName = existing.DisplayName
		def.Description = existing.Description
		def.Models = existing.Models
		// Copy existing provider/auth as defaults; inline values below fully replace them
		// when present (replacement, not merge).
		def.Provider = existing.Provider
	}

	// Apply inline provider overrides.
	if config.InlineProviderID != "" {
		def.Provider = ProviderSelection{Name: config.InlineProviderID}
	}

	// Prefer the full AuthDefinition over the legacy simple-secret path.
	if config.InlineProviderAuth != nil {
		// Normalise strategy: treat empty strategy as api-key when a secret is set.
		auth := config.InlineProviderAuth
		if auth.Strategy == "" && auth.Secret != "" {
			auth.Strategy = AuthStrategyAPIKey
		}
		def.Provider.Auth = auth
	}

	if config.InlineProviderRequest != nil {
		def.Provider.Request = config.InlineProviderRequest
	}

	engineInlineDefinitionLog.Printf("Registering inline engine definition in session catalog: id=%s, runtimeID=%s, providerID=%s",
		def.ID, def.RuntimeID, def.Provider.Name)
	c.engineCatalog.Register(def)
}

// validateEngineAuthDefinition validates AuthDefinition fields for an inline engine definition.
// Returns an error describing the first (or all, in non-fail-fast mode) validation problems found.
func (c *Compiler) validateEngineAuthDefinition(config *EngineConfig) error {
	auth := config.InlineProviderAuth
	if auth == nil {
		return nil
	}

	engineInlineDefinitionLog.Printf("Validating engine auth definition: strategy=%s", auth.Strategy)

	switch auth.Strategy {
	case AuthStrategyOAuthClientCreds:
		// oauth-client-credentials requires tokenUrl, clientId, clientSecret.
		if auth.TokenURL == "" {
			return fmt.Errorf("engine auth: strategy 'oauth-client-credentials' requires 'auth.token-url' to be set.\n\nExample:\nengine:\n  runtime:\n    id: codex\n  provider:\n    auth:\n      strategy: oauth-client-credentials\n      token-url: https://auth.example.com/oauth/token\n      client-id: MY_CLIENT_ID_SECRET\n      client-secret: MY_CLIENT_SECRET_SECRET\n\nSee: %s", constants.DocsEnginesURL)
		}
		if auth.ClientIDRef == "" {
			return fmt.Errorf("engine auth: strategy 'oauth-client-credentials' requires 'auth.client-id' to be set.\n\nSee: %s", constants.DocsEnginesURL)
		}
		if auth.ClientSecretRef == "" {
			return fmt.Errorf("engine auth: strategy 'oauth-client-credentials' requires 'auth.client-secret' to be set.\n\nSee: %s", constants.DocsEnginesURL)
		}
		// For oauth, header-name is required (the token must go somewhere).
		if auth.HeaderName == "" {
			return fmt.Errorf("engine auth: strategy 'oauth-client-credentials' requires 'auth.header-name' to be set (e.g. 'api-key' or 'Authorization').\n\nSee: %s", constants.DocsEnginesURL)
		}
	case AuthStrategyAPIKey:
		// api-key requires a secret value and a header-name so the caller knows where to inject the key.
		if auth.Secret == "" {
			return fmt.Errorf("engine auth: strategy 'api-key' requires 'auth.secret' to be set.\n\nSee: %s", constants.DocsEnginesURL)
		}
		if auth.HeaderName == "" {
			return fmt.Errorf("engine auth: strategy 'api-key' requires 'auth.header-name' to be set (e.g. 'api-key' or 'x-api-key').\n\nSee: %s", constants.DocsEnginesURL)
		}
	case AuthStrategyBearer, "":
		// bearer strategy and unset strategy (simple backwards-compat secret) require a secret value.
		if auth.Secret == "" {
			return fmt.Errorf("engine auth: strategy 'bearer' (or unset) requires 'auth.secret' to be set.\n\nSee: %s", constants.DocsEnginesURL)
		}
	default:
		validStrategies := []string{
			string(AuthStrategyAPIKey),
			string(AuthStrategyOAuthClientCreds),
			string(AuthStrategyBearer),
		}
		return fmt.Errorf("engine auth: unknown strategy %q. Valid strategies are: %s.\n\nSee: %s",
			auth.Strategy, strings.Join(validStrategies, ", "), constants.DocsEnginesURL)
	}

	engineInlineDefinitionLog.Printf("Engine auth definition is valid: strategy=%s", auth.Strategy)
	return nil
}
