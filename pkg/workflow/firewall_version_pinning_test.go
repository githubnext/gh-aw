package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestAWFInstallationStepDefaultVersion verifies that AWF installation uses the default version when not specified
func TestAWFInstallationStepDefaultVersion(t *testing.T) {
	t.Run("uses default version when no version specified", func(t *testing.T) {
		step := generateAWFInstallationStep("")
		stepStr := strings.Join(step, "\n")

		// Should NOT contain gh release view command
		if strings.Contains(stepStr, "gh release view") {
			t.Error("Should not use dynamic gh release view when default version is available")
		}

		// Should NOT contain LATEST_TAG variable
		if strings.Contains(stepStr, "LATEST_TAG") {
			t.Error("Should not use LATEST_TAG variable when default version is available")
		}

		// Should contain the default version
		expectedVersion := constants.DefaultFirewallVersion
		if !strings.Contains(stepStr, expectedVersion) {
			t.Errorf("Expected to find default version %s in installation step, but it was not found", expectedVersion)
		}

		// Should NOT have env section with GH_TOKEN
		if strings.Contains(stepStr, "GH_TOKEN") {
			t.Error("Should not require GH_TOKEN when using default version")
		}

		// Verify the curl command uses the default version
		expectedURL := "https://github.com/githubnext/gh-aw-firewall/releases/download/" + expectedVersion + "/awf-linux-x64"
		if !strings.Contains(stepStr, expectedURL) {
			t.Errorf("Expected curl command to download from %s", expectedURL)
		}
	})

	t.Run("uses specified version when provided", func(t *testing.T) {
		customVersion := "v0.2.0"
		step := generateAWFInstallationStep(customVersion)
		stepStr := strings.Join(step, "\n")

		// Should contain the custom version
		if !strings.Contains(stepStr, customVersion) {
			t.Errorf("Expected to find custom version %s in installation step", customVersion)
		}

		// Should NOT contain the default version
		if strings.Contains(stepStr, constants.DefaultFirewallVersion) && constants.DefaultFirewallVersion != customVersion {
			t.Error("Should use custom version instead of default version")
		}

		// Verify the curl command uses the custom version
		expectedURL := "https://github.com/githubnext/gh-aw-firewall/releases/download/" + customVersion + "/awf-linux-x64"
		if !strings.Contains(stepStr, expectedURL) {
			t.Errorf("Expected curl command to download from %s", expectedURL)
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

		// Verify it uses the default version
		if !strings.Contains(awfStepStr, constants.DefaultFirewallVersion) {
			t.Errorf("AWF installation step should use default version %s", constants.DefaultFirewallVersion)
		}

		// Verify it doesn't use dynamic LATEST_TAG
		if strings.Contains(awfStepStr, "LATEST_TAG") {
			t.Error("AWF installation should not use dynamic LATEST_TAG")
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

		// Verify it uses the custom version
		if !strings.Contains(awfStepStr, customVersion) {
			t.Errorf("AWF installation step should use custom version %s", customVersion)
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
