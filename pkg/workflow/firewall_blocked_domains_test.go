//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFirewallBlockedDomainsInCopilotEngine tests that blocked domains are included in AWF command
func TestFirewallBlockedDomainsInCopilotEngine(t *testing.T) {
	t.Run("blocked domains are added to AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults", "github"},
				Blocked: []string{"tracker.example.com", "analytics.example.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// With config file support, domains appear in the JSON config (not as CLI flags)
		assert.Contains(t, stepContent, "allowDomains", "Expected command to contain 'allowDomains' in config JSON")

		// Verify blockDomains is present in the JSON config
		assert.Contains(t, stepContent, "blockDomains", "Expected command to contain 'blockDomains' in config JSON")

		// Verify blocked domains are in the command
		assert.Contains(t, stepContent, "analytics.example.com", "Expected command to contain blocked domain")
		assert.Contains(t, stepContent, "tracker.example.com", "Expected command to contain blocked domain")
	})

	t.Run("no blocked domains means no blockDomains field in config JSON", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults", "github"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Verify allowDomains is present
		assert.Contains(t, stepContent, "allowDomains", "Expected command to contain 'allowDomains' in config JSON")

		// Verify blockDomains is NOT present when there are no blocked domains
		assert.NotContains(t, stepContent, "blockDomains", "Expected command to NOT contain 'blockDomains' when no domains are blocked")
	})

	t.Run("ecosystem identifiers are expanded in blocked domains", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults", "github"},
				Blocked: []string{"python"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Verify blockDomains is present in the JSON config
		assert.Contains(t, stepContent, "blockDomains", "Expected command to contain 'blockDomains' in config JSON")

		// Verify that python ecosystem domains are expanded and included
		// Get python domains to verify at least one is present
		pythonDomains := getEcosystemDomains("python")
		assert.NotEmpty(t, pythonDomains, "Python ecosystem should have domains")

		// Check that at least one python domain is in the blocked domains list
		foundPythonDomain := false
		for _, domain := range pythonDomains {
			if strings.Contains(stepContent, domain) {
				foundPythonDomain = true
				break
			}
		}
		assert.True(t, foundPythonDomain, "Expected at least one Python ecosystem domain in blocked domains")
	})
}

// TestFirewallBlockedDomainsInClaudeEngine tests that blocked domains work with Claude engine
func TestFirewallBlockedDomainsInClaudeEngine(t *testing.T) {
	t.Run("blocked domains are added to Claude AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults"},
				Blocked: []string{"tracker.example.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		assert.NotEmpty(t, steps, "Expected at least one execution step")

		stepContent := strings.Join(steps[0], "\n")

		// Verify blockDomains is present in the JSON config
		assert.Contains(t, stepContent, "blockDomains", "Expected command to contain 'blockDomains' in config JSON")
		assert.Contains(t, stepContent, "tracker.example.com", "Expected command to contain blocked domain")
	})
}

// TestFirewallBlockedDomainsInCodexEngine tests that blocked domains work with Codex engine
func TestFirewallBlockedDomainsInCodexEngine(t *testing.T) {
	t.Run("blocked domains are added to Codex AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults"},
				Blocked: []string{"tracker.example.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		assert.NotEmpty(t, steps, "Expected at least one execution step")

		stepContent := strings.Join(steps[0], "\n")

		// Verify blockDomains is present in the JSON config
		assert.Contains(t, stepContent, "blockDomains", "Expected command to contain 'blockDomains' in config JSON")
		assert.Contains(t, stepContent, "tracker.example.com", "Expected command to contain blocked domain")
	})
}
