package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var mcpNetworkCodemodLog = logger.New("cli:codemod_mcp_network")

// getMCPNetworkMigrationCodemod creates a codemod for migrating per-server MCP network configuration to top-level network configuration
func getMCPNetworkMigrationCodemod() Codemod {
	return Codemod{
		ID:           "mcp-network-to-top-level-migration",
		Name:         "Migrate MCP network config to top-level",
		Description:  "Moves per-server MCP 'network.allowed' configuration to top-level workflow 'network.allowed'. Per-server network configuration is deprecated.",
		IntroducedIn: "0.6.0",
		Apply:        getMCPNetworkMigrationCodemodApply,
	}
}

func getMCPNetworkMigrationCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	mcpServersMap, ok := getMCPNetworkMigrationCodemodServers(frontmatter)
	if !ok {
		return content, false, nil
	}

	allAllowedDomains, serversWithNetwork := getMCPNetworkMigrationCodemodServerDomains(mcpServersMap)
	if len(serversWithNetwork) == 0 {
		return content, false, nil
	}
	allAllowedDomains = sliceutil.Deduplicate(allAllowedDomains)

	existingAllowed, hasTopLevelNetwork := getMCPNetworkMigrationCodemodExistingAllowed(frontmatter)
	mergedDomains := sliceutil.Deduplicate(append(existingAllowed, allAllowedDomains...))

	return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
		return getMCPNetworkMigrationCodemodTransform(lines, serversWithNetwork, hasTopLevelNetwork, mergedDomains)
	})
}

func getMCPNetworkMigrationCodemodServers(frontmatter map[string]any) (map[string]any, bool) {
	mcpServersValue, hasMCPServers := frontmatter["mcp-servers"]
	if !hasMCPServers {
		return nil, false
	}
	mcpServersMap, ok := mcpServersValue.(map[string]any)
	return mcpServersMap, ok
}

func getMCPNetworkMigrationCodemodServerDomains(mcpServersMap map[string]any) ([]string, map[string]struct{}) {
	var allAllowedDomains []string
	serversWithNetwork := make(map[string]struct{})
	for serverName, serverValue := range mcpServersMap {
		serverConfig, ok := serverValue.(map[string]any)
		if !ok {
			continue
		}
		networkMap, ok := getMCPNetworkMigrationCodemodNetworkMap(serverConfig)
		if !ok {
			continue
		}
		allowedValue, hasAllowed := networkMap["allowed"]
		if !hasAllowed {
			continue
		}
		domains, hasDomains := getMCPNetworkMigrationCodemodAllowedStrings(allowedValue)
		allAllowedDomains = append(allAllowedDomains, domains...)
		if hasDomains {
			serversWithNetwork[serverName] = struct{}{}
		}
	}
	return allAllowedDomains, serversWithNetwork
}

func getMCPNetworkMigrationCodemodNetworkMap(serverConfig map[string]any) (map[string]any, bool) {
	networkValue, hasNetwork := serverConfig["network"]
	if !hasNetwork {
		return nil, false
	}
	networkMap, ok := networkValue.(map[string]any)
	return networkMap, ok
}

func getMCPNetworkMigrationCodemodAllowedStrings(allowedValue any) ([]string, bool) {
	var domains []string
	switch allowed := allowedValue.(type) {
	case []any:
		for _, domain := range allowed {
			if domainStr, ok := domain.(string); ok {
				domains = append(domains, domainStr)
			}
		}
		return domains, len(allowed) > 0
	case []string:
		domains = append(domains, allowed...)
		return domains, len(allowed) > 0
	}
	return nil, false
}

func getMCPNetworkMigrationCodemodExistingAllowed(frontmatter map[string]any) ([]string, bool) {
	existingNetworkValue, hasTopLevelNetwork := frontmatter["network"]
	if !hasTopLevelNetwork {
		return nil, false
	}
	existingNetworkMap, ok := existingNetworkValue.(map[string]any)
	if !ok {
		return nil, true
	}
	existingAllowedValue, hasExistingAllowed := existingNetworkMap["allowed"]
	if !hasExistingAllowed {
		return nil, true
	}
	existingAllowed, _ := getMCPNetworkMigrationCodemodAllowedStrings(existingAllowedValue)
	return existingAllowed, true
}

func getMCPNetworkMigrationCodemodTransform(
	lines []string,
	serversWithNetwork map[string]struct{},
	hasTopLevelNetwork bool,
	mergedDomains []string,
) ([]string, bool) {
	result := lines
	var modified bool
	for serverName := range serversWithNetwork {
		var serverModified bool
		result, serverModified = removeFieldFromMCPServer(result, serverName, "network")
		if serverModified {
			modified = true
			mcpNetworkCodemodLog.Printf("Removed network configuration from MCP server '%s'", serverName)
		}
	}
	if !modified {
		return lines, false
	}
	if hasTopLevelNetwork {
		result = updateNetworkAllowed(result, mergedDomains)
		mcpNetworkCodemodLog.Printf("Updated top-level network.allowed with %d domains", len(mergedDomains))
	} else {
		result = addTopLevelNetwork(result, mergedDomains)
		mcpNetworkCodemodLog.Printf("Added top-level network.allowed with %d domains", len(mergedDomains))
	}
	mcpNetworkCodemodLog.Print("Applied MCP network migration to top-level")
	return result, true
}

// removeFieldFromMCPServer removes a field from a specific MCP server configuration
func removeFieldFromMCPServer(lines []string, serverName string, fieldName string) ([]string, bool) {
	var result []string
	var modified bool
	state := removeFieldFromMCPServerState{}

	for i, line := range lines {
		keepLine, lineModified := removeFieldFromMCPServerKeepLine(line, i, serverName, fieldName, &state)
		if lineModified {
			modified = true
		}
		if keepLine {
			result = append(result, line)
		}
	}

	return result, modified
}

type removeFieldFromMCPServerState struct {
	inMCPServers     bool
	mcpServersIndent string
	inServerBlock    bool
	serverIndent     string
	inFieldBlock     bool
	fieldIndent      string
}

func removeFieldFromMCPServerKeepLine(line string, index int, serverName string, fieldName string, state *removeFieldFromMCPServerState) (bool, bool) {
	trimmedLine := strings.TrimSpace(line)

	// Track if we're in mcp-servers block
	if strings.HasPrefix(trimmedLine, "mcp-servers:") {
		state.inMCPServers = true
		state.mcpServersIndent = getIndentation(line)
		return true, false
	}

	removeFieldFromMCPServerExitBlocks(line, trimmedLine, state)

	// Track if we're in the specific server block
	if state.inMCPServers && strings.HasPrefix(trimmedLine, serverName+":") {
		state.inServerBlock = true
		state.serverIndent = getIndentation(line)
		return true, false
	}

	// Remove field line if in server block
	if state.inServerBlock && strings.HasPrefix(trimmedLine, fieldName+":") {
		state.inFieldBlock = true
		state.fieldIndent = getIndentation(line)
		mcpNetworkCodemodLog.Printf("Removed %s from mcp-server '%s' on line %d", fieldName, serverName, index+1)
		return false, true
	}

	if state.inFieldBlock {
		return removeFieldFromMCPServerKeepAfterRemovedField(line, trimmedLine, state), false
	}
	return true, false
}

func removeFieldFromMCPServerExitBlocks(line string, trimmedLine string, state *removeFieldFromMCPServerState) {
	if state.inMCPServers && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
		if hasExitedBlock(line, state.mcpServersIndent) {
			state.inMCPServers = false
			state.inServerBlock = false
		}
	}
	if state.inServerBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
		currentIndent := getIndentation(line)
		if len(currentIndent) <= len(state.serverIndent) && strings.Contains(line, ":") {
			state.inServerBlock = false
		}
	}
}

func removeFieldFromMCPServerKeepAfterRemovedField(line string, trimmedLine string, state *removeFieldFromMCPServerState) bool {
	if trimmedLine == "" {
		return false
	}

	currentIndent := getIndentation(line)
	if strings.HasPrefix(trimmedLine, "#") {
		if len(currentIndent) > len(state.fieldIndent) {
			return false
		}
		state.inFieldBlock = false
		return true
	}

	if len(currentIndent) > len(state.fieldIndent) {
		return false
	}
	state.inFieldBlock = false
	return true
}

// addTopLevelNetwork adds a new top-level network configuration
func addTopLevelNetwork(lines []string, domains []string) []string {
	// Find a good place to insert (after on: field, or at the beginning)
	insertIndex := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "on:") {
			// Insert after the on: block
			insertIndex = i + 1
			// Skip any nested content under on:
			if !strings.Contains(trimmed, "on: ") || strings.HasPrefix(trimmed, "on:") && len(trimmed) == 3 {
				// on: is a block, find the end
				onIndent := getIndentation(line)
				for j := i + 1; j < len(lines); j++ {
					nextLine := lines[j]
					nextTrimmed := strings.TrimSpace(nextLine)
					if nextTrimmed == "" {
						continue
					}
					if hasExitedBlock(nextLine, onIndent) {
						insertIndex = j
						break
					}
				}
			}
			break
		}
	}

	// Build network configuration lines
	var networkLines []string
	networkLines = append(networkLines, "network:")
	networkLines = append(networkLines, "  allowed:")
	for _, domain := range domains {
		networkLines = append(networkLines, "    - "+domain)
	}

	// Insert at the determined position
	result := make([]string, 0, len(lines)+len(networkLines))
	result = append(result, lines[:insertIndex]...)
	result = append(result, networkLines...)
	result = append(result, lines[insertIndex:]...)

	return result
}

// updateNetworkAllowed updates the existing top-level network.allowed configuration
func updateNetworkAllowed(lines []string, domains []string) []string {
	var result []string
	var inNetworkBlock bool
	var networkIndent string
	var inAllowedBlock bool
	var allowedIndent string
	var replacedAllowed bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track if we're in network block
		if strings.HasPrefix(trimmedLine, "network:") {
			inNetworkBlock = true
			networkIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left network block
		if inNetworkBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			if hasExitedBlock(line, networkIndent) {
				inNetworkBlock = false
				inAllowedBlock = false
			}
		}

		// Track if we're in allowed block within network
		if inNetworkBlock && strings.HasPrefix(trimmedLine, "allowed:") {
			inAllowedBlock = true
			allowedIndent = getIndentation(line)
			replacedAllowed = true
			// Replace the allowed block
			result = append(result, line)
			for _, domain := range domains {
				result = append(result, fmt.Sprintf("%s  - %s", allowedIndent, domain))
			}
			continue
		}

		// Skip existing allowed array items
		if inAllowedBlock {
			if updateNetworkAllowedShouldSkipAllowedLine(line, trimmedLine, allowedIndent) {
				continue
			}

			// We've exited the allowed block
			inAllowedBlock = false
		}

		result = append(result, line)
	}

	// If we didn't find an allowed block, add it to the network block
	if !replacedAllowed {
		result = addAllowedToNetwork(result, domains)
	}

	return result
}

func updateNetworkAllowedShouldSkipAllowedLine(line string, trimmedLine string, allowedIndent string) bool {
	currentIndent := getIndentation(line)
	if trimmedLine == "" {
		return true
	}
	if strings.HasPrefix(trimmedLine, "#") && len(currentIndent) > len(allowedIndent) {
		return true
	}
	return strings.HasPrefix(trimmedLine, "-") && len(currentIndent) > len(allowedIndent)
}

// addAllowedToNetwork adds an allowed field to an existing network block
func addAllowedToNetwork(lines []string, domains []string) []string {
	var result []string
	var inNetworkBlock bool
	var networkIndent string
	var insertIndex = -1

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "network:") {
			inNetworkBlock = true
			networkIndent = getIndentation(line)
		}

		if inNetworkBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			if hasExitedBlock(line, networkIndent) {
				// Found the end of network block
				insertIndex = i
				break
			}
		}

		result = append(result, line)
	}

	if insertIndex > 0 {
		// Insert allowed before the next top-level block
		allowedLines := []string{
			networkIndent + "  allowed:",
		}
		for _, domain := range domains {
			allowedLines = append(allowedLines, fmt.Sprintf("%s    - %s", networkIndent, domain))
		}

		result = append(result, allowedLines...)
		result = append(result, lines[insertIndex:]...)
	} else {
		// Append at the end of network block
		networkIndentStr := ""
		for i := range slices.Backward(result) {
			trimmed := strings.TrimSpace(result[i])
			if strings.HasPrefix(trimmed, "network:") {
				networkIndentStr = getIndentation(result[i])
				break
			}
		}
		result = append(result, networkIndentStr+"  allowed:")
		for _, domain := range domains {
			result = append(result, fmt.Sprintf("%s    - %s", networkIndentStr, domain))
		}
	}

	return result
}
