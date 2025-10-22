package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed data/ecosystem_domains.json
var ecosystemDomainsJSON []byte

// ecosystemDomains holds the loaded domain data
var ecosystemDomains map[string][]string

// init loads the ecosystem domains from the embedded JSON
func init() {
	if err := json.Unmarshal(ecosystemDomainsJSON, &ecosystemDomains); err != nil {
		panic(fmt.Sprintf("failed to load ecosystem domains from JSON: %v", err))
	}
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
		return getEcosystemDomains("defaults") // Default allow-list for backwards compatibility
	}
	if network.Mode == "defaults" {
		return getEcosystemDomains("defaults") // Default allow-list for defaults mode
	}

	// Handle empty allowed list (deny-all case)
	if len(network.Allowed) == 0 {
		return []string{} // Return empty slice, not nil
	}

	// Process the allowed list, expanding ecosystem identifiers if present
	var expandedDomains []string
	for _, domain := range network.Allowed {
		// Try to get domains for this ecosystem category
		ecosystemDomains := getEcosystemDomains(domain)
		if len(ecosystemDomains) > 0 {
			// This was an ecosystem identifier, expand it
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

// ComputeAllowedDomainsForSanitization computes the list of allowed domains for output sanitization
// by combining default GitHub domains with domains from network permissions
func ComputeAllowedDomainsForSanitization(networkPermissions *NetworkPermissions) []string {
	// Get allowed domains from network permissions (includes ecosystem expansion)
	networkDomains := GetAllowedDomains(networkPermissions)

	// Create map for deduplication
	domainsMap := make(map[string]bool)

	// Always include GitHub sanitization domains
	for _, domain := range getEcosystemDomains("github-sanitization") {
		domainsMap[domain] = true
	}

	// Add network domains
	for _, domain := range networkDomains {
		domainsMap[domain] = true
	}

	// Convert map back to slice
	var result []string
	for domain := range domainsMap {
		result = append(result, domain)
	}

	// Sort for consistency
	sort.Strings(result)

	return result
}
