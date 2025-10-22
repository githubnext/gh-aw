package workflow

import (
	"sort"
	"testing"
)

func TestComputeAllowedDomainsForSanitization(t *testing.T) {
	// Expected GitHub sanitization domains (now in defaults ecosystem)
	expectedGitHubDomains := []string{
		"github.com",
		"github.io",
		"githubusercontent.com",
		"githubassets.com",
		"github.dev",
		"codespaces.new",
	}

	t.Run("nil network permissions returns default GitHub domains", func(t *testing.T) {
		domains := ComputeAllowedDomainsForSanitization(nil)

		// Should also include ecosystem defaults (from GetAllowedDomains)
		// When nil, GetAllowedDomains returns "defaults" ecosystem
		// Check that we have at least the GitHub defaults
		for _, expected := range expectedGitHubDomains {
			found := false
			for _, domain := range domains {
				if domain == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected domain '%s' not found in result", expected)
			}
		}
	})

	t.Run("network permissions with custom domains combines with defaults", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Allowed: []string{"example.com", "trusted.org"},
		}

		domains := ComputeAllowedDomainsForSanitization(networkPermissions)

		// Should include custom domains
		expectedDomains := []string{"example.com", "trusted.org"}

		for _, expected := range expectedDomains {
			found := false
			for _, domain := range domains {
				if domain == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected domain '%s' not found in result", expected)
			}
		}
	})

	t.Run("network permissions with ecosystem identifiers expands correctly", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Allowed: []string{"defaults", "python"},
		}

		domains := ComputeAllowedDomainsForSanitization(networkPermissions)

		// Should include default GitHub domains (now part of defaults)
		for _, expected := range expectedGitHubDomains {
			found := false
			for _, domain := range domains {
				if domain == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected default domain '%s' not found in result", expected)
			}
		}

		// Should also include ecosystem domains from "python"
		// Check for at least one python ecosystem domain
		pythonDomains := getEcosystemDomains("python")
		if len(pythonDomains) > 0 {
			found := false
			for _, domain := range domains {
				if domain == pythonDomains[0] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected python ecosystem domain '%s' not found in result", pythonDomains[0])
			}
		}

		// Should also include ecosystem defaults
		defaultEcoDomains := getEcosystemDomains("defaults")
		if len(defaultEcoDomains) > 0 {
			found := false
			for _, domain := range domains {
				if domain == defaultEcoDomains[0] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected defaults ecosystem domain '%s' not found in result", defaultEcoDomains[0])
			}
		}
	})

	t.Run("network permissions with defaults mode includes ecosystem defaults", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Mode: "defaults",
		}

		domains := ComputeAllowedDomainsForSanitization(networkPermissions)

		// Should include default GitHub domains (now part of defaults)
		for _, expected := range expectedGitHubDomains {
			found := false
			for _, domain := range domains {
				if domain == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected default domain '%s' not found in result", expected)
			}
		}

		// Should also include ecosystem defaults
		defaultEcoDomains := getEcosystemDomains("defaults")
		if len(defaultEcoDomains) > 0 {
			found := false
			for _, domain := range domains {
				if domain == defaultEcoDomains[0] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected defaults ecosystem domain '%s' not found in result", defaultEcoDomains[0])
			}
		}
	})

	t.Run("removes duplicate domains", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Allowed: []string{"github.com", "example.com"},
		}

		domains := ComputeAllowedDomainsForSanitization(networkPermissions)

		// Count occurrences of github.com
		count := 0
		for _, domain := range domains {
			if domain == "github.com" {
				count++
			}
		}

		if count != 1 {
			t.Errorf("Expected github.com to appear once, but found %d occurrences", count)
		}
	})

	t.Run("returns sorted domains for consistency", func(t *testing.T) {
		networkPermissions := &NetworkPermissions{
			Allowed: []string{"zebra.com", "alpha.com"},
		}

		domains := ComputeAllowedDomainsForSanitization(networkPermissions)

		// Check if domains are sorted
		sortedDomains := make([]string, len(domains))
		copy(sortedDomains, domains)
		sort.Strings(sortedDomains)

		for i, domain := range domains {
			if domain != sortedDomains[i] {
				t.Errorf("Domains are not sorted: expected %v, got %v", sortedDomains, domains)
				break
			}
		}
	})
}
