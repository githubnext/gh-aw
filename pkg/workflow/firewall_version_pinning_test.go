package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestAWFInstallationStepDefaultVersion verifies that AWF installation uses the default version when not specified
func TestAWFInstallationStepDefaultVersion(t *testing.T) {
	t.Run("uses default version when no version specified", func(t *testing.T) {
		step := generateAWFInstallationStep("", nil)
		stepStr := strings.Join(step, "\n")

		expectedVersion := string(constants.DefaultFirewallVersion)

		// Check for composite action usage
		if !strings.Contains(stepStr, "uses: githubnext/gh-aw-firewall@main") {
			t.Error("Expected to use composite action githubnext/gh-aw-firewall@main")
		}

		// Check version is passed as input
		if !strings.Contains(stepStr, "version: "+expectedVersion) {
			t.Errorf("Expected to pass version %s as input to composite action", expectedVersion)
		}
	})

	t.Run("uses specified version when provided", func(t *testing.T) {
		customVersion := "v0.2.0"
		step := generateAWFInstallationStep(customVersion, nil)
		stepStr := strings.Join(step, "\n")

		// Check for composite action usage
		if !strings.Contains(stepStr, "uses: githubnext/gh-aw-firewall@main") {
			t.Error("Expected to use composite action githubnext/gh-aw-firewall@main")
		}

		// Check custom version is passed as input
		if !strings.Contains(stepStr, "version: "+customVersion) {
			t.Errorf("Expected to pass custom version %s as input to composite action", customVersion)
		}
	})
}

// TestCopilotEngineFirewallInstallation verifies that Copilot engine includes AWF installation when firewall is enabled
func TestCopilotEngineFirewallInstallation(t *testing.T) {
	t.Run("includes AWF installation step when firewall enabled", func(t *testing.T) {
		engine := NewCopilotEngine()
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)

		// Find the AWF installation step
		var foundAWFStep bool
		var awfStepStr string
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				foundAWFStep = true
				awfStepStr = stepStr
				break
			}
		}

		if !foundAWFStep {
			t.Fatal("Expected to find AWF installation step when firewall is enabled")
		}

		// Verify it uses the composite action
		if !strings.Contains(awfStepStr, "uses: githubnext/gh-aw-firewall@main") {
			t.Error("AWF installation should use composite action githubnext/gh-aw-firewall@main")
		}

		// Verify it passes the default version
		if !strings.Contains(awfStepStr, "version: "+string(constants.DefaultFirewallVersion)) {
			t.Errorf("AWF installation step should pass default version %s", string(constants.DefaultFirewallVersion))
		}
	})

	t.Run("uses custom version when specified in firewall config", func(t *testing.T) {
		engine := NewCopilotEngine()
		customVersion := "v0.3.0"
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: customVersion,
				},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)

		// Find the AWF installation step
		var foundAWFStep bool
		var awfStepStr string
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				foundAWFStep = true
				awfStepStr = stepStr
				break
			}
		}

		if !foundAWFStep {
			t.Fatal("Expected to find AWF installation step when firewall is enabled")
		}

		// Verify it uses the composite action
		if !strings.Contains(awfStepStr, "uses: githubnext/gh-aw-firewall@main") {
			t.Error("AWF installation should use composite action githubnext/gh-aw-firewall@main")
		}

		// Verify it passes the custom version
		if !strings.Contains(awfStepStr, "version: "+customVersion) {
			t.Errorf("AWF installation step should pass custom version %s", customVersion)
		}
	})

	t.Run("does not include AWF installation when firewall disabled", func(t *testing.T) {
		engine := NewCopilotEngine()
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: false,
				},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)

		// Should NOT find the AWF installation step
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				t.Error("Should not include AWF installation step when firewall is disabled")
			}
		}
	})
}
