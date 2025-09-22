package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
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

// NetworkHookGenerator generates network permission hooks for engine configurations
type NetworkHookGenerator struct{}

// GenerateNetworkHookWorkflowStepJS generates a GitHub Actions workflow step using JavaScript instead of Python
func (g *NetworkHookGenerator) GenerateNetworkHookWorkflowStepJS(allowedDomains []string) GitHubActionStep {
	var lines []string
	lines = append(lines, "      - name: Network Permissions Validation")
	lines = append(lines, "        uses: actions/github-script@v8")
	lines = append(lines, "        with:")
	lines = append(lines, "          script: |")

	// Get the JavaScript content and format it for YAML
	jsContent := GetNetworkPermissionsHookScript()
	jsLines := FormatJavaScriptForYAML(jsContent)
	lines = append(lines, jsLines...)

	return GitHubActionStep(lines)
}

// ShouldEnforceNetworkPermissions checks if network permissions should be enforced
// Returns true if network permissions are configured and not in "defaults" mode
func ShouldEnforceNetworkPermissions(network *NetworkPermissions) bool {
	if network == nil {
		return false // No network config, defaults to full access
	}
	if network.Mode == "defaults" {
		return true // "defaults" mode uses restricted allow-list (enforcement needed)
	}
	return true // Object format means some restriction is configured
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

// HasNetworkPermissions is deprecated - use ShouldEnforceNetworkPermissions instead
// Kept for backwards compatibility but will be removed in future versions
func HasNetworkPermissions(engineConfig *EngineConfig) bool {
	// This function is now deprecated since network permissions are top-level
	// Return false for backwards compatibility
	return false
}
