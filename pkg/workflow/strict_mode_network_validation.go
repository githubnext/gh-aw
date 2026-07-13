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
		return NewValidationError(
			"network.allowed",
			"*",
			"strict mode: wildcard '*' is not allowed in network.allowed domains to prevent unrestricted internet access; expected explicit domains or ecosystem identifiers",
			"Use specific domains or supported ecosystem identifiers:\n\nnetwork:\n  allowed:\n    - github.com\n    - api.github.com\n    - python",
		)
	}

	strictModeValidationLog.Printf("Network validation passed: allowed_count=%d", len(networkPermissions.Allowed))
	return nil
}

// validateStrictMCPNetwork requires top-level network configuration when custom MCP servers use containers
func (c *Compiler) validateStrictMCPNetwork(frontmatter map[string]any, networkPermissions *NetworkPermissions) error {
	// Check mcp-servers section (new format)
	mcpServersValue, exists := frontmatter["mcp-servers"]
	if !exists {
		strictModeValidationLog.Print("No mcp-servers section, skipping MCP network validation")
		return nil
	}

	mcpServersMap, ok := mcpServersValue.(map[string]any)
	if !ok {
		strictModeValidationLog.Print("mcp-servers is not a map, skipping MCP network validation")
		return nil
	}

	// Check if top-level network configuration exists
	hasTopLevelNetwork := networkPermissions != nil && len(networkPermissions.Allowed) > 0
	strictModeValidationLog.Printf("Checking %d MCP servers for container network requirements: hasTopLevelNetwork=%t", len(mcpServersMap), hasTopLevelNetwork)

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
					return NewValidationError(
						"network.allowed",
						serverName,
						fmt.Sprintf("strict mode: custom MCP server '%s' with container must have top-level network configuration for security; expected top-level network restrictions for container-based MCP servers", serverName),
						fmt.Sprintf("Add explicit top-level network permissions for mcp-servers.%s:\n\nnetwork:\n  allowed:\n    - github.com\n    - api.github.com\n\nmcp-servers:\n  %s:\n    type: stdio\n    container: ghcr.io/example/mcp-server:latest", serverName, serverName),
					)
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
		strictModeValidationLog.Print("No tools section, skipping strict tools validation")
		return nil
	}

	toolsMap, ok := toolsValue.(map[string]any)
	if !ok {
		strictModeValidationLog.Print("tools is not a map, skipping strict tools validation")
		return nil
	}

	// Reject private-to-public-flows: allow in strict mode.
	// Per MCP Gateway Specification Section 10.9.4, the blanket "allow" value is incompatible
	// with strict mode because it disables both forcePublicRepos and sink-visibility enforcement.
	// The list form (specific server IDs) is allowed in strict mode.
	if githubValue, hasGitHub := toolsMap["github"]; hasGitHub {
		if githubMap, ok := githubValue.(map[string]any); ok {
			if ptpFlows, exists := githubMap["private-to-public-flows"]; exists {
				if ptpStr, ok := ptpFlows.(string); ok && ptpStr == "allow" {
					strictModeValidationLog.Printf("private-to-public-flows: allow rejected in strict mode")
					return NewValidationError(
						"tools.github.private-to-public-flows",
						ptpStr,
						"strict mode: 'private-to-public-flows: allow' is not allowed; it disables forcePublicRepos and sink-visibility enforcement, which is incompatible with strict mode",
						"To exempt specific MCP servers from sink-visibility enforcement in strict mode, use the list form:\n\ntools:\n  github:\n    private-to-public-flows:\n      - my-server-id\n      - other-server-id",
					)
				}
			}
		}
	}

	// Check if cache-memory is configured with scope: repo
	cacheMemoryValue, hasCacheMemory := toolsMap["cache-memory"]
	if hasCacheMemory {
		strictModeValidationLog.Print("Checking cache-memory scope in strict mode")
		// Helper function to check scope in a cache entry
		checkScope := func(cacheMap map[string]any) error {
			if scope, hasScope := cacheMap["scope"]; hasScope {
				if scopeStr, ok := scope.(string); ok && scopeStr == "repo" {
					strictModeValidationLog.Printf("Cache-memory repo scope validation failed")
					return NewValidationError(
						"tools.cache-memory.scope",
						scopeStr,
						"strict mode: cache-memory with 'scope: repo' is not allowed for security reasons; expected 'scope: workflow' to isolate cache data per workflow",
						"Use workflow-scoped cache entries:\n\ntools:\n  cache-memory:\n    key: my-cache\n    scope: workflow",
					)
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

// validatePrivateToPublicFlowsStringValue validates that the string form of
// private-to-public-flows uses the only supported value: "allow".
func validatePrivateToPublicFlowsStringValue(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.ParsedTools == nil || workflowData.ParsedTools.GitHub == nil {
		return nil
	}
	value, ok := workflowData.ParsedTools.GitHub.PrivateToPublicFlows.(string)
	if !ok || value == "" || value == "allow" {
		return nil
	}

	return NewValidationError(
		"tools.github.private-to-public-flows",
		value,
		fmt.Sprintf("invalid value %q; expected \"allow\" or a list of MCP server IDs", value),
		"Set tools.github.private-to-public-flows to \"allow\" for blanket opt-in, or use a list of MCP server IDs for targeted exemptions.",
	)
}

// validatePrivateToPublicFlowsServerIDs validates that every server ID in the list form of
// private-to-public-flows matches a declared MCP server in the workflow's tools list.
// Per MCP Gateway Specification Section 10.9.2, unknown IDs must be rejected at compile time.
func validatePrivateToPublicFlowsServerIDs(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.ParsedTools == nil || workflowData.ParsedTools.GitHub == nil {
		return nil
	}

	servers, ok := workflowData.ParsedTools.GitHub.PrivateToPublicFlows.([]string)
	if !ok || len(servers) == 0 {
		return nil
	}
	// Collect valid server IDs from the merged tools map.
	validIDs := make(map[string]bool, len(workflowData.Tools))
	for id := range workflowData.Tools {
		validIDs[id] = true
	}
	var unknown []string
	for _, id := range servers {
		if !validIDs[id] {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	return NewValidationError(
		"tools.github.private-to-public-flows",
		fmt.Sprintf("%v", unknown),
		fmt.Sprintf("unknown MCP server ID(s) %v; every ID in private-to-public-flows must match a server declared in tools or mcp-servers", unknown),
		"Check that each server ID in private-to-public-flows matches an entry in tools or mcp-servers. "+
			"The built-in GitHub MCP server ID is \"github\". Custom servers use the key from mcp-servers.",
	)
}
