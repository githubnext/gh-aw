package workflow

import (
	"fmt"
)

// validateStrictMode performs strict mode validations on the workflow
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
