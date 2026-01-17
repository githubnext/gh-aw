// Package workflow provides engine validation for agentic workflows.
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
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var engineValidationLog = logger.New("workflow:engine_validation")

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
	// Provide helpful error with valid options
	return fmt.Errorf("❌ Invalid engine: %s\n\n💡 What went wrong: The engine '%s' is not recognized.\n\n✅ How to fix: Choose one of these supported engines:\n  - copilot (GitHub Copilot - recommended)\n  - claude (Anthropic Claude)\n  - codex (OpenAI Codex)\n  - custom (Custom engine configuration)\n\nExample:\n---\nengine: copilot\n---\n\n📚 Learn more: https://githubnext.github.io/gh-aw/reference/frontmatter/#engine", engineID, engineID)
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
		return "", fmt.Errorf("❌ Multiple engine specifications found\n\n💡 What went wrong: Found %d engine specifications. Only one engine can be specified across the main workflow and all included files.\n\n✅ How to fix: Remove duplicate engine specifications to keep only one.\n\nExample (in main workflow or one include file):\n---\nengine: copilot\n---\n\n📚 Learn more: https://githubnext.github.io/gh-aw/reference/frontmatter/#engine", len(allEngines))
	}

	// Exactly one engine found - parse and return it
	if mainEngineSetting != "" {
		return mainEngineSetting, nil
	}

	// Must be from included file
	var firstEngine any
	if err := json.Unmarshal([]byte(includedEnginesJSON[0]), &firstEngine); err != nil {
		return "", fmt.Errorf("❌ Failed to parse engine configuration\n\n💡 What went wrong: %v\n\n✅ How to fix: Use either string format or object format for engine configuration.\n\nExample (string format):\n---\nengine: copilot\n---\n\nExample (object format):\n---\nengine:\n  id: copilot\n  model: gpt-4\n---\n\n📚 Learn more: https://githubnext.github.io/gh-aw/reference/frontmatter/#engine", err)
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

	return "", fmt.Errorf("❌ Invalid engine configuration\n\n💡 What went wrong: Engine configuration is missing or has an invalid 'id' field.\n\n✅ How to fix: Use either string format or object format with an 'id' field.\n\nExample (string format):\n---\nengine: copilot\n---\n\nExample (object format):\n---\nengine:\n  id: copilot\n  model: gpt-4\n---\n\n📚 Learn more: https://githubnext.github.io/gh-aw/reference/frontmatter/#engine")
}
