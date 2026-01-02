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
		expectedInstaller := "curl -sSL https://raw.githubusercontent.com/githubnext/gh-aw-firewall/main/install.sh | sudo AWF_VERSION=" + expectedVersion + " bash"

		if !strings.Contains(stepStr, expectedInstaller) {
			t.Errorf("Expected installer one-liner: %s", expectedInstaller)
		}

		if !strings.Contains(stepStr, expectedVersion) {
			t.Errorf("Expected to log requested version %s in installation step, but it was not found", expectedVersion)
		}
	})

	t.Run("uses specified version when provided", func(t *testing.T) {
		customVersion := "v0.2.0"
		step := generateAWFInstallationStep(customVersion, nil)
		stepStr := strings.Join(step, "\n")

		expectedInstaller := "curl -sSL https://raw.githubusercontent.com/githubnext/gh-aw-firewall/main/install.sh | sudo AWF_VERSION=" + customVersion + " bash"

		if !strings.Contains(stepStr, customVersion) {
			t.Errorf("Expected to log custom version %s in installation step", customVersion)
		}

		if !strings.Contains(stepStr, expectedInstaller) {
			t.Errorf("Expected installer one-liner: %s", expectedInstaller)
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

		// Verify it logs the default version and uses installer script
		if !strings.Contains(awfStepStr, string(constants.DefaultFirewallVersion)) {
			t.Errorf("AWF installation step should reference default version %s", string(constants.DefaultFirewallVersion))
		}
		if !strings.Contains(awfStepStr, "raw.githubusercontent.com/githubnext/gh-aw-firewall/main/install.sh") {
			t.Error("AWF installation should use the installer script")
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

		// Verify it logs the custom version
		if !strings.Contains(awfStepStr, customVersion) {
			t.Errorf("AWF installation step should use custom version %s", customVersion)
		}

		if !strings.Contains(awfStepStr, "raw.githubusercontent.com/githubnext/gh-aw-firewall/main/install.sh") {
			t.Error("AWF installation should use the installer script")
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
