//go:build integration

package workflow

import (
	"testing"
)

func TestFirewallWithoutAllowedUsesDefaults(t *testing.T) {
	t.Run("firewall: true without allowed should use defaults mode", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"firewall": true,
			},
		}

		compiler := &Compiler{}
		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions == nil {
			t.Fatal("Expected networkPermissions to be parsed, got nil")
		}

		// Check that firewall is enabled
		if networkPermissions.Firewall == nil {
			t.Fatal("Expected firewall config to be present")
		}
		if !networkPermissions.Firewall.Enabled {
			t.Error("Expected firewall to be enabled")
		}

		// Check that mode is set to "defaults"
		if networkPermissions.Mode != "defaults" {
			t.Errorf("Expected mode to be 'defaults', got '%s'", networkPermissions.Mode)
		}

		// Verify that GetAllowedDomains returns default domains
		domains := GetAllowedDomains(networkPermissions)
		defaultDomains := getEcosystemDomains("defaults")

		if len(domains) != len(defaultDomains) {
			t.Errorf("Expected %d default domains, got %d", len(defaultDomains), len(domains))
		}
	})

	t.Run("firewall: true with allowed should use the allowed list", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"firewall": true,
				"allowed":  []any{"custom.com", "api.example.org"},
			},
		}

		compiler := &Compiler{}
		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions == nil {
			t.Fatal("Expected networkPermissions to be parsed, got nil")
		}

		// Check that firewall is enabled
		if networkPermissions.Firewall == nil {
			t.Fatal("Expected firewall config to be present")
		}
		if !networkPermissions.Firewall.Enabled {
			t.Error("Expected firewall to be enabled")
		}

		// Mode should NOT be "defaults" when allowed is specified
		if networkPermissions.Mode == "defaults" {
			t.Error("Mode should not be 'defaults' when allowed list is provided")
		}

		// Verify that allowed list contains the custom domains
		if len(networkPermissions.Allowed) != 2 {
			t.Fatalf("Expected 2 allowed domains, got %d", len(networkPermissions.Allowed))
		}

		if networkPermissions.Allowed[0] != "custom.com" {
			t.Errorf("Expected first allowed domain to be 'custom.com', got '%s'", networkPermissions.Allowed[0])
		}
		if networkPermissions.Allowed[1] != "api.example.org" {
			t.Errorf("Expected second allowed domain to be 'api.example.org', got '%s'", networkPermissions.Allowed[1])
		}
	})

	t.Run("firewall: false without allowed should not set defaults mode", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"firewall": false,
			},
		}

		compiler := &Compiler{}
		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions == nil {
			t.Fatal("Expected networkPermissions to be parsed, got nil")
		}

		// Check that firewall is disabled
		if networkPermissions.Firewall == nil {
			t.Fatal("Expected firewall config to be present")
		}
		if networkPermissions.Firewall.Enabled {
			t.Error("Expected firewall to be disabled")
		}

		// Mode should NOT be "defaults" when firewall is disabled
		if networkPermissions.Mode == "defaults" {
			t.Error("Mode should not be 'defaults' when firewall is disabled")
		}

		// GetAllowedDomains should return empty list (deny-all)
		domains := GetAllowedDomains(networkPermissions)
		if len(domains) != 0 {
			t.Errorf("Expected 0 domains (deny-all), got %d", len(domains))
		}
	})
}
