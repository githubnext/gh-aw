package workflow

import (
	"strings"
	"testing"
)

func TestNetworkHookGenerator(t *testing.T) {
	generator := &NetworkHookGenerator{}

	t.Run("GenerateNetworkHookWorkflowStepJS", func(t *testing.T) {
		allowedDomains := []string{"api.github.com", "*.trusted.com"}
		step := generator.GenerateNetworkHookWorkflowStepJS(allowedDomains)

		stepStr := strings.Join(step, "\n")

		// Check that the step contains proper YAML structure for JavaScript
		if !strings.Contains(stepStr, "name: Network Permissions Validation") {
			t.Error("Step should have correct name for JS version")
		}
		if !strings.Contains(stepStr, "uses: actions/github-script@v8") {
			t.Error("Step should use github-script action")
		}
		if !strings.Contains(stepStr, "script: |") {
			t.Error("Step should contain script section")
		}

		// Check that JavaScript functions are included
		if !strings.Contains(stepStr, "function extractDomain") {
			t.Error("Step should contain extractDomain function")
		}
		if !strings.Contains(stepStr, "function isDomainAllowed") {
			t.Error("Step should contain isDomainAllowed function")
		}
		if !strings.Contains(stepStr, "async function main") {
			t.Error("Step should contain main function")
		}
	})
}

func TestShouldEnforceNetworkPermissions(t *testing.T) {
	t.Run("nil permissions", func(t *testing.T) {
		if ShouldEnforceNetworkPermissions(nil) {
			t.Error("Should not enforce permissions when nil")
		}
	})

	t.Run("valid permissions with domains", func(t *testing.T) {
		permissions := &NetworkPermissions{
			Allowed: []string{"example.com", "*.trusted.com"},
		}
		if !ShouldEnforceNetworkPermissions(permissions) {
			t.Error("Should enforce permissions when provided")
		}
	})

	t.Run("empty permissions (deny-all)", func(t *testing.T) {
		permissions := &NetworkPermissions{
			Allowed: []string{}, // Empty list means deny-all
		}
		if !ShouldEnforceNetworkPermissions(permissions) {
			t.Error("Should enforce permissions even with empty allowed list (deny-all policy)")
		}
	})
}

func TestGetAllowedDomains(t *testing.T) {
	t.Run("nil permissions", func(t *testing.T) {
		domains := GetAllowedDomains(nil)
		if domains == nil {
			t.Error("Should return default allow-list when permissions are nil")
		}
		if len(domains) == 0 {
			t.Error("Expected default allow-list domains for nil permissions, got empty list")
		}
	})

	t.Run("empty permissions (deny-all)", func(t *testing.T) {
		permissions := &NetworkPermissions{
			Allowed: []string{}, // Empty list means deny-all
		}
		domains := GetAllowedDomains(permissions)
		if domains == nil {
			t.Error("Should return empty slice, not nil, for deny-all policy")
		}
		if len(domains) != 0 {
			t.Errorf("Expected 0 domains for deny-all policy, got %d", len(domains))
		}
	})

	t.Run("valid permissions with domains", func(t *testing.T) {
		permissions := &NetworkPermissions{
			Allowed: []string{"example.com", "*.trusted.com", "api.service.org"},
		}
		domains := GetAllowedDomains(permissions)
		expectedDomains := []string{"example.com", "*.trusted.com", "api.service.org"}
		if len(domains) != len(expectedDomains) {
			t.Fatalf("Expected %d domains, got %d", len(expectedDomains), len(domains))
		}

		for i, expected := range expectedDomains {
			if domains[i] != expected {
				t.Errorf("Expected domain %d to be '%s', got '%s'", i, expected, domains[i])
			}
		}
	})

	t.Run("permissions with 'defaults' in allowed list", func(t *testing.T) {
		permissions := &NetworkPermissions{
			Allowed: []string{"defaults", "good.com"},
		}
		domains := GetAllowedDomains(permissions)

		// Should have all default domains plus "good.com"
		defaultDomains := getEcosystemDomains("defaults")
		expectedTotal := len(defaultDomains) + 1

		if len(domains) != expectedTotal {
			t.Fatalf("Expected %d domains (defaults + good.com), got %d", expectedTotal, len(domains))
		}

		// Check that all default domains are included
		defaultsFound := 0
		goodComFound := false

		for _, domain := range domains {
			if domain == "good.com" {
				goodComFound = true
			}
			// Check if this domain is in the defaults list
			for _, defaultDomain := range defaultDomains {
				if domain == defaultDomain {
					defaultsFound++
					break
				}
			}
		}

		if defaultsFound != len(defaultDomains) {
			t.Errorf("Expected all %d default domains to be included, found %d", len(defaultDomains), defaultsFound)
		}

		if !goodComFound {
			t.Error("Expected 'good.com' to be included in the allowed domains")
		}
	})

	t.Run("permissions with only 'defaults' in allowed list", func(t *testing.T) {
		permissions := &NetworkPermissions{
			Allowed: []string{"defaults"},
		}
		domains := GetAllowedDomains(permissions)
		defaultDomains := getEcosystemDomains("defaults")

		if len(domains) != len(defaultDomains) {
			t.Fatalf("Expected %d domains (just defaults), got %d", len(defaultDomains), len(domains))
		}

		// Check that all default domains are included
		for i, defaultDomain := range defaultDomains {
			if domains[i] != defaultDomain {
				t.Errorf("Expected domain %d to be '%s', got '%s'", i, defaultDomain, domains[i])
			}
		}
	})
}

func TestDeprecatedHasNetworkPermissions(t *testing.T) {
	t.Run("deprecated function always returns false", func(t *testing.T) {
		// Test that the deprecated function always returns false
		if HasNetworkPermissions(nil) {
			t.Error("Deprecated HasNetworkPermissions should always return false")
		}

		config := &EngineConfig{ID: "claude"}
		if HasNetworkPermissions(config) {
			t.Error("Deprecated HasNetworkPermissions should always return false")
		}
	})
}

func TestEngineConfigParsing(t *testing.T) {
	compiler := &Compiler{}

	t.Run("ParseNetworkPermissions", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{"example.com", "*.trusted.com", "api.service.org"},
			},
		}

		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions == nil {
			t.Fatal("Network permissions should not be nil")
		}

		expectedDomains := []string{"example.com", "*.trusted.com", "api.service.org"}
		if len(networkPermissions.Allowed) != len(expectedDomains) {
			t.Fatalf("Expected %d domains, got %d", len(expectedDomains), len(networkPermissions.Allowed))
		}

		for i, expected := range expectedDomains {
			if networkPermissions.Allowed[i] != expected {
				t.Errorf("Expected domain %d to be '%s', got '%s'", i, expected, networkPermissions.Allowed[i])
			}
		}
	})

	t.Run("ParseWithoutNetworkPermissions", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":    "claude",
				"model": "claude-3-5-sonnet-20241022",
			},
		}

		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions != nil {
			t.Error("Network permissions should be nil when not specified")
		}
	})

	t.Run("ParseEmptyNetworkPermissions", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{}, // Empty list means deny-all
			},
		}

		networkPermissions := compiler.extractNetworkPermissions(frontmatter)

		if networkPermissions == nil {
			t.Fatal("Network permissions should not be nil")
		}

		if len(networkPermissions.Allowed) != 0 {
			t.Errorf("Expected 0 domains for deny-all policy, got %d", len(networkPermissions.Allowed))
		}
	})
}
