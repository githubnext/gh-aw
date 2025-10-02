package workflow

import (
	"fmt"
	"strings"
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

	// Handle permissions as a map
	permissionsMap, ok := permissionsValue.(map[string]any)
	if !ok {
		// If it's not a map, it might be a string like "read-all" which is fine
		return nil
	}

	// Check for write permissions
	writePermissions := []string{"contents", "issues", "pull-requests"}
	for _, perm := range writePermissions {
		if value, exists := permissionsMap[perm]; exists {
			if valueStr, ok := value.(string); ok {
				if valueStr == "write" {
					return fmt.Errorf("strict mode: write permission '%s: write' is not allowed", perm)
				}
			}
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
	// Check tools section for MCP servers
	toolsValue, exists := frontmatter["tools"]
	if !exists {
		return nil
	}

	toolsMap, ok := toolsValue.(map[string]any)
	if !ok {
		return nil
	}

	// Check each tool for custom MCP configuration
	for toolName, toolValue := range toolsMap {
		// Skip built-in tools
		if isBuiltInTool(toolName) {
			continue
		}

		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		// Check if it has MCP configuration
		mcpConfig, hasMCP := toolConfig["mcp"]
		if !hasMCP {
			continue
		}

		mcpMap, ok := mcpConfig.(map[string]any)
		if !ok {
			continue
		}

		// Check if it's a containerized MCP server (stdio with container)
		mcpType := ""
		if typeValue, hasType := mcpMap["type"]; hasType {
			if typeStr, ok := typeValue.(string); ok {
				mcpType = typeStr
			}
		} else {
			// Infer type
			if _, hasContainer := mcpMap["container"]; hasContainer {
				mcpType = "stdio"
			} else if _, hasCommand := mcpMap["command"]; hasCommand {
				mcpType = "stdio"
			} else if _, hasURL := mcpMap["url"]; hasURL {
				mcpType = "http"
			}
		}

		// Normalize "local" to "stdio"
		if mcpType == "local" {
			mcpType = "stdio"
		}

		// Only stdio servers with containers need network configuration
		if mcpType == "stdio" {
			if _, hasContainer := mcpMap["container"]; hasContainer {
				// Check if network configuration is present
				hasNetPerms := false
				
				// Check in toolConfig
				if _, hasNetwork := toolConfig["network"]; hasNetwork {
					hasNetPerms = true
				}
				
				// Check in mcpMap
				if _, hasNetwork := mcpMap["network"]; hasNetwork {
					hasNetPerms = true
				}

				if !hasNetPerms {
					return fmt.Errorf("strict mode: custom MCP server '%s' with container must have network configuration", toolName)
				}
			}
		}
	}

	return nil
}

// isBuiltInTool checks if a tool is a built-in tool
func isBuiltInTool(toolName string) bool {
	builtInTools := []string{
		"github",
		"edit",
		"web-fetch",
		"web-search",
		"bash",
		"playwright",
		"cache-memory",
	}

	for _, builtIn := range builtInTools {
		if strings.EqualFold(toolName, builtIn) {
			return true
		}
	}

	return false
}
