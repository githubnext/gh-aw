package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var playwrightDomainsCodemodLog = logger.New("cli:codemod_playwright_domains")

// getPlaywrightDomainsCodemod creates a codemod that migrates tools.playwright.allowed_domains
// to network.allowed. Network egress for Playwright is now controlled by the workflow firewall.
func getPlaywrightDomainsCodemod() Codemod {
	return Codemod{
		ID:           "playwright-allowed-domains-migration",
		Name:         "Migrate playwright allowed_domains to network.allowed",
		Description:  "Moves 'tools.playwright.allowed_domains' to top-level 'network.allowed'. Playwright egress is now controlled by the firewall.",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if tools.playwright.allowed_domains exists
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			playwrightValue, hasPlaywright := toolsMap["playwright"]
			if !hasPlaywright {
				return content, false, nil
			}

			playwrightMap, ok := playwrightValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			allowedDomainsValue, hasAllowedDomains := playwrightMap["allowed_domains"]
			if !hasAllowedDomains {
				return content, false, nil
			}

			// Extract domains from allowed_domains
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

			// Merge with existing network.allowed
			existingNetworkValue, hasTopLevelNetwork := frontmatter["network"]
			var existingAllowed []string
			if hasTopLevelNetwork {
				if existingNetworkMap, ok := existingNetworkValue.(map[string]any); ok {
					if existingAllowedValue, hasExistingAllowed := existingNetworkMap["allowed"]; hasExistingAllowed {
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
					}
				}
			}

			mergedDomains := sliceutil.Deduplicate(append(existingAllowed, domains...))

			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				// Remove allowed_domains from the tools.playwright block
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
			})
		},
	}
}

// removeFieldFromPlaywright removes a field from the tools.playwright block (two-level nesting)
func removeFieldFromPlaywright(lines []string, fieldName string) ([]string, bool) {
	var result []string
	var modified bool
	var inTools bool
	var toolsIndent string
	var inPlaywright bool
	var playwrightIndent string
	var inFieldBlock bool
	var fieldIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track if we're in the tools block
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the tools block
		if inTools && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			if hasExitedBlock(line, toolsIndent) {
				inTools = false
				inPlaywright = false
			}
		}

		// Track if we're in the playwright block within tools
		if inTools && strings.HasPrefix(trimmed, "playwright:") {
			inPlaywright = true
			playwrightIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the playwright block
		if inPlaywright && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			if hasExitedBlock(line, playwrightIndent) {
				inPlaywright = false
			}
		}

		// Remove the target field if in playwright block
		if inPlaywright && strings.HasPrefix(trimmed, fieldName+":") {
			modified = true
			inFieldBlock = true
			fieldIndent = getIndentation(line)
			playwrightDomainsCodemodLog.Printf("Removed %s from tools.playwright on line %d", fieldName, i+1)
			continue
		}

		// Skip nested properties under the removed field
		if inFieldBlock {
			if len(trimmed) == 0 {
				continue
			}
			currentIndent := getIndentation(line)
			if strings.HasPrefix(trimmed, "#") {
				if len(currentIndent) > len(fieldIndent) {
					continue
				}
				inFieldBlock = false
				result = append(result, line)
				continue
			}
			if len(currentIndent) > len(fieldIndent) {
				continue
			}
			inFieldBlock = false
		}

		result = append(result, line)
	}

	return result, modified
}
