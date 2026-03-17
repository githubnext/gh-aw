package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var domainsLog = logger.New("workflow:domains")

//go:embed data/ecosystem_domains.json
var ecosystemDomainsJSON []byte

// ecosystemDomains holds the loaded domain data
var ecosystemDomains map[string][]string

// CopilotDefaultDomains are the default domains required for GitHub Copilot CLI authentication and operation
var CopilotDefaultDomains = []string{
	"api.business.githubcopilot.com",
	"api.enterprise.githubcopilot.com",
	"api.github.com",
	"api.githubcopilot.com",
	"api.individual.githubcopilot.com",
	"github.com",
	"host.docker.internal",
	"raw.githubusercontent.com",
	"registry.npmjs.org",
	"telemetry.enterprise.githubcopilot.com",
}

// CodexDefaultDomains are the minimal default domains required for Codex CLI operation
var CodexDefaultDomains = []string{
	"172.30.0.1", // AWF gateway IP - Codex resolves host.docker.internal to this IP for Rust DNS compatibility
	"api.openai.com",
	"host.docker.internal",
	"openai.com",
}

// ClaudeDefaultDomains are the default domains required for Claude Code CLI authentication and operation
var ClaudeDefaultDomains = []string{
	"*.githubusercontent.com",
	"anthropic.com",
	"api.anthropic.com",
	"api.github.com",
	"api.snapcraft.io",
	"archive.ubuntu.com",
	"azure.archive.ubuntu.com",
	"cdn.playwright.dev",
	"codeload.github.com",
	"crl.geotrust.com",
	"crl.globalsign.com",
	"crl.identrust.com",
	"crl.sectigo.com",
	"crl.thawte.com",
	"crl.usertrust.com",
	"crl.verisign.com",
	"crl3.digicert.com",
	"crl4.digicert.com",
	"crls.ssl.com",
	"files.pythonhosted.org",
	"ghcr.io",
	"github-cloud.githubusercontent.com",
	"github-cloud.s3.amazonaws.com",
	"github.com",
	"host.docker.internal",
	"json-schema.org",
	"json.schemastore.org",
	"keyserver.ubuntu.com",
	"lfs.github.com",
	"objects.githubusercontent.com",
	"ocsp.digicert.com",
	"ocsp.geotrust.com",
	"ocsp.globalsign.com",
	"ocsp.identrust.com",
	"ocsp.sectigo.com",
	"ocsp.ssl.com",
	"ocsp.thawte.com",
	"ocsp.usertrust.com",
	"ocsp.verisign.com",
	"packagecloud.io",
	"packages.cloud.google.com",
	"packages.microsoft.com",
	"playwright.download.prss.microsoft.com",
	"ppa.launchpad.net",
	"pypi.org",
	"raw.githubusercontent.com",
	"registry.npmjs.org",
	"s.symcb.com",
	"s.symcd.com",
	"security.ubuntu.com",
	"sentry.io",
	"statsig.anthropic.com",
	"ts-crl.ws.symantec.com",
	"ts-ocsp.ws.symantec.com",
}

// GeminiDefaultDomains are the default domains required for Google Gemini CLI authentication and operation
var GeminiDefaultDomains = []string{
	"*.googleapis.com",
	"generativelanguage.googleapis.com",
	"github.com",
	"host.docker.internal",
	"raw.githubusercontent.com",
	"registry.npmjs.org",
}

// PlaywrightDomains are the domains required for Playwright browser downloads
// These domains are needed when Playwright MCP server initializes in the Docker container
var PlaywrightDomains = []string{
	"cdn.playwright.dev",
	"playwright.download.prss.microsoft.com",
}

// init loads the ecosystem domains from the embedded JSON
func init() {
	domainsLog.Print("Loading ecosystem domains from embedded JSON")

	if err := json.Unmarshal(ecosystemDomainsJSON, &ecosystemDomains); err != nil {
		panic(fmt.Sprintf("failed to load ecosystem domains from JSON: %v", err))
	}

	domainsLog.Printf("Loaded %d ecosystem categories", len(ecosystemDomains))
}

// compoundEcosystems defines ecosystem identifiers that expand to the union of multiple
// component ecosystems. These are resolved at lookup time, so they stay in sync with
// any future changes to the component ecosystems.
var compoundEcosystems = map[string][]string{
	// default-safe-outputs: the recommended baseline for URL redaction in safe-outputs.
	// Covers common infrastructure certificate/OCSP hosts (via "defaults"), popular
	// developer-tool and CI/CD service domains (via "dev-tools"), GitHub domains (via "github"),
	// and loopback/localhost addresses (via "local").
	"default-safe-outputs": {"defaults", "dev-tools", "github", "local"},
}

// getEcosystemDomains returns the domains for a given ecosystem category.
// Supports compound ecosystem identifiers (see compoundEcosystems).
// The returned list is sorted and contains unique entries.
func getEcosystemDomains(category string) []string {
	// Check for compound ecosystem first
	if components, ok := compoundEcosystems[category]; ok {
		domainMap := make(map[string]bool)
		for _, component := range components {
			for _, d := range getEcosystemDomains(component) {
				domainMap[d] = true
			}
		}
		result := make([]string, 0, len(domainMap))
		for d := range domainMap {
			result = append(result, d)
		}
		sort.Strings(result)
		return result
	}

	domains, exists := ecosystemDomains[category]
	if !exists {
		return []string{}
	}
	// Return a sorted copy to avoid external modification
	result := make([]string, len(domains))
	copy(result, domains)
	sort.Strings(result)
	return result
}

// runtimeToEcosystem maps runtime IDs to their corresponding ecosystem categories in ecosystem_domains.json
// Some runtimes share ecosystems (e.g., bun and deno use node ecosystem domains)
var runtimeToEcosystem = map[string]string{
	"node":    "node",
	"python":  "python",
	"go":      "go",
	"java":    "java",
	"ruby":    "ruby",
	"dotnet":  "dotnet",
	"haskell": "haskell",
	"bun":     "node",   // bun.sh is in the node ecosystem
	"deno":    "node",   // deno.land is in the node ecosystem
	"uv":      "python", // uv is a Python package manager
	"clojure": "clojure",
	"dart":    "dart",
	"elixir":  "elixir",
	"kotlin":  "kotlin",
	"php":     "php",
	"scala":   "scala",
	"swift":   "swift",
	"zig":     "zig",
}

// getDomainsFromRuntimes extracts ecosystem domains based on the specified runtimes
// Returns a deduplicated list of domains for all specified runtimes
func getDomainsFromRuntimes(runtimes map[string]any) []string {
	if len(runtimes) == 0 {
		return []string{}
	}

	domainMap := make(map[string]bool)

	for runtimeID := range runtimes {
		// Look up the ecosystem for this runtime
		ecosystem, exists := runtimeToEcosystem[runtimeID]
		if !exists {
			domainsLog.Printf("No ecosystem mapping for runtime '%s'", runtimeID)
			continue
		}

		// Get domains for this ecosystem
		domains := getEcosystemDomains(ecosystem)
		if len(domains) > 0 {
			domainsLog.Printf("Runtime '%s' mapped to ecosystem '%s' with %d domains", runtimeID, ecosystem, len(domains))
			for _, d := range domains {
				domainMap[d] = true
			}
		}
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		result = append(result, domain)
	}
	sort.Strings(result)

	return result
}

// GetAllowedDomains returns the allowed domains from network permissions.
//
// # Behavior based on network permissions configuration:
//
//  1. No network permissions (nil):
//     Returns default ecosystem domains for backwards compatibility.
//
//  2. Allowed list with "defaults" only:
//     network: defaults  OR  network: { allowed: [defaults] }
//     Returns default ecosystem domains.
//
//  3. Allowed list with multiple ecosystems:
//     network:
//     allowed:
//     - defaults
//     - github
//     Processes the Allowed list, expanding all ecosystem identifiers and merging them.
//
//  4. Allowed list with custom domains:
//     network:
//     allowed:
//     - example.com
//     - python
//     Processes the Allowed list, expanding ecosystem identifiers.
//
//  5. Empty Allowed list (deny-all):
//     network: {}  OR  network: { allowed: [] }
//     Returns empty slice (no network access).
//
// The returned list is sorted and deduplicated.
//
// # Supported ecosystem identifiers:
//   - "defaults": basic infrastructure (certs, JSON schema, Ubuntu, package mirrors)
//   - "clojure": Clojure/Clojars
//   - "containers": container registries (Docker, GHCR, etc.)
//   - "dart": Dart/Flutter ecosystem
//   - "dotnet": .NET and NuGet ecosystem
//   - "elixir": Elixir/Hex
//   - "github": GitHub domains (*.githubusercontent.com, github.githubassets.com, etc.)
//   - "github-actions": GitHub Actions blob storage domains
//   - "go": Go ecosystem
//   - "haskell": Haskell ecosystem
//   - "java": Java/Maven/Gradle
//   - "kotlin": Kotlin/JetBrains
//   - "linux-distros": Linux distribution package repositories
//   - "node": Node.js/NPM/Yarn
//   - "perl": Perl/CPAN
//   - "php": PHP/Composer
//   - "playwright": Playwright testing framework
//   - "python": Python/PyPI/Conda
//   - "ruby": Ruby/RubyGems
//   - "rust": Rust/Cargo/Crates
//   - "scala": Scala/SBT
//   - "swift": Swift/CocoaPods
//   - "terraform": HashiCorp/Terraform
//   - "zig": Zig
func GetAllowedDomains(network *NetworkPermissions) []string {
	if network == nil {
		domainsLog.Print("No network permissions specified, using defaults")
		return getEcosystemDomains("defaults") // Default allow-list for backwards compatibility
	}

	// Handle empty allowed list (deny-all case)
	if len(network.Allowed) == 0 {
		domainsLog.Print("Empty allowed list, denying all network access")
		return []string{} // Return empty slice, not nil
	}

	domainsLog.Printf("Processing %d allowed domains/ecosystems", len(network.Allowed))

	// Process the allowed list, expanding ecosystem identifiers if present
	// Use a map to deduplicate domains
	domainMap := make(map[string]bool)
	for _, domain := range network.Allowed {
		// Try to get domains for this ecosystem category
		ecosystemDomains := getEcosystemDomains(domain)
		if len(ecosystemDomains) > 0 {
			// This was an ecosystem identifier, expand it
			domainsLog.Printf("Expanded ecosystem '%s' to %d domains", domain, len(ecosystemDomains))
			for _, d := range ecosystemDomains {
				domainMap[d] = true
			}
		} else {
			// Add the domain as-is (regular domain name)
			domainMap[domain] = true
		}
	}

	// Convert map to sorted slice
	expandedDomains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		expandedDomains = append(expandedDomains, domain)
	}
	sort.Strings(expandedDomains)

	return expandedDomains
}

// ecosystemPriority defines the order in which ecosystems are checked by GetDomainEcosystem.
// More specific sub-ecosystems are listed before their parent ecosystems so that domains
// shared between multiple ecosystems resolve deterministically to the most specific one.
// For example, "node-cdns" is listed before "node" so that cdn.jsdelivr.net returns "node-cdns".
// All known ecosystems are enumerated here; any ecosystem not in this list is checked last
// in sorted order (for forward-compatibility with new entries).
var ecosystemPriority = []string{
	"node-cdns", // before "node" — more specific CDN sub-ecosystem
	"rust",      // before "python" — crates.io/index.crates.io/static.crates.io are native Rust domains
	"clojure",
	"containers",
	"dart",
	"defaults",
	"dev-tools",
	"dotnet",
	"elixir",
	"fonts",
	"github",
	"github-actions",
	"go",
	"haskell",
	"java",
	"kotlin",
	"linux-distros",
	"local",
	"node",
	"perl",
	"php",
	"playwright",
	"python",
	"ruby",
	"scala",
	"swift",
	"terraform",
	"zig",
	"default-safe-outputs", // compound: defaults + dev-tools + github + local
}

// GetDomainEcosystem returns the ecosystem identifier for a given domain, or empty string if not found.
// Ecosystems are checked in ecosystemPriority order so that the result is deterministic even when
// a domain appears in multiple ecosystems (e.g. cdn.jsdelivr.net is in both "node" and "node-cdns").
func GetDomainEcosystem(domain string) string {
	checked := make(map[string]bool, len(ecosystemPriority))

	// Check ecosystems in priority order first
	for _, ecosystem := range ecosystemPriority {
		checked[ecosystem] = true
		domains := getEcosystemDomains(ecosystem)
		for _, ecosystemDomain := range domains {
			if matchesDomain(domain, ecosystemDomain) {
				return ecosystem
			}
		}
	}

	// Fall back to any ecosystems not in the priority list, sorted for determinism
	remaining := make([]string, 0)
	for ecosystem := range ecosystemDomains {
		if !checked[ecosystem] {
			remaining = append(remaining, ecosystem)
		}
	}
	sort.Strings(remaining)
	for _, ecosystem := range remaining {
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

// extractHTTPMCPDomains extracts domain names from HTTP MCP server URLs in tools configuration
// Returns a slice of domain names (e.g., ["mcp.tavily.com", "api.example.com"])
func extractHTTPMCPDomains(tools map[string]any) []string {
	if tools == nil {
		return []string{}
	}

	domains := []string{}

	// Iterate through tools to find HTTP MCP servers
	for toolName, toolConfig := range tools {
		configMap, ok := toolConfig.(map[string]any)
		if !ok {
			// Tool has no explicit config (e.g., github: null means local mode)
			continue
		}

		// Special handling for GitHub MCP in remote mode
		// When mode: remote is set, the URL is implicitly the hosted GitHub Copilot MCP server
		if toolName == "github" {
			if modeField, hasMode := configMap["mode"]; hasMode {
				if modeStr, ok := modeField.(string); ok && modeStr == "remote" {
					domainsLog.Printf("Detected GitHub MCP remote mode, adding %s to domains", constants.GitHubCopilotMCPDomain)
					domains = append(domains, constants.GitHubCopilotMCPDomain)
					continue
				}
			}
		}

		// Check if this is an HTTP MCP server
		mcpType, hasType := configMap["type"].(string)
		url, hasURL := configMap["url"].(string)

		// HTTP MCP servers have either type: http or just a url field
		isHTTPMCP := (hasType && mcpType == "http") || (!hasType && hasURL)

		if isHTTPMCP && hasURL {
			// Extract domain from URL (e.g., "https://mcp.tavily.com/mcp/" -> "mcp.tavily.com")
			domain := stringutil.ExtractDomainFromURL(url)
			if domain != "" {
				domainsLog.Printf("Extracted HTTP MCP domain '%s' from tool '%s'", domain, toolName)
				domains = append(domains, domain)
			}
		}
	}

	return domains
}

// extractPlaywrightDomains returns Playwright domains when Playwright tool is configured
// Returns a slice of domain names required for Playwright browser downloads
// These domains are needed when Playwright MCP server initializes in the Docker container
func extractPlaywrightDomains(tools map[string]any) []string {
	if tools == nil {
		return []string{}
	}

	// Check if Playwright tool is configured
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		domainsLog.Printf("Detected Playwright tool, adding %d domains for browser downloads", len(PlaywrightDomains))
		return PlaywrightDomains
	}

	return []string{}
}

// mergeDomainsWithNetworkToolsAndRuntimes combines default domains with NetworkPermissions, HTTP MCP server domains, and runtime ecosystem domains
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag
func mergeDomainsWithNetworkToolsAndRuntimes(defaultDomains []string, network *NetworkPermissions, tools map[string]any, runtimes map[string]any) string {
	domainMap := make(map[string]bool)

	// Add default domains
	for _, domain := range defaultDomains {
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

	// Add HTTP MCP server domains (if tools are specified)
	if tools != nil {
		mcpDomains := extractHTTPMCPDomains(tools)
		for _, domain := range mcpDomains {
			domainMap[domain] = true
		}
	}

	// Add Playwright ecosystem domains (if Playwright tool is specified)
	// This ensures browser binaries can be downloaded when Playwright initializes
	if tools != nil {
		playwrightDomains := extractPlaywrightDomains(tools)
		for _, domain := range playwrightDomains {
			domainMap[domain] = true
		}
	}

	// Add runtime ecosystem domains (if runtimes are specified)
	if runtimes != nil {
		runtimeDomains := getDomainsFromRuntimes(runtimes)
		for _, domain := range runtimeDomains {
			domainMap[domain] = true
		}
	}

	// Convert to sorted slice for consistent output
	domains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	// Join with commas for AWF --allow-domains flag
	return strings.Join(domains, ",")
}

// engineDefaultDomains maps each engine to its default required domains.
// Add new engines here to avoid adding new engine-specific domain functions.
var engineDefaultDomains = map[constants.EngineName][]string{
	constants.CopilotEngine: CopilotDefaultDomains,
	constants.ClaudeEngine:  ClaudeDefaultDomains,
	constants.CodexEngine:   CodexDefaultDomains,
	constants.GeminiEngine:  GeminiDefaultDomains,
}

// GetAllowedDomainsForEngine merges the engine's default domains with NetworkPermissions,
// HTTP MCP server domains, and runtime ecosystem domains.
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag.
// Falls back to an empty default domain list for unknown engines.
func GetAllowedDomainsForEngine(engine constants.EngineName, network *NetworkPermissions, tools map[string]any, runtimes map[string]any) string {
	return mergeDomainsWithNetworkToolsAndRuntimes(engineDefaultDomains[engine], network, tools, runtimes)
}

// GetCopilotAllowedDomainsWithToolsAndRuntimes merges Copilot default domains with NetworkPermissions, HTTP MCP server domains, and runtime ecosystem domains
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag
func GetCopilotAllowedDomainsWithToolsAndRuntimes(network *NetworkPermissions, tools map[string]any, runtimes map[string]any) string {
	return GetAllowedDomainsForEngine(constants.CopilotEngine, network, tools, runtimes)
}

// GetCodexAllowedDomainsWithToolsAndRuntimes merges Codex default domains with NetworkPermissions, HTTP MCP server domains, and runtime ecosystem domains
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag
func GetCodexAllowedDomainsWithToolsAndRuntimes(network *NetworkPermissions, tools map[string]any, runtimes map[string]any) string {
	return GetAllowedDomainsForEngine(constants.CodexEngine, network, tools, runtimes)
}

// GetClaudeAllowedDomainsWithToolsAndRuntimes merges Claude default domains with NetworkPermissions, HTTP MCP server domains, and runtime ecosystem domains
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag
func GetClaudeAllowedDomainsWithToolsAndRuntimes(network *NetworkPermissions, tools map[string]any, runtimes map[string]any) string {
	return GetAllowedDomainsForEngine(constants.ClaudeEngine, network, tools, runtimes)
}

// GetGeminiAllowedDomainsWithToolsAndRuntimes merges Gemini default domains with NetworkPermissions, HTTP MCP server domains, and runtime ecosystem domains
// Returns a deduplicated, sorted, comma-separated string suitable for AWF's --allow-domains flag
func GetGeminiAllowedDomainsWithToolsAndRuntimes(network *NetworkPermissions, tools map[string]any, runtimes map[string]any) string {
	return GetAllowedDomainsForEngine(constants.GeminiEngine, network, tools, runtimes)
}

// GetBlockedDomains returns the blocked domains from network permissions
// Returns empty slice if no network permissions configured or no domains blocked
// The returned list is sorted and deduplicated
// Supports ecosystem identifiers (same as allowed domains)
func GetBlockedDomains(network *NetworkPermissions) []string {
	if network == nil {
		domainsLog.Print("No network permissions specified, no blocked domains")
		return []string{}
	}

	// Handle empty blocked list
	if len(network.Blocked) == 0 {
		domainsLog.Print("Empty blocked list, no domains blocked")
		return []string{}
	}

	domainsLog.Printf("Processing %d blocked domains/ecosystems", len(network.Blocked))

	// Process the blocked list, expanding ecosystem identifiers if present
	// Use a map to deduplicate domains
	domainMap := make(map[string]bool)
	for _, domain := range network.Blocked {
		// Try to get domains for this ecosystem category
		ecosystemDomains := getEcosystemDomains(domain)
		if len(ecosystemDomains) > 0 {
			// This was an ecosystem identifier, expand it
			domainsLog.Printf("Expanded ecosystem '%s' to %d domains", domain, len(ecosystemDomains))
			for _, d := range ecosystemDomains {
				domainMap[d] = true
			}
		} else {
			// Add the domain as-is (regular domain name)
			domainMap[domain] = true
		}
	}

	// Convert map to sorted slice
	expandedDomains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		expandedDomains = append(expandedDomains, domain)
	}
	sort.Strings(expandedDomains)

	return expandedDomains
}

// formatBlockedDomains formats blocked domains as a comma-separated string suitable for AWF's --block-domains flag
// Returns empty string if no blocked domains
func formatBlockedDomains(network *NetworkPermissions) string {
	if network == nil {
		return ""
	}

	blockedDomains := GetBlockedDomains(network)
	if len(blockedDomains) == 0 {
		return ""
	}

	return strings.Join(blockedDomains, ",")
}

// GetAPITargetDomains returns the set of domains to add to the allow-list when engine.api-target is set.
// For a GHES instance with api-target "api.acme.ghe.com", this returns both the API domain
// ("api.acme.ghe.com") and the base hostname ("acme.ghe.com") so that both the GitHub web UI
// and API requests pass through the firewall without manual lock file edits.
// Returns nil for empty apiTarget.
func GetAPITargetDomains(apiTarget string) []string {
	if apiTarget == "" {
		return nil
	}

	domains := []string{apiTarget}

	// Derive the base hostname by stripping the first subdomain label, but only for
	// API-style hostnames that start with "api.".
	// e.g., "api.acme.ghe.com" → "acme.ghe.com"
	// Only add the base hostname if it still looks like a multi-label hostname (contains a dot).
	if strings.HasPrefix(apiTarget, "api.") {
		if idx := strings.Index(apiTarget, "."); idx > 0 {
			baseHost := apiTarget[idx+1:]
			if strings.Contains(baseHost, ".") && baseHost != apiTarget {
				domains = append(domains, baseHost)
			}
		}
	}

	return domains
}

// mergeAPITargetDomains merges the api-target domains into an existing comma-separated domain string.
// When engine.api-target is set, both the API hostname and its base hostname are added to the allow-list.
// Returns the original string unchanged when apiTarget is empty.
func mergeAPITargetDomains(domainsStr string, apiTarget string) string {
	extraDomains := GetAPITargetDomains(apiTarget)
	if len(extraDomains) == 0 {
		return domainsStr
	}

	domainMap := make(map[string]bool)
	for d := range strings.SplitSeq(domainsStr, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			domainMap[d] = true
		}
	}
	for _, d := range extraDomains {
		domainMap[d] = true
	}

	result := make([]string, 0, len(domainMap))
	for d := range domainMap {
		result = append(result, d)
	}
	sort.Strings(result)
	return strings.Join(result, ",")
}

// computeAllowedDomainsForSanitization computes the allowed domains for sanitization
// based on the engine and network configuration, matching what's provided to the firewall
func (c *Compiler) computeAllowedDomainsForSanitization(data *WorkflowData) string {
	// Determine which engine is being used
	var engineID string
	if data.EngineConfig != nil {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}

	// Compute domains based on engine type, including tools and runtimes to match
	// what's provided to the actual firewall at runtime
	var base string
	switch engineID {
	case "copilot":
		base = GetCopilotAllowedDomainsWithToolsAndRuntimes(data.NetworkPermissions, data.Tools, data.Runtimes)
	case "codex":
		base = GetCodexAllowedDomainsWithToolsAndRuntimes(data.NetworkPermissions, data.Tools, data.Runtimes)
	case "claude":
		base = GetClaudeAllowedDomainsWithToolsAndRuntimes(data.NetworkPermissions, data.Tools, data.Runtimes)
	case "gemini":
		base = GetGeminiAllowedDomainsWithToolsAndRuntimes(data.NetworkPermissions, data.Tools, data.Runtimes)
	default:
		// For other engines, use network permissions only
		domains := GetAllowedDomains(data.NetworkPermissions)
		base = strings.Join(domains, ",")
	}

	// Add GHES/custom API target domains so GH_AW_ALLOWED_DOMAINS stays in sync with --allow-domains
	if data.EngineConfig != nil && data.EngineConfig.APITarget != "" {
		base = mergeAPITargetDomains(base, data.EngineConfig.APITarget)
	}

	return base
}

// expandAllowedDomains expands a list of domain entries (which may include ecosystem
// identifiers like "python", "node", "dev-tools") into a deduplicated, sorted list of
// concrete domain strings. This uses the same expansion logic as network.allowed.
func expandAllowedDomains(entries []string) []string {
	domainMap := make(map[string]bool)
	for _, entry := range entries {
		ecosystemDomains := getEcosystemDomains(entry)
		if len(ecosystemDomains) > 0 {
			for _, d := range ecosystemDomains {
				domainMap[d] = true
			}
		} else {
			domainMap[entry] = true
		}
	}
	result := make([]string, 0, len(domainMap))
	for d := range domainMap {
		result = append(result, d)
	}
	sort.Strings(result)
	return result
}

// computeExpandedAllowedDomainsForSanitization computes the allowed domains for URL sanitization,
// unioning the engine/network base set with the safe-outputs.allowed-domains entries.
// It always includes "localhost" and "github.com" in the result.
// The allowed-domains entries support ecosystem identifiers (same syntax as network.allowed).
func (c *Compiler) computeExpandedAllowedDomainsForSanitization(data *WorkflowData) string {
	// Start from the base set (engine defaults + network.allowed + tools + runtimes)
	base := c.computeAllowedDomainsForSanitization(data)

	domainMap := make(map[string]bool)

	// Seed from the base computation
	if base != "" {
		for d := range strings.SplitSeq(base, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				domainMap[d] = true
			}
		}
	}

	// Union with allowed-domains (expanded)
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		for _, d := range expandAllowedDomains(data.SafeOutputs.AllowedDomains) {
			domainMap[d] = true
		}
	}

	// Always allow localhost (for local development URL references)
	domainMap["localhost"] = true

	// Always allow github.com (GitHub page of the current repo)
	domainMap["github.com"] = true

	// Produce a sorted, comma-separated result
	result := make([]string, 0, len(domainMap))
	for d := range domainMap {
		result = append(result, d)
	}
	sort.Strings(result)
	return strings.Join(result, ",")
}
