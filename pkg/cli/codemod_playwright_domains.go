package cli

import (
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getPlaywrightDomainsToNetworkAllowedCodemod migrates Playwright domains to the workflow firewall.
func getPlaywrightDomainsToNetworkAllowedCodemod() Codemod {
	return Codemod{
		ID:           "playwright-allowed-domains-migration",
		Name:         "Migrate playwright allowed_domains to network.allowed",
		Description:  "Moves 'tools.playwright.allowed_domains' to top-level 'network.allowed'. Playwright egress is now controlled by the firewall.",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			tools, ok := frontmatter["tools"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			playwright, ok := tools["playwright"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			allowed, exists := playwright["allowed_domains"]
			if !exists {
				return content, false, nil
			}
			domains := networkDomainStrings(allowed)
			if domain, ok := allowed.(string); ok {
				domains = []string{domain}
			}
			if network, ok := frontmatter["network"].(map[string]any); ok {
				domains = append(networkDomainStrings(network["allowed"]), domains...)
			}
			domains = sliceutil.Deduplicate(domains)
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if err := f.Remove("tools", "playwright", "allowed_domains"); err != nil {
					return err
				}
				return f.Set([]string{"network", "allowed"}, domains)
			})
		},
	}
}
