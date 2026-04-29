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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
)

var engineValidationLog = newValidationLogger("engine")
var safeHarnessScriptPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

// validateEngineVersion warns (non-strict) or errors (strict) when the workflow
// explicitly pins the engine CLI to "latest". Unpinned "latest" versions change
// unpredictably and undermine supply chain security guarantees.
func (c *Compiler) validateEngineVersion(workflowData *WorkflowData) error {
	if workflowData.EngineConfig == nil || workflowData.EngineConfig.Version == "" {
		// No explicit version set; the compiler uses its own pinned default.
		return nil
	}

	if !strings.EqualFold(workflowData.EngineConfig.Version, "latest") {
		return nil
	}

	engineValidationLog.Print("engine.version: latest detected")

	warningMsg := "engine.version: latest is set – the engine CLI will be installed without a pinned version. " +
		"This is a supply chain security risk: unpinned 'latest' versions can change unexpectedly " +
		"and may introduce vulnerabilities or breaking changes. " +
		"Pin the engine version to a specific version for reproducibility and security."

	if c.strictMode {
		return fmt.Errorf("strict mode: %s", warningMsg)
	}

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
	c.IncrementWarningCount()
	return nil
}

// validateEngineHarnessScript validates optional engine.harness configuration.
// engine.harness must point to a Node.js script.
func (c *Compiler) validateEngineHarnessScript(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.HarnessScript == "" {
		return nil
	}

	harnessScript := workflowData.EngineConfig.HarnessScript
	if strings.TrimSpace(harnessScript) != harnessScript {
		return fmt.Errorf("engine.harness must be a safe basename without leading/trailing whitespace (found: %s).\n\nSee: %s", workflowData.EngineConfig.HarnessScript, constants.DocsEnginesURL)
	}

	if filepath.IsAbs(harnessScript) ||
		strings.Contains(harnessScript, "/") ||
		strings.Contains(harnessScript, `\`) ||
		strings.Contains(harnessScript, "..") ||
		!safeHarnessScriptPattern.MatchString(harnessScript) {
		return fmt.Errorf("engine.harness must be a safe basename (no path separators, '..', or shell metacharacters) ending with .js, .cjs, or .mjs (found: %s).\n\nSee: %s", workflowData.EngineConfig.HarnessScript, constants.DocsEnginesURL)
	}

	ext := strings.ToLower(filepath.Ext(harnessScript))
	switch ext {
	case ".js", ".cjs", ".mjs":
		return nil
	default:
		return fmt.Errorf("engine.harness must be a Node.js script ending with .js, .cjs, or .mjs (found: %s).\n\nSee: %s", workflowData.EngineConfig.HarnessScript, constants.DocsEnginesURL)
	}
}

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
		// Handle object format: either engine.id (named engine) or engine.runtime.id (inline definition)
		if id, hasID := engineObj["id"]; hasID {
			if idStr, ok := id.(string); ok {
				return idStr, nil
			}
		}
		// Handle inline definition with 'runtime' sub-object (engine.runtime.id)
		if runtime, hasRuntime := engineObj["runtime"]; hasRuntime {
			if runtimeObj, ok := runtime.(map[string]any); ok {
				if id, hasID := runtimeObj["id"]; hasID {
					if idStr, ok := id.(string); ok {
						return idStr, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("invalid engine configuration in included file, missing or invalid 'id' field. Expected string, object with 'id' field, or inline definition with 'runtime.id'.\n\nExample (string):\nengine: copilot\n\nExample (object with id):\nengine:\n  id: copilot\n  model: gpt-4\n\nExample (inline runtime definition):\nengine:\n  runtime:\n    id: codex\n\nSee: %s", constants.DocsEnginesURL)
}
