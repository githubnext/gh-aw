package workflow

import (
	"strings"
	"testing"
)

func TestNetworkHookGenerator(t *testing.T) {
	generator := &NetworkHookGenerator{}

	t.Run("GenerateNetworkHookScript", func(t *testing.T) {
		allowedDomains := []string{"example.com", "*.trusted.com", "api.service.org"}
		script := generator.GenerateNetworkHookScript(allowedDomains)

		// Check that script is JavaScript (not Python anymore)
		if !strings.Contains(script, "function extractDomain") {
			t.Error("Script should define extractDomain function in JavaScript")
		}
		if !strings.Contains(script, "function isDomainAllowed") {
			t.Error("Script should define isDomainAllowed function in JavaScript")
		}
		if !strings.Contains(script, "async function main") {
			t.Error("Script should define main async function in JavaScript")
		}
		if !strings.Contains(script, "actions/github-script") {
			t.Error("Script should be designed for actions/github-script")
		}

		// JavaScript should not contain Python-specific imports
		if strings.Contains(script, "import json") {
			t.Error("Script should not contain Python imports")
		}
		if strings.Contains(script, "#!/usr/bin/env python3") {
			t.Error("Script should not be a Python script")
		}
	})

	t.Run("GenerateNetworkHookWorkflowStep", func(t *testing.T) {
		allowedDomains := []string{"api.github.com", "*.trusted.com"}
		step := generator.GenerateNetworkHookWorkflowStep(allowedDomains)

		stepStr := strings.Join(step, "\n")

		// Check that the step uses actions/github-script instead of shell commands
		if !strings.Contains(stepStr, "name: Network Permissions Hook") {
			t.Error("Step should have correct name")
		}
		if !strings.Contains(stepStr, "uses: actions/github-script@v7") {
			t.Error("Step should use actions/github-script@v7")
		}
		if !strings.Contains(stepStr, "GITHUB_AW_NETWORK_DOMAINS") {
			t.Error("Step should set GITHUB_AW_NETWORK_DOMAINS environment variable")
		}

		// Check that domains are included in the environment variable
		if !strings.Contains(stepStr, "api.github.com") {
			t.Error("Step should contain api.github.com domain in env")
		}
		if !strings.Contains(stepStr, "*.trusted.com") {
			t.Error("Step should contain *.trusted.com domain in env")
		}

		// Should not contain old Python-style file creation
		if strings.Contains(stepStr, ".claude/hooks/network_permissions.py") {
			t.Error("Step should not reference Python file creation")
		}
		if strings.Contains(stepStr, "chmod +x") {
			t.Error("Step should not contain chmod commands (using JavaScript now)")
		}
	})

	t.Run("EmptyDomainsGeneration", func(t *testing.T) {
		allowedDomains := []string{} // Empty list means deny-all
		script := generator.GenerateNetworkHookScript(allowedDomains)

		// Should return the JavaScript script (domains are passed via env vars now)
		if !strings.Contains(script, "function extractDomain") {
			t.Error("Script should define JavaScript functions")
		}
		if !strings.Contains(script, "GITHUB_AW_NETWORK_DOMAINS") {
			t.Error("Script should reference environment variable for domains")
		}

		// Test the workflow step with empty domains
		step := generator.GenerateNetworkHookWorkflowStep(allowedDomains)
		stepStr := strings.Join(step, "\n")

		if !strings.Contains(stepStr, `GITHUB_AW_NETWORK_DOMAINS: '[]'`) {
			t.Error("Step should handle empty domains list (deny-all policy) in environment variable")
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
