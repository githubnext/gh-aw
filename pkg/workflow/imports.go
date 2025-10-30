package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var importsLog = logger.New("workflow:imports")

// MergeTools merges two tools maps, combining allowed arrays when keys coincide
// Handles newline-separated JSON objects from multiple imports/includes
func (c *Compiler) MergeTools(topTools map[string]any, includedToolsJSON string) (map[string]any, error) {
	importsLog.Print("Merging tools from imports")

	if includedToolsJSON == "" || includedToolsJSON == "{}" {
		importsLog.Print("No included tools to merge")
		return topTools, nil
	}

	// Split by newlines to handle multiple JSON objects from different imports/includes
	lines := strings.Split(includedToolsJSON, "\n")
	result := topTools
	if result == nil {
		result = make(map[string]any)
	}

	importsLog.Printf("Processing %d tool definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		var includedTools map[string]any
		if err := json.Unmarshal([]byte(line), &includedTools); err != nil {
			continue // Skip invalid lines
		}

		// Merge this set of tools
		merged, err := parser.MergeTools(result, includedTools)
		if err != nil {
			return nil, fmt.Errorf("failed to merge tools: %w", err)
		}
		result = merged
	}

	return result, nil
}

// MergeMCPServers merges mcp-servers from imports with top-level mcp-servers
// Takes object maps and merges them directly
func (c *Compiler) MergeMCPServers(topMCPServers map[string]any, importedMCPServersJSON string) (map[string]any, error) {
	importsLog.Print("Merging MCP servers from imports")

	if importedMCPServersJSON == "" || importedMCPServersJSON == "{}" {
		importsLog.Print("No imported MCP servers to merge")
		return topMCPServers, nil
	}

	// Initialize result with top-level MCP servers
	result := make(map[string]any)
	for k, v := range topMCPServers {
		result[k] = v
	}

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedMCPServersJSON, "\n")
	importsLog.Printf("Processing %d MCP server definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to map
		var importedMCPServers map[string]any
		if err := json.Unmarshal([]byte(line), &importedMCPServers); err != nil {
			continue // Skip invalid lines
		}

		// Merge MCP servers - imported servers take precedence over top-level ones
		for serverName, serverConfig := range importedMCPServers {
			result[serverName] = serverConfig
		}
	}

	return result, nil
}

// MergeNetworkPermissions merges network permissions from imports with top-level network permissions
// Combines allowed domains from both sources into a single list
func (c *Compiler) MergeNetworkPermissions(topNetwork *NetworkPermissions, importedNetworkJSON string) (*NetworkPermissions, error) {
	importsLog.Print("Merging network permissions from imports")

	// If no imported network config, return top-level network as-is
	if importedNetworkJSON == "" || importedNetworkJSON == "{}" {
		importsLog.Print("No imported network permissions to merge")
		return topNetwork, nil
	}

	// Start with top-level network or create a new one
	result := &NetworkPermissions{}
	if topNetwork != nil {
		result.Mode = topNetwork.Mode
		result.Allowed = make([]string, len(topNetwork.Allowed))
		copy(result.Allowed, topNetwork.Allowed)
		importsLog.Printf("Starting with %d top-level allowed domains", len(topNetwork.Allowed))
	}

	// Track domains to avoid duplicates
	domainSet := make(map[string]bool)
	for _, domain := range result.Allowed {
		domainSet[domain] = true
	}

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedNetworkJSON, "\n")
	importsLog.Printf("Processing %d network permission lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to NetworkPermissions struct
		var importedNetwork NetworkPermissions
		if err := json.Unmarshal([]byte(line), &importedNetwork); err != nil {
			continue // Skip invalid lines
		}

		// Merge allowed domains from imported network
		for _, domain := range importedNetwork.Allowed {
			if !domainSet[domain] {
				result.Allowed = append(result.Allowed, domain)
				domainSet[domain] = true
			}
		}
	}

	// Sort the final domain list for consistent output
	SortStrings(result.Allowed)

	return result, nil
}

// ValidatePermissions validates that the main workflow permissions satisfy the imported workflow requirements
// Takes the top-level permissions YAML string and imported permissions JSON string
// Returns an error if the main workflow permissions are insufficient
func (c *Compiler) ValidatePermissions(topPermissionsYAML string, importedPermissionsJSON string) error {
	importsLog.Print("Validating permissions from imports")

	// If no imported permissions, no validation needed
	if importedPermissionsJSON == "" || importedPermissionsJSON == "{}" {
		importsLog.Print("No imported permissions to validate")
		return nil
	}

	// Parse top-level permissions
	var topPerms *Permissions
	if topPermissionsYAML != "" {
		topPerms = NewPermissionsParser(topPermissionsYAML).ToPermissions()
	} else {
		topPerms = NewPermissions()
	}

	// Track missing permissions
	missingPermissions := make(map[PermissionScope]PermissionLevel)
	insufficientPermissions := make(map[PermissionScope]struct {
		required PermissionLevel
		current  PermissionLevel
	})

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedPermissionsJSON, "\n")
	importsLog.Printf("Processing %d permission definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to permissions map
		var importedPermsMap map[string]any
		if err := json.Unmarshal([]byte(line), &importedPermsMap); err != nil {
			importsLog.Printf("Skipping malformed permission entry: %q (error: %v)", line, err)
			continue
		}

		// Check each permission from the imported map
		for scopeStr, levelValue := range importedPermsMap {
			scope := PermissionScope(scopeStr)

			// Parse the level - it might be a string or already unmarshaled
			var requiredLevel PermissionLevel
			if levelStr, ok := levelValue.(string); ok {
				requiredLevel = PermissionLevel(levelStr)
			} else {
				// Skip invalid level values
				continue
			}

			// Get current level for this scope
			currentLevel, exists := topPerms.Get(scope)

			// Validate that the main workflow has sufficient permissions
			if !exists || currentLevel == PermissionNone {
				// Permission is missing entirely
				missingPermissions[scope] = requiredLevel
				importsLog.Printf("Missing permission: %s: %s", scope, requiredLevel)
			} else if !isPermissionSufficient(currentLevel, requiredLevel) {
				// Permission exists but is insufficient
				insufficientPermissions[scope] = struct {
					required PermissionLevel
					current  PermissionLevel
				}{requiredLevel, currentLevel}
				importsLog.Printf("Insufficient permission: %s: has %s, needs %s", scope, currentLevel, requiredLevel)
			}
		}
	}

	// If there are missing or insufficient permissions, return an error
	if len(missingPermissions) > 0 || len(insufficientPermissions) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("ERROR: Imported workflows require permissions that are not granted in the main workflow.\n\n")
		errorMsg.WriteString("The permission set must be explicitly declared in the main workflow.\n\n")

		if len(missingPermissions) > 0 {
			errorMsg.WriteString("Missing permissions:\n")
			// Sort for consistent output
			var scopes []PermissionScope
			for scope := range missingPermissions {
				scopes = append(scopes, scope)
			}
			SortPermissionScopes(scopes)
			for _, scope := range scopes {
				level := missingPermissions[scope]
				errorMsg.WriteString(fmt.Sprintf("  - %s: %s\n", scope, level))
			}
			errorMsg.WriteString("\n")
		}

		if len(insufficientPermissions) > 0 {
			errorMsg.WriteString("Insufficient permissions:\n")
			// Sort for consistent output
			var scopes []PermissionScope
			for scope := range insufficientPermissions {
				scopes = append(scopes, scope)
			}
			SortPermissionScopes(scopes)
			for _, scope := range scopes {
				info := insufficientPermissions[scope]
				errorMsg.WriteString(fmt.Sprintf("  - %s: has %s, requires %s\n", scope, info.current, info.required))
			}
			errorMsg.WriteString("\n")
		}

		errorMsg.WriteString("Suggested fix: Add the required permissions to your main workflow frontmatter:\n")
		errorMsg.WriteString("permissions:\n")

		// Combine all required permissions for the suggestion
		allRequired := make(map[PermissionScope]PermissionLevel)
		for scope, level := range missingPermissions {
			allRequired[scope] = level
		}
		for scope, info := range insufficientPermissions {
			allRequired[scope] = info.required
		}

		var scopes []PermissionScope
		for scope := range allRequired {
			scopes = append(scopes, scope)
		}
		SortPermissionScopes(scopes)
		for _, scope := range scopes {
			level := allRequired[scope]
			errorMsg.WriteString(fmt.Sprintf("  %s: %s\n", scope, level))
		}

		return fmt.Errorf("%s", errorMsg.String())
	}

	importsLog.Print("All imported permissions are satisfied by main workflow")
	return nil
}

// isPermissionSufficient checks if the current permission level is sufficient for the required level
// write > read > none
func isPermissionSufficient(current, required PermissionLevel) bool {
	if current == required {
		return true
	}
	// write satisfies read requirement
	if current == PermissionWrite && required == PermissionRead {
		return true
	}
	return false
}
