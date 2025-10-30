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

// MergePermissions merges permissions from imports with top-level permissions
// Takes the top-level permissions YAML string and imported permissions JSON string
// Returns the merged permissions YAML string
func (c *Compiler) MergePermissions(topPermissionsYAML string, importedPermissionsJSON string) (string, error) {
	importsLog.Print("Merging permissions from imports")

	// If no imported permissions, return top-level permissions as-is
	if importedPermissionsJSON == "" || importedPermissionsJSON == "{}" {
		importsLog.Print("No imported permissions to merge")
		return topPermissionsYAML, nil
	}

	// Parse top-level permissions if they exist
	var topPerms *Permissions
	if topPermissionsYAML != "" {
		topPerms = NewPermissionsParser(topPermissionsYAML).ToPermissions()
	} else {
		topPerms = NewPermissions()
	}

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
			continue // Skip invalid lines
		}

		// Convert imported permissions to Permissions struct
		// Handle shorthand forms like "read-all", "write-all", etc.
		if len(importedPermsMap) == 1 {
			for key := range importedPermsMap {
				// Check for shorthand
				if key == "read-all" || key == "write-all" || key == "read" || key == "write" || key == "none" {
					// Shorthand not supported in imports - skip
					importsLog.Printf("Skipping shorthand permission in import: %s", key)
					continue
				}
			}
		}

		// Merge each permission from the imported map
		for scopeStr, levelValue := range importedPermsMap {
			scope := PermissionScope(scopeStr)
			
			// Parse the level - it might be a string or already unmarshaled
			var level PermissionLevel
			if levelStr, ok := levelValue.(string); ok {
				level = PermissionLevel(levelStr)
			} else {
				// Skip invalid level values
				continue
			}

			// Get current level for this scope
			currentLevel, exists := topPerms.Get(scope)

			// Merge logic: take the higher permission level
			// write > read > none
			shouldUpdate := false
			if !exists {
				shouldUpdate = true
			} else if level == PermissionWrite && currentLevel != PermissionWrite {
				shouldUpdate = true
			} else if level == PermissionRead && currentLevel == PermissionNone {
				shouldUpdate = true
			}

			if shouldUpdate {
				topPerms.Set(scope, level)
				importsLog.Printf("Merged permission: %s: %s", scope, level)
			}
		}
	}

	// Convert back to YAML string
	mergedYAML := topPerms.RenderToYAML()

	// Adjust indentation from 6 spaces to 2 spaces for workflow-level permissions
	lines = strings.Split(mergedYAML, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "      ") {
			lines[i] = "  " + lines[i][6:]
		}
	}

	return strings.Join(lines, "\n"), nil
}
