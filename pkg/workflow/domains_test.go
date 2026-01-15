package workflow

import (
	"strings"
	"testing"
)

func TestGetDomainEcosystem(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		// Exact matches for defaults ecosystem
		{
			name:     "defaults ecosystem - exact match",
			domain:   "json-schema.org",
			expected: "defaults",
		},
		{
			name:     "defaults ecosystem - ubuntu archive",
			domain:   "archive.ubuntu.com",
			expected: "defaults",
		},
		{
			name:     "defaults ecosystem - digicert",
			domain:   "ocsp.digicert.com",
			expected: "defaults",
		},

		// Container ecosystem exact matches
		{
			name:     "containers ecosystem - ghcr.io",
			domain:   "ghcr.io",
			expected: "containers",
		},
		{
			name:     "containers ecosystem - quay.io",
			domain:   "quay.io",
			expected: "containers",
		},

		// Container ecosystem wildcard matches
		{
			name:     "containers ecosystem - docker.io subdomain",
			domain:   "registry-1.docker.io",
			expected: "containers",
		},
		{
			name:     "containers ecosystem - docker.com subdomain",
			domain:   "hub.docker.com",
			expected: "containers",
		},
		{
			name:     "containers ecosystem - docker.io base domain",
			domain:   "docker.io",
			expected: "containers",
		},

		// Python ecosystem (assuming pypi.org exists)
		{
			name:     "python ecosystem - pypi",
			domain:   "pypi.org",
			expected: "python",
		},

		// Non-matching domain
		{
			name:     "no ecosystem match - custom domain",
			domain:   "example.com",
			expected: "",
		},
		{
			name:     "no ecosystem match - empty string",
			domain:   "",
			expected: "",
		},

		// Edge cases
		{
			name:     "no ecosystem match - partial match should not work",
			domain:   "notdocker.io",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDomainEcosystem(tt.domain)
			if result != tt.expected {
				t.Errorf("GetDomainEcosystem(%q) = %q, expected %q", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestMatchesDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		pattern  string
		expected bool
	}{
		// Exact matches
		{
			name:     "exact match - same string",
			domain:   "example.com",
			pattern:  "example.com",
			expected: true,
		},
		{
			name:     "exact match - github.com",
			domain:   "github.com",
			pattern:  "github.com",
			expected: true,
		},
		{
			name:     "no match - different domains",
			domain:   "example.com",
			pattern:  "different.com",
			expected: false,
		},

		// Wildcard matches with subdomains
		{
			name:     "wildcard match - subdomain of docker.io",
			domain:   "registry-1.docker.io",
			pattern:  "*.docker.io",
			expected: true,
		},
		{
			name:     "wildcard match - multiple levels deep",
			domain:   "a.b.c.docker.io",
			pattern:  "*.docker.io",
			expected: true,
		},
		{
			name:     "wildcard match - base domain without wildcard",
			domain:   "docker.io",
			pattern:  "*.docker.io",
			expected: true,
		},
		{
			name:     "wildcard match - docker.com subdomain",
			domain:   "hub.docker.com",
			pattern:  "*.docker.com",
			expected: true,
		},
		{
			name:     "wildcard match - base domain docker.com",
			domain:   "docker.com",
			pattern:  "*.docker.com",
			expected: true,
		},

		// Wildcard non-matches
		{
			name:     "no wildcard match - wrong domain",
			domain:   "example.com",
			pattern:  "*.docker.io",
			expected: false,
		},
		{
			name:     "no wildcard match - partial suffix",
			domain:   "notdocker.io",
			pattern:  "*.docker.io",
			expected: false,
		},
		{
			name:     "no wildcard match - prefix instead of suffix",
			domain:   "docker.io.example",
			pattern:  "*.docker.io",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty domain and pattern",
			domain:   "",
			pattern:  "",
			expected: true,
		},
		{
			name:     "empty domain with pattern",
			domain:   "",
			pattern:  "example.com",
			expected: false,
		},
		{
			name:     "domain with empty pattern",
			domain:   "example.com",
			pattern:  "",
			expected: false,
		},
		{
			name:     "wildcard with empty base",
			domain:   "example.com",
			pattern:  "*.",
			expected: false,
		},
		{
			name:     "just wildcard",
			domain:   "example.com",
			pattern:  "*",
			expected: false,
		},
		{
			name:     "pattern with only *. matches empty domain",
			domain:   "",
			pattern:  "*.",
			expected: true, // Edge case: suffix is "", domain == suffix returns true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesDomain(tt.domain, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesDomain(%q, %q) = %v, expected %v", tt.domain, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestCopilotDefaultDomains(t *testing.T) {
	// Verify that expected Copilot domains are present
	expectedDomains := []string{
		"api.business.githubcopilot.com",
		"api.enterprise.githubcopilot.com",
		"api.github.com",
		"api.githubcopilot.com",
		"api.individual.githubcopilot.com",
		"github.com",
		"host.docker.internal",
		"raw.githubusercontent.com",
		"registry.npmjs.org",
	}

	// Create a map for O(1) lookups
	domainMap := make(map[string]bool)
	for _, domain := range CopilotDefaultDomains {
		domainMap[domain] = true
	}

	for _, expected := range expectedDomains {
		if !domainMap[expected] {
			t.Errorf("Expected domain %q not found in CopilotDefaultDomains", expected)
		}
	}

	// Verify the count matches (no extra domains)
	if len(CopilotDefaultDomains) != len(expectedDomains) {
		t.Errorf("CopilotDefaultDomains has %d domains, expected %d", len(CopilotDefaultDomains), len(expectedDomains))
	}
}

func TestCodexDefaultDomains(t *testing.T) {
	// Verify that expected Codex domains are present
	expectedDomains := []string{
		"api.openai.com",
		"host.docker.internal",
		"openai.com",
	}

	// Create a map for O(1) lookups
	domainMap := make(map[string]bool)
	for _, domain := range CodexDefaultDomains {
		domainMap[domain] = true
	}

	for _, expected := range expectedDomains {
		if !domainMap[expected] {
			t.Errorf("Expected domain %q not found in CodexDefaultDomains", expected)
		}
	}

	// Verify the count matches (no extra domains)
	if len(CodexDefaultDomains) != len(expectedDomains) {
		t.Errorf("CodexDefaultDomains has %d domains, expected %d", len(CodexDefaultDomains), len(expectedDomains))
	}
}

func TestGetCodexAllowedDomains(t *testing.T) {
	t.Run("nil network permissions returns only defaults", func(t *testing.T) {
		result := GetCodexAllowedDomains(nil)
		// Should contain default Codex domains, sorted
		if result != "api.openai.com,host.docker.internal,openai.com" {
			t.Errorf("Expected 'api.openai.com,host.docker.internal,openai.com', got %q", result)
		}
	})

	t.Run("with network permissions merges domains", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"example.com"},
		}
		result := GetCodexAllowedDomains(network)
		// Should contain both default Codex domains and user-specified domain
		if result != "api.openai.com,example.com,host.docker.internal,openai.com" {
			t.Errorf("Expected 'api.openai.com,example.com,host.docker.internal,openai.com', got %q", result)
		}
	})

	t.Run("deduplicates domains", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"api.openai.com", "example.com"},
		}
		result := GetCodexAllowedDomains(network)
		// api.openai.com should not appear twice
		if result != "api.openai.com,example.com,host.docker.internal,openai.com" {
			t.Errorf("Expected 'api.openai.com,example.com,host.docker.internal,openai.com', got %q", result)
		}
	})

	t.Run("empty allowed list returns only defaults", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{},
		}
		result := GetCodexAllowedDomains(network)
		// Empty allowed list should still return Codex defaults
		if result != "api.openai.com,host.docker.internal,openai.com" {
			t.Errorf("Expected 'api.openai.com,host.docker.internal,openai.com', got %q", result)
		}
	})
}

func TestClaudeDefaultDomains(t *testing.T) {
	// Verify that critical Claude domains are present
	criticalDomains := []string{
		"anthropic.com",
		"api.anthropic.com",
		"statsig.anthropic.com",
		"api.github.com",
		"github.com",
		"host.docker.internal",
		"registry.npmjs.org",
	}

	// Create a map for O(1) lookups
	domainMap := make(map[string]bool)
	for _, domain := range ClaudeDefaultDomains {
		domainMap[domain] = true
	}

	for _, expected := range criticalDomains {
		if !domainMap[expected] {
			t.Errorf("Expected domain %q not found in ClaudeDefaultDomains", expected)
		}
	}

	// Verify minimum count (Claude has many more domains than the critical ones)
	if len(ClaudeDefaultDomains) < len(criticalDomains) {
		t.Errorf("ClaudeDefaultDomains has %d domains, expected at least %d", len(ClaudeDefaultDomains), len(criticalDomains))
	}
}

func TestGetClaudeAllowedDomains(t *testing.T) {
	t.Run("returns Claude defaults when no network permissions", func(t *testing.T) {
		result := GetClaudeAllowedDomains(nil)
		// Should contain Claude default domains
		if !strings.Contains(result, "api.anthropic.com") {
			t.Error("Expected api.anthropic.com in result")
		}
		if !strings.Contains(result, "anthropic.com") {
			t.Error("Expected anthropic.com in result")
		}
	})

	t.Run("merges network permissions with Claude defaults", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"custom.example.com"},
		}
		result := GetClaudeAllowedDomains(network)
		// Should contain both Claude defaults and custom domain
		if !strings.Contains(result, "api.anthropic.com") {
			t.Error("Expected api.anthropic.com in result")
		}
		if !strings.Contains(result, "custom.example.com") {
			t.Error("Expected custom.example.com in result")
		}
	})

	t.Run("domains are sorted", func(t *testing.T) {
		result := GetClaudeAllowedDomains(nil)
		// Should be comma-separated and sorted
		domains := strings.Split(result, ",")
		for i := 1; i < len(domains); i++ {
			if domains[i-1] > domains[i] {
				t.Errorf("Domains not sorted: %s > %s", domains[i-1], domains[i])
				break
			}
		}
	})
}
