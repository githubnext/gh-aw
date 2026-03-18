// This file contains strict mode network validation functions.
//
// It validates network configuration, MCP network requirements, and tools
// configuration for workflows compiled with the --strict flag.

package workflow

import (
	"errors"
	"fmt"
	"slices"
)

// validateStrictNetwork validates network configuration in strict mode and refuses "*" wildcard
// Note: networkPermissions should never be nil at this point because the compiler orchestrator
// applies defaults (Allowed: ["defaults"]) when no network configuration is specified in frontmatter.
// This automatic default application means users don't need to explicitly declare network in strict mode.
func (c *Compiler) validateStrictNetwork(networkPermissions *NetworkPermissions) error {
	// This check should never trigger in production since the compiler orchestrator
	// always applies defaults before calling validation. However, we keep it for defensive programming
	// and to handle direct unit test calls.
	if networkPermissions == nil {
		strictModeValidationLog.Printf("Network configuration unexpectedly nil (defaults should have been applied)")
		return errors.New("internal error: network permissions not initialized (this should not happen in normal operation)")
	}

	// If allowed list contains "defaults", that's acceptable (this is the automatic default)
	if slices.Contains(networkPermissions.Allowed, "defaults") {
		strictModeValidationLog.Printf("Network validation passed: allowed list contains 'defaults'")
		return nil
	}

	// Check for wildcard "*" in allowed domains
	if slices.Contains(networkPermissions.Allowed, "*") {
		strictModeValidationLog.Printf("Network validation failed: wildcard detected")
		return errors.New("strict mode: wildcard '*' is not allowed in network.allowed domains to prevent unrestricted internet access. Specify explicit domains or use ecosystem identifiers like 'python', 'node', 'containers'. See: https://github.github.com/gh-aw/reference/network/#available-ecosystem-identifiers")
	}

	strictModeValidationLog.Printf("Network validation passed: allowed_count=%d", len(networkPermissions.Allowed))
	return nil
}

// validateStrictMCPNetwork requires top-level network configuration when custom MCP servers use containers
func (c *Compiler) validateStrictMCPNetwork(frontmatter map[string]any, networkPermissions *NetworkPermissions) error {
	// Check mcp-servers section (new format)
	mcpServersValue, exists := frontmatter["mcp-servers"]
	if !exists {
		return nil
	}

	mcpServersMap, ok := mcpServersValue.(map[string]any)
	if !ok {
		return nil
	}

	// Check if top-level network configuration exists
	hasTopLevelNetwork := networkPermissions != nil && len(networkPermissions.Allowed) > 0

	// Check each MCP server for containers
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
				// Require top-level network configuration
				if !hasTopLevelNetwork {
					return fmt.Errorf("strict mode: custom MCP server '%s' with container must have top-level network configuration for security. Add 'network: { allowed: [...] }' to the workflow to restrict network access. See: https://github.github.com/gh-aw/reference/network/", serverName)
				}
			}
		}
	}

	return nil
}

// validateStrictTools validates tools configuration in strict mode
func (c *Compiler) validateStrictTools(frontmatter map[string]any) error {
	// Check tools section
	toolsValue, exists := frontmatter["tools"]
	if !exists {
		return nil
	}

	toolsMap, ok := toolsValue.(map[string]any)
	if !ok {
		return nil
	}

	// Check if cache-memory is configured with scope: repo
	cacheMemoryValue, hasCacheMemory := toolsMap["cache-memory"]
	if hasCacheMemory {
		// Helper function to check scope in a cache entry
		checkScope := func(cacheMap map[string]any) error {
			if scope, hasScope := cacheMap["scope"]; hasScope {
				if scopeStr, ok := scope.(string); ok && scopeStr == "repo" {
					strictModeValidationLog.Printf("Cache-memory repo scope validation failed")
					return errors.New("strict mode: cache-memory with 'scope: repo' is not allowed for security reasons. Repo scope allows cache sharing across all workflows in the repository, which can enable cross-workflow cache poisoning attacks. Use 'scope: workflow' (default) instead, which isolates caches to individual workflows. See: https://github.github.com/gh-aw/reference/tools/#cache-memory")
				}
			}
			return nil
		}

		// Check if cache-memory is a map (object notation)
		if cacheMemoryConfig, ok := cacheMemoryValue.(map[string]any); ok {
			if err := checkScope(cacheMemoryConfig); err != nil {
				return err
			}
		}

		// Check if cache-memory is an array (array notation)
		if cacheMemoryArray, ok := cacheMemoryValue.([]any); ok {
			for _, item := range cacheMemoryArray {
				if cacheMap, ok := item.(map[string]any); ok {
					if err := checkScope(cacheMap); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
