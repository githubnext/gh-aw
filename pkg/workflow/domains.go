package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var domainsLog = logger.New("workflow:domains")

//go:embed data/ecosystem_domains.json
var ecosystemDomainsJSON []byte

// ecosystemDomains holds the loaded domain data
var ecosystemDomains map[string][]string

// CopilotDefaultDomains are the default domains required for GitHub Copilot CLI authentication and operation
var CopilotDefaultDomains = []string{
	"api.enterprise.githubcopilot.com",
	"api.github.com",
	"github.com",
	"raw.githubusercontent.com",
	"registry.npmjs.org",
}

// init loads the ecosystem domains from the embedded JSON
func init() {
	domainsLog.Print("Loading ecosystem domains from embedded JSON")

	if err := json.Unmarshal(ecosystemDomainsJSON, &ecosystemDomains); err != nil {
		panic(fmt.Sprintf("failed to load ecosystem domains from JSON: %v", err))
	}

	domainsLog.Printf("Loaded %d ecosystem categories", len(ecosystemDomains))
}

// getEcosystemDomains returns the domains for a given ecosystem category
func getEcosystemDomains(category string) []string {
	domains, exists := ecosystemDomains[category]
	if !exists {
		return []string{}
	}
	// Return a copy to avoid external modification
	result := make([]string, len(domains))
	copy(result, domains)
	return result
}

// GetAllowedDomains returns the allowed domains from network permissions
// Returns default allow-list if no network permissions configured or in "defaults" mode
// Returns empty slice if network permissions configured but no domains allowed (deny all)
// Returns domain list if network permissions configured with allowed domains
// Supports ecosystem identifiers:
//   - "defaults": basic infrastructure (certs, JSON schema, Ubuntu, common package mirrors, Microsoft sources)
//   - "containers": container registries (Docker, GitHub Container Registry, etc.)
//   - "dotnet": .NET and NuGet ecosystem
//   - "dart": Dart/Flutter ecosystem
//   - "github": GitHub domains
//   - "go": Go ecosystem
//   - "terraform": HashiCorp/Terraform
//   - "haskell": Haskell ecosystem
//   - "java": Java/Maven/Gradle
//   - "linux-distros": Linux distribution package repositories
//   - "node": Node.js/NPM/Yarn
//   - "perl": Perl/CPAN
//   - "php": PHP/Composer
//   - "playwright": Playwright testing framework
//   - "python": Python/PyPI/Conda
//   - "ruby": Ruby/RubyGems
//   - "rust": Rust/Cargo/Crates
//   - "swift": Swift/CocoaPods
//   - "github-actions": GitHub Actions domains
func GetAllowedDomains(network *NetworkPermissions) []string {
	if network == nil {
		domainsLog.Print("No network permissions specified, using defaults")
		return getEcosystemDomains("defaults") // Default allow-list for backwards compatibility
	}
	if network.Mode == "defaults" {
		domainsLog.Print("Network mode is defaults, using default ecosystem domains")
		return getEcosystemDomains("defaults") // Default allow-list for defaults mode
	}

	// Handle empty allowed list (deny-all case)
	if len(network.Allowed) == 0 {
		domainsLog.Print("Empty allowed list, denying all network access")
		return []string{} // Return empty slice, not nil
	}

	domainsLog.Printf("Processing %d allowed domains/ecosystems", len(network.Allowed))

	// Process the allowed list, expanding ecosystem identifiers if present
	var expandedDomains []string
	for _, domain := range network.Allowed {
		// Try to get domains for this ecosystem category
		ecosystemDomains := getEcosystemDomains(domain)
		if len(ecosystemDomains) > 0 {
			// This was an ecosystem identifier, expand it
			domainsLog.Printf("Expanded ecosystem '%s' to %d domains", domain, len(ecosystemDomains))
			expandedDomains = append(expandedDomains, ecosystemDomains...)
		} else {
			// Add the domain as-is (regular domain name)
			expandedDomains = append(expandedDomains, domain)
		}
	}

	return expandedDomains
}

// GetDomainEcosystem returns the ecosystem identifier for a given domain, or empty string if not found
func GetDomainEcosystem(domain string) string {
	// Check each ecosystem for domain match
	for ecosystem := range ecosystemDomains {
		domains := getEcosystemDomains(ecosystem)
		for _, ecosystemDomain := range domains {
			if matchesDomain(domain, ecosystemDomain) {
				return ecosystem
			}
		}
	}

	return "" // No ecosystem found
}

// matchesDomain checks if a domain matches a pattern (supports wildcards)
func matchesDomain(domain, pattern string) bool {
	// Exact match
	if domain == pattern {
		return true
	}

	// Wildcard match
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:] // Remove "*."
		return strings.HasSuffix(domain, "."+suffix) || domain == suffix
	}

	return false
}

// GetCopilotAllowedDomains merges Copilot default domains with NetworkPermissions allowed domains
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag
func GetCopilotAllowedDomains(network *NetworkPermissions) string {
	domainMap := make(map[string]bool)

	// Add Copilot default domains
	for _, domain := range CopilotDefaultDomains {
		domainMap[domain] = true
	}

	// Add NetworkPermissions domains (if specified)
	if network != nil && len(network.Allowed) > 0 {
		// Expand ecosystem identifiers and add individual domains
		expandedDomains := GetAllowedDomains(network)
		for _, domain := range expandedDomains {
			domainMap[domain] = true
		}
	}

	// Convert to sorted slice for consistent output
	domains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		domains = append(domains, domain)
	}
	SortStrings(domains)

	// Join with commas for AWF --allow-domains flag
	return strings.Join(domains, ",")
}
