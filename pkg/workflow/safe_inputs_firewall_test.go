package workflow

import (
	"strings"
	"testing"
)

// TestSafeInputsWithFirewallIncludesHostDockerInternal tests that host.docker.internal
// is added to allowed domains when safe-inputs is enabled with firewall
func TestSafeInputsWithFirewallIncludesHostDockerInternal(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
			Allowed: []string{"github.com"},
		},
		SafeInputs: &SafeInputsConfig{
			Tools: map[string]*SafeInputToolConfig{
				"test-tool": {
					Name:        "test-tool",
					Description: "A test tool",
					Script:      "return 'test';",
				},
			},
		},
	}

	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(workflowData, "test.log")

	if len(steps) == 0 {
		t.Fatal("Expected at least one execution step")
	}

	stepContent := strings.Join(steps[0], "\n")

	// Verify that host.docker.internal is in the allowed domains
	if !strings.Contains(stepContent, "host.docker.internal") {
		t.Error("Expected firewall command to include 'host.docker.internal' when safe-inputs is enabled")
	}

	// Verify the firewall command structure
	if !strings.Contains(stepContent, "--allow-domains") {
		t.Error("Expected command to contain '--allow-domains'")
	}
}

// TestGetCopilotAllowedDomainsWithSafeInputs tests the domain calculation function
func TestGetCopilotAllowedDomainsWithSafeInputs(t *testing.T) {
	t.Run("includes host.docker.internal when safe-inputs enabled", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"github.com"},
		}

		result := GetCopilotAllowedDomainsWithSafeInputs(network, true)

		if !strings.Contains(result, "host.docker.internal") {
			t.Errorf("Expected result to contain 'host.docker.internal', got: %s", result)
		}

		if !strings.Contains(result, "github.com") {
			t.Errorf("Expected result to contain 'github.com', got: %s", result)
		}
	})

	t.Run("does not include host.docker.internal when safe-inputs disabled", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"github.com"},
		}

		result := GetCopilotAllowedDomainsWithSafeInputs(network, false)

		if strings.Contains(result, "host.docker.internal") {
			t.Errorf("Expected result to NOT contain 'host.docker.internal', got: %s", result)
		}

		if !strings.Contains(result, "github.com") {
			t.Errorf("Expected result to contain 'github.com', got: %s", result)
		}
	})

	t.Run("backward compatibility with GetCopilotAllowedDomains", func(t *testing.T) {
		network := &NetworkPermissions{
			Allowed: []string{"github.com"},
		}

		result := GetCopilotAllowedDomains(network)

		// Should not include host.docker.internal by default
		if strings.Contains(result, "host.docker.internal") {
			t.Errorf("Expected backward compatible function to NOT contain 'host.docker.internal', got: %s", result)
		}
	})
}
