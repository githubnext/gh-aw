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
	if config.InlineProviderSecret != "" {
		def.Auth = []AuthBinding{{Role: "api-key", Secret: config.InlineProviderSecret}}
	}

	engineValidationLog.Printf("Registering inline engine definition in session catalog: id=%s, runtimeID=%s, providerID=%s",
		def.ID, def.RuntimeID, def.Provider.Name)
	c.engineCatalog.Register(def)
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
