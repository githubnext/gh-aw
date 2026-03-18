// This file provides the strict mode validation orchestrator for agentic workflows.
//
// # Strict Mode Validation
//
// Strict mode is designed for production workflows that require enhanced security
// guarantees. The validation logic is split across focused files:
//   - strict_mode_permissions_validation.go: permissions, deprecated fields, firewall
//   - strict_mode_network_validation.go: network, MCP network, tools checks
//   - strict_mode_env_validation.go: environment secrets validation
//
// # Integration with Security Scanners
//
// Strict mode also affects the zizmor security scanner behavior (see pkg/cli/zizmor.go).
// When zizmor is enabled with --zizmor flag, strict mode treats any security findings
// as compilation errors rather than warnings.
//
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

var strictModeValidationLog = newValidationLogger("strict_mode")

// validateStrictMode performs strict mode validations on the workflow
//
// This is the main orchestrator that calls individual validation functions.
// It performs progressive validation:
//  1. validateStrictPermissions() - Refuses write permissions on sensitive scopes
//  2. validateStrictNetwork() - Requires explicit network configuration
//  3. validateStrictMCPNetwork() - Requires top-level network config for container-based MCP servers
//  4. validateStrictTools() - Validates tools configuration (e.g., serena local mode)
//  5. validateStrictDeprecatedFields() - Refuses deprecated fields
//
// Note: Env secrets validation (validateEnvSecrets) is called separately outside of strict mode
// to emit warnings in non-strict mode and errors in strict mode.
//
// Note: Strict mode also affects zizmor security scanner behavior (see pkg/cli/zizmor.go)
// When zizmor is enabled with --zizmor flag, strict mode will treat any security
// findings as compilation errors rather than warnings.
func (c *Compiler) validateStrictMode(frontmatter map[string]any, networkPermissions *NetworkPermissions) error {
	if !c.strictMode {
		strictModeValidationLog.Printf("Strict mode disabled, skipping validation")
		return nil
	}

	strictModeValidationLog.Printf("Starting strict mode validation")

	// Collect all strict mode validation errors
	collector := NewErrorCollector(c.failFast)

	// 1. Refuse write permissions
	if err := c.validateStrictPermissions(frontmatter); err != nil {
		if returnErr := collector.Add(err); returnErr != nil {
			return returnErr // Fail-fast mode
		}
	}

	// 2. Require network configuration and refuse "*" wildcard
	if err := c.validateStrictNetwork(networkPermissions); err != nil {
		if returnErr := collector.Add(err); returnErr != nil {
			return returnErr // Fail-fast mode
		}
	}

	// 3. Require network configuration on custom MCP servers
	if err := c.validateStrictMCPNetwork(frontmatter, networkPermissions); err != nil {
		if returnErr := collector.Add(err); returnErr != nil {
			return returnErr // Fail-fast mode
		}
	}

	// 4. Validate tools configuration
	if err := c.validateStrictTools(frontmatter); err != nil {
		if returnErr := collector.Add(err); returnErr != nil {
			return returnErr // Fail-fast mode
		}
	}

	// 5. Refuse deprecated fields
	if err := c.validateStrictDeprecatedFields(frontmatter); err != nil {
		if returnErr := collector.Add(err); returnErr != nil {
			return returnErr // Fail-fast mode
		}
	}

	strictModeValidationLog.Printf("Strict mode validation completed: error_count=%d", collector.Count())

	return collector.FormattedError("strict mode")
}
