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
	// Provide empathetic, educational error with valid options
	return fmt.Errorf("🤔 Hmm, we don't recognize the engine '%s'.\n\nValid options are:\n  • copilot (GitHub Copilot)\n  • claude (Anthropic Claude)\n  • codex (OpenAI Codex)\n  • custom (your own engine)\n\nExample:\n  engine: copilot\n\nNeed help choosing? See: https://githubnext.github.io/gh-aw/reference/engines/", engineID)
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
		return "", fmt.Errorf("⚠️  Multiple engine specifications detected (%d found).\n\nWhy this matters: Workflows can only use one AI engine at a time to ensure consistent behavior.\n\nPlease remove duplicate engine fields across your main workflow and included files, keeping only one.\n\nExample:\n  engine: copilot\n\nTip: Check both your main .md file and any included files for engine: settings", len(allEngines))
	}

	// Exactly one engine found - parse and return it
	if mainEngineSetting != "" {
		return mainEngineSetting, nil
	}

	// Must be from included file
	var firstEngine any
	if err := json.Unmarshal([]byte(includedEnginesJSON[0]), &firstEngine); err != nil {
		return "", fmt.Errorf("💡 Having trouble parsing the engine configuration from your included file.\n\nEngine configs can be either a simple string or an object format:\n\nString format:\n  engine: copilot\n\nObject format:\n  engine:\n    id: copilot\n    model: gpt-4\n\nError details: %w", err)
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

	return "", fmt.Errorf("💡 The engine configuration in your included file needs an 'id' field.\n\nYou can use either format:\n\nString format (simple):\n  engine: copilot\n\nObject format (with options):\n  engine:\n    id: copilot\n    model: gpt-4\n\nThe 'id' field tells us which AI engine to use")
}
