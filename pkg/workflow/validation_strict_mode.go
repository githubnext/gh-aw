// Package workflow provides strict mode security validation for agentic workflows.
//
// # Strict Mode Validation Functions
//
// This file contains the individual validation functions that enforce security
// and safety constraints when workflows are compiled with the --strict flag.
// These functions are called by validateStrictMode() in strict_mode.go.
//
// # Validation Functions
//
// The strict mode validator performs progressive validation:
//  1. validateStrictPermissions() - Refuses write permissions on sensitive scopes
//  2. validateStrictNetwork() - Requires explicit network configuration
//  3. validateStrictMCPNetwork() - Requires network config on custom MCP servers
//  4. validateStrictBashTools() - Refuses bash wildcard tools ("*" and ":*")
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
// For the main strict mode orchestrator, see strict_mode.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
)

// validateStrictPermissions refuses write permissions in strict mode
func (c *Compiler) validateStrictPermissions(frontmatter map[string]any) error {
	permissionsValue, exists := frontmatter["permissions"]
	if !exists {
		// No permissions specified is fine
		return nil
	}

	// Parse permissions using the PermissionsParser
	perms := NewPermissionsParserFromValue(permissionsValue)

	// Check for write permissions on sensitive scopes
	writePermissions := []string{"contents", "issues", "pull-requests"}
	for _, scope := range writePermissions {
		if perms.IsAllowed(scope, "write") {
			return fmt.Errorf("strict mode: write permission '%s: write' is not allowed", scope)
		}
	}

	return nil
}

// validateStrictNetwork requires network configuration and refuses "*" wildcard
func (c *Compiler) validateStrictNetwork(networkPermissions *NetworkPermissions) error {
	if networkPermissions == nil {
		return fmt.Errorf("strict mode: 'network' configuration is required")
	}

	// If mode is "defaults", that's acceptable
	if networkPermissions.Mode == "defaults" {
		return nil
	}

	// Check for wildcard "*" in allowed domains
	for _, domain := range networkPermissions.Allowed {
		if domain == "*" {
			return fmt.Errorf("strict mode: wildcard '*' is not allowed in network.allowed domains")
		}
	}

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
					return fmt.Errorf("strict mode: custom MCP server '%s' with container must have network configuration", serverName)
				}
			}
		}
	}

	return nil
}

// validateStrictBashTools refuses bash wildcard tools ("*" and ":*")
func (c *Compiler) validateStrictBashTools(frontmatter map[string]any) error {
	// Check tools section
	toolsValue, exists := frontmatter["tools"]
	if !exists {
		return nil
	}

	toolsMap, ok := toolsValue.(map[string]any)
	if !ok {
		return nil
	}

	// Check bash tool for wildcards
	bashValue, hasBash := toolsMap["bash"]
	if !hasBash {
		return nil
	}

	// Check if bash is an array of commands
	bashCommands, ok := bashValue.([]any)
	if !ok {
		// If bash is not an array (e.g., true, null, or object), it's allowed in strict mode
		return nil
	}

	// Check for wildcard patterns in bash commands
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}

		// Refuse "*" and ":*" wildcards
		if cmdStr == "*" || cmdStr == ":*" {
			return fmt.Errorf("strict mode: bash wildcard '%s' is not allowed - use specific commands instead", cmdStr)
		}
	}

	return nil
}
