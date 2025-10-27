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
