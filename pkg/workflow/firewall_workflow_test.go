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

	t.Run("network hook is generated with default domains", func(t *testing.T) {
		engine := NewClaudeEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Should have 5 steps: secret validation, Node.js setup, install, settings, hook
		if len(steps) != 5 {
			t.Errorf("Expected 5 installation steps with network permissions, got %d", len(steps))
		}

		// Check the network permissions hook step (5th step, index 4)
		hookStepStr := strings.Join(steps[4], "\n")
		if !strings.Contains(hookStepStr, "Generate Network Permissions Hook") {
			t.Error("Fifth step should generate network permissions hook")
		}

		// Verify example.com is NOT in the allowed domains
		if strings.Contains(hookStepStr, "\"example.com\"") {
			t.Error("example.com should not be in the allowed domains for firewall workflow")
		}

		// Verify some default domains ARE present
		defaultDomains := []string{"json-schema.org", "archive.ubuntu.com"}
		for _, domain := range defaultDomains {
			if !strings.Contains(hookStepStr, domain) {
				t.Errorf("Expected default domain '%s' to be in allowed domains", domain)
			}
		}
	})

	t.Run("execution step includes settings parameter", func(t *testing.T) {
		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test-log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepYAML := strings.Join(steps[0], "\n")

		// Verify settings parameter is present (required for network permissions)
		if !strings.Contains(stepYAML, "--settings /tmp/gh-aw/.claude/settings.json") {
			t.Error("Settings parameter should be present with network permissions")
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
