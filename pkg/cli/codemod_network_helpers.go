package cli

import (
	"fmt"
	"strings"
)

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
					if len(nextTrimmed) == 0 {
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
		if inNetworkBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
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
			currentIndent := getIndentation(line)

			// Empty lines - skip
			if len(trimmedLine) == 0 {
				continue
			}

			// Comments at deeper indentation - skip
			if strings.HasPrefix(trimmedLine, "#") && len(currentIndent) > len(allowedIndent) {
				continue
			}

			// Array items (lines starting with -)
			if strings.HasPrefix(trimmedLine, "-") && len(currentIndent) > len(allowedIndent) {
				continue
			}

			// We've exited the allowed block
			inAllowedBlock = false
		}

		result = append(result, line)
	}

	// If we didn't find an allowed block, add it to the network block
	if !replacedAllowed {
		// Find the end of the network block and insert allowed
		result = addAllowedToNetwork(result, domains)
	}

	return result
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

		if inNetworkBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
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
		for i := len(result) - 1; i >= 0; i-- {
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
