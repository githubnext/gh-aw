// Package workflow provides strict mode security validation for agentic workflows.
//
// # Strict Mode Validation
//
// This file contains strict mode validation functions that enforce security
// and safety constraints when workflows are compiled with the --strict flag.
//
// Strict mode is designed for production workflows that require enhanced security
// guarantees. It enforces constraints on:
//   - Write permissions on sensitive scopes
//   - Network access configuration
//   - Custom MCP server network settings
//   - Bash wildcard tool usage
//
// # Validation Functions
//
// The strict mode validator performs progressive validation:
//  1. validateStrictMode() - Main orchestrator that coordinates all strict mode checks
//  2. validateStrictPermissions() - Refuses write permissions on sensitive scopes
//  3. validateStrictNetwork() - Requires explicit network configuration
//  4. validateStrictMCPNetwork() - Requires network config on custom MCP servers
//
// # Integration with Security Scanners
//
// Strict mode also affects the zizmor security scanner behavior (see pkg/cli/zizmor.go).
// When zizmor is enabled with --zizmor flag, strict mode treats any security findings
// as compilation errors rather than warnings.
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It enforces a strict mode security policy
//   - It restricts permissions or access in production workflows
//   - It validates network access controls
//   - It enforces tool usage restrictions for security
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var strictModeValidationLog = logger.New("workflow:strict_mode_validation")

// validateStrictPermissions refuses write permissions in strict mode
func (c *Compiler) validateStrictPermissions(frontmatter map[string]any) error {
	permissionsValue, exists := frontmatter["permissions"]
	if !exists {
		// No permissions specified is fine
		strictModeValidationLog.Printf("No permissions specified, validation passed")
		return nil
	}

	// Parse permissions using the PermissionsParser
	perms := NewPermissionsParserFromValue(permissionsValue)

	// Check for write permissions on sensitive scopes
	writePermissions := []string{"contents", "issues", "pull-requests"}
	for _, scope := range writePermissions {
		if perms.IsAllowed(scope, "write") {
			strictModeValidationLog.Printf("Write permission validation failed: scope=%s", scope)
			return fmt.Errorf("strict mode: write permission '%s: write' is not allowed for security reasons. Use 'safe-outputs.create-issue', 'safe-outputs.create-pull-request', 'safe-outputs.add-comment', or 'safe-outputs.update-issue' to perform write operations safely. See: https://githubnext.github.io/gh-aw/reference/safe-outputs/", scope)
		}
	}

	strictModeValidationLog.Printf("Permissions validation passed")
	return nil
}

// validateStrictNetwork requires network configuration and refuses "*" wildcard
func (c *Compiler) validateStrictNetwork(networkPermissions *NetworkPermissions) error {
	if networkPermissions == nil {
		strictModeValidationLog.Printf("Network configuration missing")
		return fmt.Errorf("strict mode: 'network' configuration is required to prevent unrestricted network access. Add 'network: { allowed: [...] }' or 'network: defaults' to your frontmatter. See: https://githubnext.github.io/gh-aw/reference/network/")
	}

	// If mode is "defaults", that's acceptable
	if networkPermissions.Mode == "defaults" {
		strictModeValidationLog.Printf("Network validation passed: mode=defaults")
		return nil
	}

	// Check for wildcard "*" in allowed domains
	for _, domain := range networkPermissions.Allowed {
		if domain == "*" {
			strictModeValidationLog.Printf("Network validation failed: wildcard detected")
			return fmt.Errorf("strict mode: wildcard '*' is not allowed in network.allowed domains to prevent unrestricted internet access. Specify explicit domains or use ecosystem identifiers like 'python', 'node', 'containers'. See: https://githubnext.github.io/gh-aw/reference/network/#available-ecosystem-identifiers")
		}
	}

	strictModeValidationLog.Printf("Network validation passed: allowed_count=%d", len(networkPermissions.Allowed))
	return nil
}

// validateStrictMCPNetwork requires network configuration on custom MCP servers
func (c *Compiler) validateStrictMCPNetwork(frontmatter map[string]any) error {
	// Check mcp-servers section (new format)
	mcpServersValue, exists := frontmatter["mcp-servers"]
	if !exists {
		return nil
	}

	mcpServersMap, ok := mcpServersValue.(map[string]any)
	if !ok {
		return nil
	}

	// Check each MCP server for network configuration
	for serverName, serverValue := range mcpServersMap {
		serverConfig, ok := serverValue.(map[string]any)
		if !ok {
			continue
		}

		// Use helper function to determine if this is an MCP config and its type
		hasMCP, mcpType := hasMCPConfig(serverConfig)
		if !hasMCP {
			continue
		}

		// Only stdio servers with containers need network configuration
		if mcpType == "stdio" {
			if _, hasContainer := serverConfig["container"]; hasContainer {
				// Check if network configuration is present
				if _, hasNetwork := serverConfig["network"]; !hasNetwork {
					return fmt.Errorf("strict mode: custom MCP server '%s' with container must have network configuration for security. Add 'network: { allowed: [...] }' to the server configuration to restrict network access. See: https://githubnext.github.io/gh-aw/reference/network/", serverName)
				}
			}
		}
	}

	return nil
}

// validateStrictMode performs strict mode validations on the workflow
//
// This is the main orchestrator that calls individual validation functions.
// It performs progressive validation:
//  1. validateStrictPermissions() - Refuses write permissions on sensitive scopes
//  2. validateStrictNetwork() - Requires explicit network configuration
//  3. validateStrictMCPNetwork() - Requires network config on custom MCP servers
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

	strictModeValidationLog.Printf("Strict mode validation completed successfully")
	return nil
}
