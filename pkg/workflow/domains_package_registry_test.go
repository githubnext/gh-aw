//go:build !integration

package workflow

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GitHub Packages is a documented exception, tested separately below.
var packageRegistryDomains = []string{
	"registry.npmjs.org",
	"registry.npmjs.com",
	"pypi.org",
	"files.pythonhosted.org",
}

var githubPackageRegistryDomains = []string{
	"npm.pkg.github.com",
	"nuget.pkg.github.com",
	"maven.pkg.github.com",
	"rubygems.pkg.github.com",
}

// AWF treats both bare domains and leading wildcards as suffix rules. Do not use
// matchesDomain here: that helper intentionally has different classification semantics.
func awfDomainAllowsHost(rule, host string) bool {
	suffix := strings.TrimPrefix(rule, "*.")
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

func awfDomainRulesOverlap(a, b string) bool {
	return awfDomainAllowsHost(a, strings.TrimPrefix(b, "*.")) ||
		awfDomainAllowsHost(b, strings.TrimPrefix(a, "*."))
}

func TestAWFDomainRulesOverlap(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"*.googleapis.com", "storage.googleapis.com", true},
		{"googleapis.com", "maven-central.storage-download.googleapis.com", true},
		{"github.com", "npm.pkg.github.com", true},
		{"api.github.com", "npm.pkg.github.com", false},
		{"generativelanguage.googleapis.com", "storage.googleapis.com", false},
		{"*.pythonhosted.org", "files.pythonhosted.org", true},
		{"*.example.com", "*.nested.example.com", true},
		{"example.com", "*.example.com", true},
		{"example.com", "notexample.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.a+"/"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.want, awfDomainRulesOverlap(tt.a, tt.b))
			assert.Equal(t, tt.want, awfDomainRulesOverlap(tt.b, tt.a))
		})
	}
}

// allEngines is the set of engines whose default domain lists are asserted below.
var allEngines = []constants.EngineName{
	constants.CopilotEngine,
	constants.ClaudeEngine,
	constants.CodexEngine,
	constants.GeminiEngine,
	constants.AgyEngine,
	constants.PiEngine,
}

func domainSet(t *testing.T, csv string) map[string]struct{} {
	t.Helper()
	set := make(map[string]struct{})
	for d := range strings.SplitSeq(csv, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			set[d] = struct{}{}
		}
	}
	return set
}

// TestEngineDefaultDomainsExcludePackageRegistries verifies that selecting an engine alone
// never grants access to npm or PyPI. Engine CLIs and SDKs are installed by trusted runner
// steps outside the AWF sandbox, so registries do not belong in the engine default lists.
func TestEngineDefaultDomainsExcludePackageRegistries(t *testing.T) {
	for _, engine := range allEngines {
		t.Run(string(engine), func(t *testing.T) {
			defaults, err := GetDefaultDomainsForEngine(engine, "")
			require.NoError(t, err)
			for _, registry := range packageRegistryDomains {
				for _, rule := range defaults {
					assert.False(t, awfDomainAllowsHost(rule, registry),
						"engine %q rule %q must not allow package registry %q", engine, rule, registry)
				}
			}
		})
	}

	// Pi's provider-specific and static default lists must be clean too.
	for _, registry := range packageRegistryDomains {
		assert.NotContains(t, PiBaseDefaultDomains, registry)
		assert.NotContains(t, PiDefaultDomains, registry)
	}
}

// TestEngineDefaultDomainsKeepTransportDomains verifies that removing package registries did
// not remove the transport domains each engine genuinely requires in its static defaults.
// Claude's Anthropic API host is added dynamically when Anthropic is the effective provider.
func TestEngineDefaultDomainsKeepTransportDomains(t *testing.T) {
	required := map[constants.EngineName][]string{
		constants.CopilotEngine: {"api.githubcopilot.com", "host.docker.internal"},
		constants.ClaudeEngine:  {"host.docker.internal"},
		constants.CodexEngine:   {"api.openai.com", "host.docker.internal"},
		constants.GeminiEngine:  {"generativelanguage.googleapis.com", "host.docker.internal"},
		constants.PiEngine:      {"api.githubcopilot.com", "host.docker.internal"},
	}

	for engine, domains := range required {
		t.Run(string(engine), func(t *testing.T) {
			defaults, err := GetDefaultDomainsForEngine(engine, "")
			require.NoError(t, err)
			for _, domain := range domains {
				assert.Contains(t, defaults, domain,
					"engine %q must keep required transport domain %q", engine, domain)
			}
		})
	}
}

// Compare effective suffix/wildcard coverage, not just identical strings. Only the
// exact github.com rule may overlap the four documented GitHub Packages hosts.
func TestEngineDefaultDomainsDoNotOverlapEcosystems(t *testing.T) {
	gatedEcosystems := []string{
		"clojure", "dart", "deno", "dotnet", "elixir", "go", "haskell", "java",
		"julia", "kotlin", "latex", "lean", "lua", "node", "node-cdns", "ocaml",
		"perl", "php", "powershell", "python", "python-native", "r", "ruby",
		"rust", "scala", "swift", "terraform", "zig",
	}

	for _, ecosystem := range gatedEcosystems {
		ecosystemDomains := getEcosystemDomains(ecosystem)
		require.NotEmpty(t, ecosystemDomains, "ecosystem %q must resolve to a non-empty domain list", ecosystem)

		for engine, defaults := range engineDefaultDomainSets {
			if engine == "threat-detection" {
				continue // Detection intentionally permits read-only npm lockfile validation.
			}
			t.Run(ecosystem+"/"+engine, func(t *testing.T) {
				for _, domain := range ecosystemDomains {
					for _, rule := range defaults {
						if rule == "github.com" && slices.Contains(githubPackageRegistryDomains, domain) {
							continue
						}
						assert.False(t, awfDomainRulesOverlap(rule, domain),
							"engine %q rule %q overlaps %q ecosystem domain %q; "+
								"package/language ecosystems require explicit opt-in via network.allowed or runtimes",
							engine, rule, ecosystem, domain)
					}
				}
			})
		}
	}
}

func TestGitHubTransportPackageRegistryException(t *testing.T) {
	// Direct repository clone/web access needs github.com. AWF has no exact-host-only
	// spelling, so that capability also permits GitHub Packages without a language opt-in.
	sets := map[string][]string{"github": getEcosystemDomains("github")}
	maps.Copy(sets, engineDefaultDomainSets)
	for name, domains := range sets {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, domains, "github.com", "preserve direct GitHub repository access")
			for _, registry := range githubPackageRegistryDomains {
				assert.True(t, awfDomainAllowsHost("github.com", registry))
				assert.NotContains(t, domains, registry, "do not add registries as transport hosts")
			}
		})
	}

	t.Run("explicit package restriction preserves repository access", func(t *testing.T) {
		network := &NetworkPermissions{Allowed: []string{"github"}, Blocked: []string{"pkg.github.com"}}
		allowed := GetAllowedDomains(network)
		blocked := GetBlockedDomains(network)
		for _, host := range append([]string{"github.com", "api.github.com"}, githubPackageRegistryDomains...) {
			reachable, denied := false, false
			for _, rule := range allowed {
				reachable = reachable || awfDomainAllowsHost(rule, host)
			}
			for _, rule := range blocked {
				denied = denied || awfDomainAllowsHost(rule, host)
			}
			assert.Equal(t, !slices.Contains(githubPackageRegistryDomains, host), reachable && !denied, host)
		}
	})
}

// TestNetworkPermissionsGatePackageRegistries is the end-to-end guarantee: a restrictive
// network configuration excludes package registries, and explicit ecosystem opt-in
// (network.allowed or runtimes) brings them back.
func TestNetworkPermissionsGatePackageRegistries(t *testing.T) {
	tests := []struct {
		name              string
		network           *NetworkPermissions
		runtimes          map[string]any
		expectContains    []string
		expectNotContains []string
	}{
		{
			name:              "deny-all network excludes registries",
			network:           &NetworkPermissions{Allowed: []string{}},
			expectNotContains: packageRegistryDomains,
		},
		{
			name:              "defaults + github excludes registries",
			network:           &NetworkPermissions{Allowed: []string{"defaults", "github"}},
			expectContains:    []string{"github.com"},
			expectNotContains: packageRegistryDomains,
		},
		{
			name:              "nil network excludes registries",
			network:           nil,
			expectNotContains: packageRegistryDomains,
		},
		{
			name:              "explicit node ecosystem includes npm but not pypi",
			network:           &NetworkPermissions{Allowed: []string{"defaults", "node"}},
			expectContains:    []string{"registry.npmjs.org"},
			expectNotContains: []string{"pypi.org", "files.pythonhosted.org"},
		},
		{
			name:              "explicit python ecosystem includes pypi but not npm",
			network:           &NetworkPermissions{Allowed: []string{"defaults", "python"}},
			expectContains:    []string{"pypi.org", "files.pythonhosted.org"},
			expectNotContains: []string{"registry.npmjs.org"},
		},
		{
			name:           "node runtime declaration includes npm",
			network:        &NetworkPermissions{Allowed: []string{"defaults"}},
			runtimes:       map[string]any{"node": map[string]any{"version": "24"}},
			expectContains: []string{"registry.npmjs.org"},
		},
		{
			name:           "python runtime declaration includes pypi",
			network:        &NetworkPermissions{Allowed: []string{"defaults"}},
			runtimes:       map[string]any{"python": map[string]any{"version": "3.12"}},
			expectContains: []string{"pypi.org", "files.pythonhosted.org"},
		},
	}

	for _, tt := range tests {
		for _, engine := range allEngines {
			t.Run(tt.name+"/"+string(engine), func(t *testing.T) {
				result, err := GetAllowedDomainsForEngineWithModel(engine, "", tt.network, nil, tt.runtimes)
				require.NoError(t, err)
				domains := domainSet(t, result)

				for _, expected := range tt.expectContains {
					assert.Contains(t, domains, expected,
						"engine %q should allow %q for %s", engine, expected, tt.name)
				}
				for _, unexpected := range tt.expectNotContains {
					for rule := range domains {
						assert.False(t, awfDomainAllowsHost(rule, unexpected),
							"engine %q rule %q must not allow %q for %s", engine, rule, unexpected, tt.name)
					}
				}
			})
		}
	}
}
