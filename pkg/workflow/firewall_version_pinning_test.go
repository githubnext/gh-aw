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
		expectedCommit := constants.DefaultFirewallInstallerCommit
		expectedChecksum := constants.DefaultFirewallInstallerChecksum

		// Verify version is logged
		if !strings.Contains(stepStr, expectedVersion) {
			t.Errorf("Expected to log requested version %s in installation step, but it was not found", expectedVersion)
		}

		// Verify commit SHA is used (pinned install.sh)
		if !strings.Contains(stepStr, expectedCommit) {
			t.Errorf("Expected installer to use pinned commit %s", expectedCommit)
		}

		// Verify checksum verification is included
		if !strings.Contains(stepStr, expectedChecksum) {
			t.Errorf("Expected installer to verify checksum %s", expectedChecksum)
		}

		// Verify secure download-verify-execute pattern
		if !strings.Contains(stepStr, "sha256sum") {
			t.Error("Expected installer to verify checksum using sha256sum")
		}

		if !strings.Contains(stepStr, "Checksum verification failed") {
			t.Error("Expected installer to include checksum verification error handling")
		}

		// Verify AWF_VERSION is passed to installer
		if !strings.Contains(stepStr, "AWF_VERSION="+expectedVersion) {
			t.Errorf("Expected installer to pass AWF_VERSION=%s", expectedVersion)
		}
	})

	t.Run("uses specified version when provided", func(t *testing.T) {
		customVersion := "v0.2.0"
		step := generateAWFInstallationStep(customVersion, nil)
		stepStr := strings.Join(step, "\n")

		expectedCommit := constants.DefaultFirewallInstallerCommit
		expectedChecksum := constants.DefaultFirewallInstallerChecksum

		// Verify custom version is logged
		if !strings.Contains(stepStr, customVersion) {
			t.Errorf("Expected to log custom version %s in installation step", customVersion)
		}

		// Verify commit SHA is used (pinned install.sh)
		if !strings.Contains(stepStr, expectedCommit) {
			t.Errorf("Expected installer to use pinned commit %s", expectedCommit)
		}

		// Verify checksum verification is included
		if !strings.Contains(stepStr, expectedChecksum) {
			t.Errorf("Expected installer to verify checksum %s", expectedChecksum)
		}

		// Verify AWF_VERSION is passed to installer
		if !strings.Contains(stepStr, "AWF_VERSION="+customVersion) {
			t.Errorf("Expected installer to pass AWF_VERSION=%s", customVersion)
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

		// Verify it logs the default version and uses installer script
		if !strings.Contains(awfStepStr, string(constants.DefaultFirewallVersion)) {
			t.Errorf("AWF installation step should reference default version %s", string(constants.DefaultFirewallVersion))
		}

		// Verify it uses pinned commit SHA (secure pattern)
		expectedCommit := constants.DefaultFirewallInstallerCommit
		if !strings.Contains(awfStepStr, expectedCommit) {
			t.Errorf("AWF installation should use pinned commit %s", expectedCommit)
		}

		// Verify checksum verification is included
		if !strings.Contains(awfStepStr, "sha256sum") {
			t.Error("AWF installation should verify checksum")
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

		// Verify it uses pinned commit SHA (secure pattern)
		expectedCommit := constants.DefaultFirewallInstallerCommit
		if !strings.Contains(awfStepStr, expectedCommit) {
			t.Errorf("AWF installation should use pinned commit %s", expectedCommit)
		}

		// Verify checksum verification is included
		if !strings.Contains(awfStepStr, "sha256sum") {
			t.Error("AWF installation should verify checksum")
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
