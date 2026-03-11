// This file provides engine validation for agentic workflows.
//
// # Engine Validation
//
// This file validates engine configurations used in agentic workflows.
// Validation ensures that engine IDs are supported and that only one engine
// specification exists across the main workflow and all included files.
//
// # Validation Functions
//
//   - validateEngine() - Validates that a given engine ID is supported
//   - validateSingleEngineSpecification() - Validates that only one engine field exists across all files
//
// # Validation Pattern: Engine Registry
//
// Engine validation uses the compiler's engine registry:
//   - Supports exact engine ID matching (e.g., "copilot", "claude")
//   - Supports prefix matching for backward compatibility (e.g., "codex-experimental")
//   - Empty engine IDs are valid and use the default engine
//   - Detailed logging of validation steps for debugging
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates engine IDs or engine configurations
//   - It checks engine registry entries
//   - It validates engine-specific settings
//   - It validates engine field consistency across imports
//
// For engine configuration extraction, see engine.go.
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

var engineValidationLog = newValidationLogger("engine")

// validateEngineInlineDefinition validates an inline engine definition parsed from
// engine.runtime + optional engine.provider in the workflow frontmatter.
// Returns an error if:
//   - The required runtime.id field is missing
//   - The runtime.id does not match a known runtime adapter
func (c *Compiler) validateEngineInlineDefinition(config *EngineConfig) error {
	if !config.IsInlineDefinition {
		return nil
	}

	engineValidationLog.Printf("Validating inline engine definition: runtimeID=%s", config.ID)

	if config.ID == "" {
		return fmt.Errorf("inline engine definition is missing required 'runtime.id' field.\n\nExample:\nengine:\n  runtime:\n    id: codex\n\nSee: %s", constants.DocsEnginesURL)
	}

	// Validate that runtime.id maps to a known runtime adapter.
	if !c.engineRegistry.IsValidEngine(config.ID) {
		// Try prefix match for backward compatibility (e.g. "codex-experimental")
		if matched, err := c.engineRegistry.GetEngineByPrefix(config.ID); err == nil {
			engineValidationLog.Printf("Inline engine runtime.id %q matched via prefix to runtime %q", config.ID, matched.GetID())
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
			return fmt.Errorf("%s", errMsg)
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
		def.Auth = existing.Auth
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
		// Keep legacy AuthBinding in sync for callers that still read def.Auth.
		// When an AuthDefinition is provided, always reset legacy bindings to avoid
		// leaking stale secrets from existing engine definitions.
		def.Auth = nil
		if auth.Secret != "" {
			def.Auth = []AuthBinding{{Role: string(auth.Strategy), Secret: auth.Secret}}
		}
	} else if config.InlineProviderSecret != "" {
		def.Auth = []AuthBinding{{Role: "api-key", Secret: config.InlineProviderSecret}}
	}

	if config.InlineProviderRequest != nil {
		def.Provider.Request = config.InlineProviderRequest
	}

	engineValidationLog.Printf("Registering inline engine definition in session catalog: id=%s, runtimeID=%s, providerID=%s",
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

	engineValidationLog.Printf("Validating engine auth definition: strategy=%s", auth.Strategy)

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

	engineValidationLog.Printf("Engine auth definition is valid: strategy=%s", auth.Strategy)
	return nil
}

// validateEngine validates that the given engine ID is supported
func (c *Compiler) validateEngine(engineID string) error {
	if engineID == "" {
		engineValidationLog.Print("No engine ID specified, will use default")
		return nil // Empty engine is valid (will use default)
	}

	engineValidationLog.Printf("Validating engine ID: %s", engineID)

	// First try exact match
	if c.engineRegistry.IsValidEngine(engineID) {
		engineValidationLog.Printf("Engine ID %s is valid (exact match)", engineID)
		return nil
	}

	// Try prefix match for backward compatibility (e.g., "codex-experimental")
	engine, err := c.engineRegistry.GetEngineByPrefix(engineID)
	if err == nil {
		engineValidationLog.Printf("Engine ID %s matched by prefix to: %s", engineID, engine.GetID())
		return nil
	}

	engineValidationLog.Printf("Engine ID %s not found: %v", engineID, err)

	// Get list of valid engine IDs from the engine registry
	validEngines := c.engineRegistry.GetSupportedEngines()

	// Try to find close matches for "did you mean" suggestion
	suggestions := parser.FindClosestMatches(engineID, validEngines, 1)

	// Build comma-separated list of valid engines for error message
	enginesStr := strings.Join(validEngines, ", ")

	// Build error message with helpful context
	errMsg := fmt.Sprintf("invalid engine: %s. Valid engines are: %s.\n\nExample:\nengine: copilot\n\nSee: %s",
		engineID,
		enginesStr,
		constants.DocsEnginesURL)

	// Add "did you mean" suggestion if we found a close match
	if len(suggestions) > 0 {
		errMsg = fmt.Sprintf("invalid engine: %s. Valid engines are: %s.\n\nDid you mean: %s?\n\nExample:\nengine: copilot\n\nSee: %s",
			engineID,
			enginesStr,
			suggestions[0],
			constants.DocsEnginesURL)
	}

	return fmt.Errorf("%s", errMsg)
}

// validateSingleEngineSpecification validates that only one engine field exists across all files
func (c *Compiler) validateSingleEngineSpecification(mainEngineSetting string, includedEnginesJSON []string) (string, error) {
	var allEngines []string

	// Add main engine if specified
	if mainEngineSetting != "" {
		allEngines = append(allEngines, mainEngineSetting)
	}

	// Add included engines
	for _, engineJSON := range includedEnginesJSON {
		if engineJSON != "" {
			allEngines = append(allEngines, engineJSON)
		}
	}

	// Check count
	if len(allEngines) == 0 {
		return "", nil // No engine specified anywhere, will use default
	}

	if len(allEngines) > 1 {
		return "", fmt.Errorf("multiple engine fields found (%d engine specifications detected). Only one engine field is allowed across the main workflow and all included files. Remove duplicate engine specifications to keep only one.\n\nExample:\nengine: copilot\n\nSee: %s", len(allEngines), constants.DocsEnginesURL)
	}

	// Exactly one engine found - parse and return it
	if mainEngineSetting != "" {
		return mainEngineSetting, nil
	}

	// Must be from included file
	var firstEngine any
	if err := json.Unmarshal([]byte(includedEnginesJSON[0]), &firstEngine); err != nil {
		return "", fmt.Errorf("failed to parse included engine configuration: %w. Expected string or object format.\n\nExample (string):\nengine: copilot\n\nExample (object):\nengine:\n  id: copilot\n  model: gpt-4\n\nSee: %s", err, constants.DocsEnginesURL)
	}

	// Handle string format
	if engineStr, ok := firstEngine.(string); ok {
		return engineStr, nil
	} else if engineObj, ok := firstEngine.(map[string]any); ok {
		// Handle object format - return the ID
		if id, hasID := engineObj["id"]; hasID {
			if idStr, ok := id.(string); ok {
				return idStr, nil
			}
		}
	}

	return "", fmt.Errorf("invalid engine configuration in included file, missing or invalid 'id' field. Expected string or object with 'id' field.\n\nExample (string):\nengine: copilot\n\nExample (object):\nengine:\n  id: copilot\n  model: gpt-4\n\nSee: %s", constants.DocsEnginesURL)
}

// validatePluginSupport validates that plugins are only used with engines that support them
func (c *Compiler) validatePluginSupport(pluginInfo *PluginInfo, agenticEngine CodingAgentEngine) error {
	// No plugins specified, validation passes
	if pluginInfo == nil || len(pluginInfo.Plugins) == 0 {
		return nil
	}

	engineValidationLog.Printf("Validating plugin support for engine: %s", agenticEngine.GetID())

	// Check if the engine supports plugins
	if !agenticEngine.SupportsPlugins() {
		// Build error message listing the plugins that were specified
		pluginsList := strings.Join(pluginInfo.Plugins, ", ")

		// Get list of engines that support plugins from the engine registry
		var supportedEngines []string
		for _, engineID := range c.engineRegistry.GetSupportedEngines() {
			if engine, err := c.engineRegistry.GetEngine(engineID); err == nil {
				if engine.SupportsPlugins() {
					supportedEngines = append(supportedEngines, engineID)
				}
			}
		}

		// Build the list of supported engines for the error message
		var supportedEnginesMsg string
		if len(supportedEngines) == 0 {
			supportedEnginesMsg = "No engines currently support plugin installation."
		} else if len(supportedEngines) == 1 {
			supportedEnginesMsg = fmt.Sprintf("Only the '%s' engine supports plugin installation.", supportedEngines[0])
		} else {
			supportedEnginesMsg = "The following engines support plugin installation: " + strings.Join(supportedEngines, ", ")
		}

		return fmt.Errorf("engine '%s' does not support plugins. The following plugins cannot be installed: %s\n\n%s\n\nTo fix this, either:\n1. Remove the 'plugins' field from your workflow\n2. Change to an engine that supports plugins (e.g., engine: %s)\n\nSee: %s",
			agenticEngine.GetID(),
			pluginsList,
			supportedEnginesMsg,
			supportedEngines[0],
			constants.DocsEnginesURL)
	}

	engineValidationLog.Printf("Engine %s supports plugins: %d plugins to install", agenticEngine.GetID(), len(pluginInfo.Plugins))
	return nil
}
