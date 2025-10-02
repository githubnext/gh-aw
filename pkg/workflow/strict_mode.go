package workflow

import (
	"fmt"
)

// validateStrictMode performs strict mode validations on the workflow
func (c *Compiler) validateStrictMode(frontmatter map[string]any, networkPermissions *NetworkPermissions) error {
	if !c.strictMode {
		return nil
	}

	// 1. Require timeout_minutes
	if err := c.validateStrictTimeout(frontmatter); err != nil {
		return err
	}

	// 2. Refuse write permissions
	if err := c.validateStrictPermissions(frontmatter); err != nil {
		return err
	}

	// 3. Require network configuration and refuse "*" wildcard
	if err := c.validateStrictNetwork(networkPermissions); err != nil {
		return err
	}

	// 4. Require network configuration on custom MCP servers
	if err := c.validateStrictMCPNetwork(frontmatter); err != nil {
		return err
	}

	return nil
}

// validateStrictTimeout ensures timeout_minutes is specified in strict mode
func (c *Compiler) validateStrictTimeout(frontmatter map[string]any) error {
	timeoutValue, exists := frontmatter["timeout_minutes"]
	if !exists {
		return fmt.Errorf("strict mode: 'timeout_minutes' is required in workflow frontmatter")
	}

	// Validate it's a positive integer (handle various numeric types)
	switch v := timeoutValue.(type) {
	case int:
		if v <= 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %d", v)
		}
	case int32:
		if v <= 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %d", v)
		}
	case int64:
		if v <= 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %d", v)
		}
	case uint:
		if v == 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %d", v)
		}
	case uint32:
		if v == 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %d", v)
		}
	case uint64:
		if v == 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %d", v)
		}
	case float64:
		if v <= 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %f", v)
		}
	case float32:
		if v <= 0 {
			return fmt.Errorf("strict mode: 'timeout_minutes' must be a positive integer, got %f", v)
		}
	default:
		return fmt.Errorf("strict mode: 'timeout_minutes' must be an integer, got type %T", timeoutValue)
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

	// Parse permissions using the helper
	perms := ParsePermissions(permissionsValue)

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
