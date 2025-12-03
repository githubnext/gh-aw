package workflow

import (
	"strings"
	"testing"
)

// TestFirewallWorkflowNetworkConfiguration verifies that the firewall workflow
// is properly configured to block access to example.com
func TestFirewallWorkflowNetworkConfiguration(t *testing.T) {
	// Create workflow data with network defaults and web-fetch tool
	workflowData := &WorkflowData{
		Name: "firewall",
		EngineConfig: &EngineConfig{
			ID:    "claude",
			Model: "claude-3-5-sonnet-20241022",
		},
		NetworkPermissions: &NetworkPermissions{
			Mode: "defaults",
		},
		Tools: map[string]any{
			"web-fetch": nil,
		},
	}

	t.Run("example.com is not in default allowed domains", func(t *testing.T) {
		allowedDomains := GetAllowedDomains(workflowData.NetworkPermissions)
		for _, domain := range allowedDomains {
			if domain == "example.com" {
				t.Error("example.com should not be in the default allowed domains list")
			}
		}
	})

	t.Run("no AWF installation with defaults mode (no firewall needed)", func(t *testing.T) {
		engine := NewClaudeEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Should have 3 steps: secret validation, Node.js setup, install
		// No AWF installation since "defaults" mode doesn't require firewall
		if len(steps) != 3 {
			t.Errorf("Expected 3 installation steps with defaults mode (no firewall), got %d", len(steps))
		}

		// Verify no AWF installation
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				t.Error("AWF should not be installed with defaults network mode")
			}
		}
	})

	t.Run("execution step does not include AWF wrapper with defaults mode", func(t *testing.T) {
		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test-log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepYAML := strings.Join(steps[0], "\n")

		// Verify AWF wrapper is NOT present (defaults mode doesn't require firewall)
		if strings.Contains(stepYAML, "sudo -E awf") {
			t.Error("AWF wrapper should not be present with defaults network mode")
		}

		// Verify settings parameter is NOT present (we use AWF now, not settings)
		if strings.Contains(stepYAML, "--settings") {
			t.Error("Settings parameter should not be present (Claude now uses AWF, not settings)")
		}
	})
}

// TestFirewallWorkflowCompilation verifies the firewall workflow compiles correctly
func TestFirewallWorkflowCompilation(t *testing.T) {
	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
		},
		"engine":  "claude",
		"network": "defaults",
		"tools": map[string]any{
			"web-fetch": nil,
		},
		"timeout_minutes": 5,
	}

	// Create compiler
	c := NewCompiler(false, "", "firewall")
	c.SetSkipValidation(true)

	// Extract and verify tools
	tools := extractToolsFromFrontmatter(frontmatter)
	if _, exists := tools["web-fetch"]; !exists {
		t.Error("web-fetch tool should be present in firewall workflow")
	}

	// Verify network permissions
	networkPerms := c.extractNetworkPermissions(frontmatter)
	if networkPerms == nil {
		t.Fatal("Network permissions should be configured")
	}

	// Verify it's using defaults mode
	if networkPerms.Mode != "defaults" {
		t.Errorf("Expected network mode 'defaults', got '%s'", networkPerms.Mode)
	}

	// Get the actual allowed domains using the GetAllowedDomains function
	allowedDomains := GetAllowedDomains(networkPerms)
	if len(allowedDomains) == 0 {
		t.Error("Default network permissions should have allowed domains")
	}

	// Verify example.com is not in the allowed list
	for _, domain := range allowedDomains {
		if domain == "example.com" {
			t.Error("example.com should not be in the allowed domains")
		}
	}
}

// TestFirewallWorkflowBlocksExampleCom tests that the network hook would block example.com
func TestFirewallWorkflowBlocksExampleCom(t *testing.T) {
	networkPerms := &NetworkPermissions{
		Mode: "defaults",
	}
	allowedDomains := GetAllowedDomains(networkPerms)

	// Create a simple function to check if domain would be allowed
	isDomainAllowed := func(domain string, allowedList []string) bool {
		for _, allowed := range allowedList {
			if allowed == domain {
				return true
			}
			// Check wildcard patterns
			if strings.HasPrefix(allowed, "*.") {
				suffix := allowed[2:]
				if strings.HasSuffix(domain, suffix) {
					return true
				}
			}
		}
		return false
	}

	// Test that example.com is blocked
	if isDomainAllowed("example.com", allowedDomains) {
		t.Error("example.com should be blocked by default network permissions")
	}

	// Test that some infrastructure domains are allowed
	infrastructureDomains := []string{
		"json-schema.org",
		"archive.ubuntu.com",
		"ocsp.digicert.com",
	}

	for _, domain := range infrastructureDomains {
		if !isDomainAllowed(domain, allowedDomains) {
			t.Errorf("Infrastructure domain '%s' should be allowed by default network permissions", domain)
		}
	}
}
