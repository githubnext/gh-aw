package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var networkAllowedDomainsCodemodLog = logger.New("cli:codemod_network_allowed_domains")

// getNetworkAllowedDomainsCodemod creates a codemod that renames network.allowed-domains to network.allowed.
// Users sometimes write "allowed-domains" by analogy with "safe-outputs.allowed-domains", but the correct
// key under the network block is "allowed". This codemod merges the two lists when both are present.
func getNetworkAllowedDomainsCodemod() Codemod {
	return Codemod{
		ID:           "network-allowed-domains-rename",
		Name:         "Rename network.allowed-domains to network.allowed",
		Description:  "Renames the deprecated 'network.allowed-domains' key to 'network.allowed', merging with any existing 'network.allowed' entries.",
		IntroducedIn: "0.1.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			networkValue, hasNetwork := frontmatter["network"]
			if !hasNetwork {
				return content, false, nil
			}

			networkMap, ok := networkValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			if _, hasAllowedDomains := networkMap["allowed-domains"]; !hasAllowedDomains {
				return content, false, nil
			}

			networkAllowedDomainsCodemodLog.Print("network.allowed-domains detected, migrating to network.allowed")

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return renameNetworkAllowedDomains(lines)
			})

			if err != nil {
				return content, false, err
			}
			if applied {
				networkAllowedDomainsCodemodLog.Print("Applied network.allowed-domains -> network.allowed migration")
			}
			return newContent, applied, nil
		},
	}
}

// renameNetworkAllowedDomains performs a single-pass rename of network.allowed-domains to
// network.allowed in the YAML frontmatter lines. When both keys are present, the domains are
// merged (allowed first, allowed-domains second, deduped).
func renameNetworkAllowedDomains(lines []string) ([]string, bool) {
	// Locate the network block
	networkStart, networkEnd := findNetworkBlockBounds(lines)
	if networkStart == -1 {
		return lines, false
	}

	networkLines := lines[networkStart : networkEnd+1]

	// Find allowed-domains and allowed blocks within network
	adStart, adEnd := findFieldBlock(networkLines, "allowed-domains")
	if adStart == -1 {
		return lines, false
	}
	allowedStart, allowedEnd := findFieldBlock(networkLines, "allowed")

	networkIndent := getIndentation(networkLines[0])
	fieldIndent := networkIndent + "  "

	// Collect domain items from allowed-domains
	adDomains := collectListItems(networkLines[adStart : adEnd+1])

	if allowedStart == -1 {
		// No existing allowed — simply rename allowed-domains: → allowed:
		result := make([]string, len(lines))
		copy(result, lines)
		absADStart := networkStart + adStart
		result[absADStart] = fieldIndent + "allowed:"
		networkAllowedDomainsCodemodLog.Printf("Renamed allowed-domains to allowed at line %d", absADStart+1)
		return result, true
	}

	// Both exist — merge and rewrite. Collect existing allowed domains.
	existingDomains := collectListItems(networkLines[allowedStart : allowedEnd+1])

	// Merge: existing first, then new (dedup)
	merged := mergeDomains(existingDomains, adDomains)

	// Build new network block: remove both fields, insert merged allowed
	newNetworkLines := removeBlock(networkLines, adStart, adEnd)
	allowedStart2, allowedEnd2 := findFieldBlock(newNetworkLines, "allowed")
	if allowedStart2 != -1 {
		newNetworkLines = removeBlock(newNetworkLines, allowedStart2, allowedEnd2)
	}

	// Insert merged allowed after network: header
	mergedLines := buildAllowedLines(fieldIndent, merged)
	newNetworkLines = insertAfterIndex(newNetworkLines, 0, mergedLines)

	result := make([]string, 0, len(lines))
	result = append(result, lines[:networkStart]...)
	result = append(result, newNetworkLines...)
	result = append(result, lines[networkEnd+1:]...)
	networkAllowedDomainsCodemodLog.Printf("Merged allowed-domains into allowed (%d domain(s))", len(merged))
	return result, true
}

// findNetworkBlockBounds returns the start and end indices (inclusive) of the network block
// in the YAML frontmatter lines. Returns (-1, -1) if not found.
func findNetworkBlockBounds(lines []string) (int, int) {
	start := -1
	indent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "network:") {
			start = i
			indent = getIndentation(line)
			continue
		}
		if start != -1 {
			if trimmed == "" {
				continue
			}
			if len(getIndentation(line)) <= len(indent) && !strings.HasPrefix(trimmed, "#") {
				return start, i - 1
			}
		}
	}
	if start != -1 {
		return start, len(lines) - 1
	}
	return -1, -1
}

// findFieldBlock returns the start and end indices (inclusive) of a YAML field block
// (key + all nested list items / sub-keys) within a slice of lines.
// Returns (-1, -1) if the field is not found.
func findFieldBlock(lines []string, key string) (int, int) {
	start := -1
	keyIndent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, key+":") {
			start = i
			keyIndent = getIndentation(line)
			continue
		}
		if start != -1 {
			if trimmed == "" {
				continue
			}
			currentIndent := getIndentation(line)
			if len(currentIndent) <= len(keyIndent) {
				return start, i - 1
			}
		}
	}
	if start != -1 {
		return start, len(lines) - 1
	}
	return -1, -1
}

// collectListItems extracts the string values from a YAML list block.
// It reads lines of the form "  - value".
func collectListItems(lines []string) []string {
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if val, ok := strings.CutPrefix(trimmed, "- "); ok {
			val = strings.Trim(val, `"'`)
			if val != "" {
				result = append(result, val)
			}
		}
	}
	return result
}

// mergeDomains merges two domain slices, deduplicating while preserving order (base first).
func mergeDomains(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))
	for _, d := range base {
		if !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}
	for _, d := range extra {
		if !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}
	return result
}

// removeBlock removes lines[start:end+1] from a slice and returns the new slice.
func removeBlock(lines []string, start, end int) []string {
	result := make([]string, 0, len(lines)-(end-start+1))
	result = append(result, lines[:start]...)
	result = append(result, lines[end+1:]...)
	return result
}

// insertAfterIndex inserts insertion after lines[idx] and returns the new slice.
func insertAfterIndex(lines []string, idx int, insertion []string) []string {
	result := make([]string, 0, len(lines)+len(insertion))
	result = append(result, lines[:idx+1]...)
	result = append(result, insertion...)
	result = append(result, lines[idx+1:]...)
	return result
}

// buildAllowedLines builds the "allowed:" YAML lines for the given indent and domain list.
func buildAllowedLines(indent string, domains []string) []string {
	if len(domains) == 0 {
		return []string{indent + "allowed: []"}
	}
	lines := make([]string, 0, 1+len(domains))
	lines = append(lines, indent+"allowed:")
	itemIndent := indent + "  "
	for _, d := range domains {
		// Quote domains that contain special characters (e.g. wildcards)
		if strings.Contains(d, "*") {
			lines = append(lines, fmt.Sprintf("%s- %q", itemIndent, d))
		} else {
			lines = append(lines, fmt.Sprintf("%s- %s", itemIndent, d))
		}
	}
	return lines
}
