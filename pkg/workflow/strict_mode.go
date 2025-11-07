// Package workflow provides strict mode security validation for agentic workflows.
//
// # Strict Mode Validation
//
// This file contains the main orchestrator for strict mode validation. Individual
// validation functions are implemented in validation_strict_mode.go.
//
// Strict mode is designed for production workflows that require enhanced security
// guarantees. It enforces constraints on:
//   - Write permissions on sensitive scopes
//   - Network access configuration
//   - Custom MCP server network settings
//   - Bash wildcard tool usage
//
// # Integration with Security Scanners
//
// Strict mode also affects the zizmor security scanner behavior (see pkg/cli/zizmor.go).
// When zizmor is enabled with --zizmor flag, strict mode treats any security findings
// as compilation errors rather than warnings.
//
// # Architecture
//
// The strict mode validation is split across two files:
//   - strict_mode.go (this file) - Main orchestrator function
//   - validation_strict_mode.go - Individual validation functions
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

// validateStrictMode performs strict mode validations on the workflow
//
// This is the main orchestrator that calls individual validation functions
// defined in validation_strict_mode.go. It performs progressive validation:
//  1. validateStrictPermissions() - Refuses write permissions on sensitive scopes
//  2. validateStrictNetwork() - Requires explicit network configuration
//  3. validateStrictMCPNetwork() - Requires network config on custom MCP servers
//
// Note: Strict mode also affects zizmor security scanner behavior (see pkg/cli/zizmor.go)
// When zizmor is enabled with --zizmor flag, strict mode will treat any security
// findings as compilation errors rather than warnings.
func (c *Compiler) validateStrictMode(frontmatter map[string]any, networkPermissions *NetworkPermissions) error {
	if !c.strictMode {
		return nil
	}

	// 1. Refuse write permissions
	if err := c.validateStrictPermissions(frontmatter); err != nil {
		return err
	}

	// 2. Require network configuration and refuse "*" wildcard
	if err := c.validateStrictNetwork(networkPermissions); err != nil {
		return err
	}

	// 3. Require network configuration on custom MCP servers
	if err := c.validateStrictMCPNetwork(frontmatter); err != nil {
		return err
	}

	return nil
}
