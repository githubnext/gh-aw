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

		// Verify version is used in the installation step
		if !strings.Contains(stepStr, expectedVersion) {
			t.Errorf("Expected to log requested version %s in installation step, but it was not found", expectedVersion)
		}

		// Verify checksum verification is included
		if !strings.Contains(stepStr, "sha256sum") {
			t.Error("Expected checksum verification using sha256sum")
		}

		if !strings.Contains(stepStr, "checksums.txt") {
			t.Error("Expected to download checksums.txt for verification")
		}

		// Verify direct binary download (not installer script)
		if !strings.Contains(stepStr, "releases/download") {
			t.Error("Expected direct binary download from GitHub releases")
		}

		if !strings.Contains(stepStr, "AWF_BINARY=\"awf-linux-x64\"") {
			t.Error("Expected to download awf-linux-x64 binary")
		}

		// Ensure it's NOT using the unverified installer script
		if strings.Contains(stepStr, "install.sh") {
			t.Error("Should NOT use unverified installer script - use direct binary download with checksum verification")
		}
	})

	t.Run("uses specified version when provided", func(t *testing.T) {
		customVersion := "v0.2.0"
		step := generateAWFInstallationStep(customVersion, nil)
		stepStr := strings.Join(step, "\n")

		// Verify custom version is used
		if !strings.Contains(stepStr, customVersion) {
			t.Errorf("Expected to log custom version %s in installation step", customVersion)
		}

		// Verify checksum verification is included
		if !strings.Contains(stepStr, "sha256sum") {
			t.Error("Expected checksum verification using sha256sum")
		}

		if !strings.Contains(stepStr, "Checksum verification") {
			t.Error("Expected checksum verification message")
		}

		// Ensure it's NOT using the unverified installer script
		if strings.Contains(stepStr, "install.sh") {
			t.Error("Should NOT use unverified installer script - use direct binary download with checksum verification")
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

		// Verify it logs the default version and uses checksum verification
		if !strings.Contains(awfStepStr, string(constants.DefaultFirewallVersion)) {
			t.Errorf("AWF installation step should reference default version %s", string(constants.DefaultFirewallVersion))
		}
		if !strings.Contains(awfStepStr, "releases/download") {
			t.Error("AWF installation should use direct binary download from GitHub releases")
		}
		if !strings.Contains(awfStepStr, "sha256sum") {
			t.Error("AWF installation should include checksum verification")
		}
		// Verify it's NOT using the unverified installer script
		if strings.Contains(awfStepStr, "install.sh") {
			t.Error("AWF installation should NOT use unverified installer script")
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

		// Verify it logs the custom version
		if !strings.Contains(awfStepStr, customVersion) {
			t.Errorf("AWF installation step should use custom version %s", customVersion)
		}

		// Verify checksum verification is included
		if !strings.Contains(awfStepStr, "sha256sum") {
			t.Error("AWF installation should include checksum verification")
		}

		if !strings.Contains(awfStepStr, "releases/download") {
			t.Error("AWF installation should use direct binary download from GitHub releases")
		}

		// Verify it's NOT using the unverified installer script
		if strings.Contains(awfStepStr, "install.sh") {
			t.Error("AWF installation should NOT use unverified installer script")
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
