package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var playwrightDomainsCodemodLog = logger.New("cli:codemod_playwright_domains")

// getPlaywrightDomainsToNetworkAllowedCodemod creates a codemod that migrates tools.playwright.allowed_domains
// to network.allowed. Network egress for Playwright is now controlled by the workflow firewall.
func getPlaywrightDomainsToNetworkAllowedCodemod() Codemod {
	return Codemod{
		ID:           "playwright-allowed-domains-migration",
		Name:         "Migrate playwright allowed_domains to network.allowed",
		Description:  "Moves 'tools.playwright.allowed_domains' to top-level 'network.allowed'. Playwright egress is now controlled by the firewall.",
		IntroducedIn: "0.9.0",
		Apply:        getPlaywrightDomainsToNetworkAllowedCodemodApply,
	}
}

func getPlaywrightDomainsToNetworkAllowedCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	playwrightMap, ok := getPlaywrightDomainsToNetworkAllowedCodemodPlaywrightMap(frontmatter)
	if !ok {
		return content, false, nil
	}
	allowedDomainsValue, hasAllowedDomains := playwrightMap["allowed_domains"]
	if !hasAllowedDomains {
		return content, false, nil
	}

	domains := getPlaywrightDomainsToNetworkAllowedCodemodDomains(allowedDomainsValue)
	existingAllowed, hasTopLevelNetwork := getPlaywrightDomainsToNetworkAllowedCodemodExistingAllowed(frontmatter)
	mergedDomains := sliceutil.Deduplicate(append(existingAllowed, domains...))

	return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
		return getPlaywrightDomainsToNetworkAllowedCodemodTransform(lines, domains, mergedDomains, hasTopLevelNetwork)
	})
}

func getPlaywrightDomainsToNetworkAllowedCodemodPlaywrightMap(frontmatter map[string]any) (map[string]any, bool) {
	toolsValue, hasTools := frontmatter["tools"]
	if !hasTools {
		return nil, false
	}
	toolsMap, ok := toolsValue.(map[string]any)
	if !ok {
		return nil, false
	}
	playwrightValue, hasPlaywright := toolsMap["playwright"]
	if !hasPlaywright {
		return nil, false
	}
	playwrightMap, ok := playwrightValue.(map[string]any)
	return playwrightMap, ok
}

func getPlaywrightDomainsToNetworkAllowedCodemodDomains(allowedDomainsValue any) []string {
	var domains []string
	switch v := allowedDomainsValue.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				domains = append(domains, s)
			}
		}
	case []string:
		domains = v
	case string:
		domains = []string{v}
	}
	return domains
}

func getPlaywrightDomainsToNetworkAllowedCodemodExistingAllowed(frontmatter map[string]any) ([]string, bool) {
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
	var existingAllowed []string
	switch allowed := existingAllowedValue.(type) {
	case []any:
		for _, domain := range allowed {
			if domainStr, ok := domain.(string); ok {
				existingAllowed = append(existingAllowed, domainStr)
			}
		}
	case []string:
		existingAllowed = append(existingAllowed, allowed...)
	}
	return existingAllowed, true
}

func getPlaywrightDomainsToNetworkAllowedCodemodTransform(
	lines []string,
	domains []string,
	mergedDomains []string,
	hasTopLevelNetwork bool,
) ([]string, bool) {
	result, modified := removeFieldFromPlaywright(lines, "allowed_domains")
	if !modified {
		return lines, false
	}
	playwrightDomainsCodemodLog.Printf("Removed allowed_domains from tools.playwright (%d domain(s))", len(domains))

	if hasTopLevelNetwork {
		result = updateNetworkAllowed(result, mergedDomains)
		playwrightDomainsCodemodLog.Printf("Updated top-level network.allowed with %d domain(s)", len(mergedDomains))
	} else {
		result = addTopLevelNetwork(result, mergedDomains)
		playwrightDomainsCodemodLog.Printf("Added top-level network.allowed with %d domain(s)", len(mergedDomains))
	}
	return result, true
}

// removeFieldFromPlaywright removes a field from the tools.playwright block (two-level nesting)
func removeFieldFromPlaywright(lines []string, fieldName string) ([]string, bool) {
	var result []string
	var modified bool
	state := removeFieldFromPlaywrightState{}

	for i, line := range lines {
		keepLine, lineModified := removeFieldFromPlaywrightKeepLine(line, i, fieldName, &state)
		if lineModified {
			modified = true
		}
		if keepLine {
			result = append(result, line)
		}
	}

	return result, modified
}

type removeFieldFromPlaywrightState struct {
	inTools          bool
	toolsIndent      string
	inPlaywright     bool
	playwrightIndent string
	inFieldBlock     bool
	fieldIndent      string
}

func removeFieldFromPlaywrightKeepLine(line string, index int, fieldName string, state *removeFieldFromPlaywrightState) (bool, bool) {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "tools:") {
		state.inTools = true
		state.toolsIndent = getIndentation(line)
		return true, false
	}

	removeFieldFromPlaywrightExitBlocks(line, trimmed, state)

	if state.inTools && strings.HasPrefix(trimmed, "playwright:") {
		state.inPlaywright = true
		state.playwrightIndent = getIndentation(line)
		return true, false
	}

	if state.inPlaywright && strings.HasPrefix(trimmed, fieldName+":") {
		state.inFieldBlock = true
		state.fieldIndent = getIndentation(line)
		playwrightDomainsCodemodLog.Printf("Removed %s from tools.playwright on line %d", fieldName, index+1)
		return false, true
	}

	if state.inFieldBlock {
		return removeFieldFromPlaywrightKeepAfterRemovedField(line, trimmed, state), false
	}
	return true, false
}

func removeFieldFromPlaywrightExitBlocks(line string, trimmed string, state *removeFieldFromPlaywrightState) {
	if state.inTools && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
		if hasExitedBlock(line, state.toolsIndent) {
			state.inTools = false
			state.inPlaywright = false
		}
	}
	if state.inPlaywright && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
		if hasExitedBlock(line, state.playwrightIndent) {
			state.inPlaywright = false
		}
	}
}

func removeFieldFromPlaywrightKeepAfterRemovedField(line string, trimmed string, state *removeFieldFromPlaywrightState) bool {
	if trimmed == "" {
		return false
	}
	currentIndent := getIndentation(line)
	if strings.HasPrefix(trimmed, "#") {
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
