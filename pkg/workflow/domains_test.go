package workflow

import (
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
