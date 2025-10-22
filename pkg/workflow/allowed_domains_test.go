package workflow

import (
	"sort"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestComputeAllowedDomainsForSanitization(t *testing.T) {
	t.Run("nil network permissions returns default GitHub domains", func(t *testing.T) {
		data := &WorkflowData{
			NetworkPermissions: nil,
		}

		domains := computeAllowedDomainsForSanitization(data)

		// Should also include ecosystem defaults (from GetAllowedDomains)
		// When nil, GetAllowedDomains returns "defaults" ecosystem
		// Check that we have at least the GitHub defaults
		for _, expected := range constants.DefaultSanitizationDomains {
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
		data := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com", "trusted.org"},
			},
		}

		domains := computeAllowedDomainsForSanitization(data)

		// Should include default GitHub domains plus custom domains
		expectedDomains := append(constants.DefaultSanitizationDomains, "example.com", "trusted.org")

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
		data := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults", "python"},
			},
		}

		domains := computeAllowedDomainsForSanitization(data)

		// Should include default GitHub domains
		for _, expected := range constants.DefaultSanitizationDomains {
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
		data := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Mode: "defaults",
			},
		}

		domains := computeAllowedDomainsForSanitization(data)

		// Should include default GitHub domains
		for _, expected := range constants.DefaultSanitizationDomains {
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
		data := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"github.com", "example.com"},
			},
		}

		domains := computeAllowedDomainsForSanitization(data)

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
		data := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"zebra.com", "alpha.com"},
			},
		}

		domains := computeAllowedDomainsForSanitization(data)

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
