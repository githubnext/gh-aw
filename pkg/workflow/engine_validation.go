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
//   - validateCopilotNetworkConfig() - Validates Copilot workflows use GitHub MCP instead of direct api.github.com access
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
	"strings"

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
	return fmt.Errorf("invalid engine: %s. Valid engines are: copilot, claude, codex, custom. Example: engine: copilot", engineID)
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
		return "", fmt.Errorf("multiple engine fields found (%d engine specifications detected). Only one engine field is allowed across the main workflow and all included files. Remove duplicate engine specifications to keep only one. Example: engine: copilot", len(allEngines))
	}

	// Exactly one engine found - parse and return it
	if mainEngineSetting != "" {
		return mainEngineSetting, nil
	}

	// Must be from included file
	var firstEngine any
	if err := json.Unmarshal([]byte(includedEnginesJSON[0]), &firstEngine); err != nil {
		return "", fmt.Errorf("failed to parse included engine configuration: %w. Expected string or object format. Example (string): engine: copilot or (object): engine:\\n  id: copilot\\n  model: gpt-4", err)
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

	return "", fmt.Errorf("invalid engine configuration in included file, missing or invalid 'id' field. Expected string or object with 'id' field. Example (string): engine: copilot or (object): engine:\\n  id: copilot\\n  model: gpt-4")
}

// validateCopilotNetworkConfig validates that Copilot workflows use GitHub MCP server
// instead of attempting direct api.github.com access
//
// The Copilot agent cannot directly access api.github.com due to network restrictions.
// All GitHub API operations must go through the GitHub MCP server.
func (c *Compiler) validateCopilotNetworkConfig(engineID string, networkPermissions *NetworkPermissions, tools *Tools) error {
	// Only validate for Copilot engine
	if engineID != "copilot" {
		return nil
	}

	engineValidationLog.Printf("Validating Copilot network configuration")

	// Check if api.github.com is in the network allowed list
	if networkPermissions != nil && len(networkPermissions.Allowed) > 0 {
		for _, domain := range networkPermissions.Allowed {
			if domain == "api.github.com" {
				engineValidationLog.Printf("Found api.github.com in network allowed list for Copilot workflow")

				// Check if GitHub MCP is configured
				hasGitHubMCP := tools != nil && tools.GitHub != nil

				// Build error message
				var errorMsg strings.Builder
				errorMsg.WriteString("Copilot workflows cannot directly access api.github.com. ")
				errorMsg.WriteString("The Copilot agent requires the GitHub MCP server for GitHub API operations.\n\n")

				if hasGitHubMCP {
					errorMsg.WriteString("You have GitHub MCP configured, so you can remove 'api.github.com' from your network.allowed list.\n\n")
				} else {
					errorMsg.WriteString("Configure the GitHub MCP server instead:\n\n")
					errorMsg.WriteString("Option 1 - Remote mode (hosted):\n")
					errorMsg.WriteString("  tools:\n")
					errorMsg.WriteString("    github:\n")
					errorMsg.WriteString("      mode: remote\n")
					errorMsg.WriteString("      toolsets: [default]\n\n")
					errorMsg.WriteString("Option 2 - Local mode (Docker):\n")
					errorMsg.WriteString("  tools:\n")
					errorMsg.WriteString("    github:\n")
					errorMsg.WriteString("      mode: local\n")
					errorMsg.WriteString("      toolsets: [default]\n\n")
				}

				errorMsg.WriteString("Then remove 'api.github.com' from your network.allowed list.\n\n")
				errorMsg.WriteString("See: https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default")

				return fmt.Errorf("%s", errorMsg.String())
			}
		}
	}

	engineValidationLog.Printf("Copilot network configuration validation passed")
	return nil
}
