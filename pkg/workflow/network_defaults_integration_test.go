//go:build integration

package workflow

import (
	"testing"
)

func TestNetworkDefaultsIntegration(t *testing.T) {
	t.Run("YAML with defaults in allowed list", func(t *testing.T) {
		// Test the complete workflow: YAML parsing -> GetAllowedDomains
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{"defaults", "good.com", "api.example.org"},
			},
		}

		compiler := &Compiler{}
		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions == nil {
			t.Fatal("Expected networkPermissions to be parsed, got nil")
		}

		// Check that the allowed list contains the original entries
		expectedAllowed := []string{"defaults", "good.com", "api.example.org"}
		if len(networkPermissions.Allowed) != len(expectedAllowed) {
			t.Fatalf("Expected %d allowed entries, got %d", len(expectedAllowed), len(networkPermissions.Allowed))
		}

		for i, expected := range expectedAllowed {
			if networkPermissions.Allowed[i] != expected {
				t.Errorf("Expected allowed[%d] to be '%s', got '%s'", i, expected, networkPermissions.Allowed[i])
			}
		}

		// Now test that GetAllowedDomains expands "defaults" correctly
		domains := GetAllowedDomains(networkPermissions)
		defaultDomains := getEcosystemDomains("defaults")

		// Should have all default domains plus the 2 custom ones
		expectedTotal := len(defaultDomains) + 2
		if len(domains) != expectedTotal {
			t.Fatalf("Expected %d total domains (defaults + 2 custom), got %d", expectedTotal, len(domains))
		}

		// Verify that the default domains are included
		defaultsFound := 0
		goodComFound := false
		apiExampleFound := false

		for _, domain := range domains {
			switch domain {
			case "good.com":
				goodComFound = true
			case "api.example.org":
				apiExampleFound = true
			default:
				// Check if this is a default domain
				for _, defaultDomain := range defaultDomains {
					if domain == defaultDomain {
						defaultsFound++
						break
					}
				}
			}
		}

		if defaultsFound != len(defaultDomains) {
			t.Errorf("Expected all %d default domains to be included, found %d", len(defaultDomains), defaultsFound)
		}

		if !goodComFound {
			t.Error("Expected 'good.com' to be included in the expanded domains")
		}

		if !apiExampleFound {
			t.Error("Expected 'api.example.org' to be included in the expanded domains")
		}
	})

	t.Run("YAML with only defaults", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{"defaults"},
			},
		}

		compiler := &Compiler{}
		networkPermissions := compiler.extractNetworkPermissions(frontmatter)
		domains := GetAllowedDomains(networkPermissions)
		defaultDomains := getEcosystemDomains("defaults")

		if len(domains) != len(defaultDomains) {
			t.Fatalf("Expected %d domains (just defaults), got %d", len(defaultDomains), len(domains))
		}

		// Verify all defaults are included
		for i, defaultDomain := range defaultDomains {
			if domains[i] != defaultDomain {
				t.Errorf("Expected domain %d to be '%s', got '%s'", i, defaultDomain, domains[i])
			}
		}
	})

	t.Run("YAML without defaults should work as before", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{"custom1.com", "custom2.org"},
			},
		}

		compiler := &Compiler{}
		networkPermissions := compiler.extractNetworkPermissions(frontmatter)
		domains := GetAllowedDomains(networkPermissions)

		// Should only have the 2 custom domains
		if len(domains) != 2 {
			t.Fatalf("Expected 2 domains, got %d", len(domains))
		}

		if domains[0] != "custom1.com" {
			t.Errorf("Expected first domain to be 'custom1.com', got '%s'", domains[0])
		}

		if domains[1] != "custom2.org" {
			t.Errorf("Expected second domain to be 'custom2.org', got '%s'", domains[1])
		}
	})
}
