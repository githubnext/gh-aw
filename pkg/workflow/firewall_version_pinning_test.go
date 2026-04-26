//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestAWFInstallationStepDefaultVersion verifies that AWF installation uses the default version when not specified
func TestAWFInstallationStepDefaultVersion(t *testing.T) {
	t.Run("uses default version when no version specified", func(t *testing.T) {
		step := generateAWFInstallationStep("", nil)
		stepStr := strings.Join(step, "\n")

		expectedVersion := string(constants.DefaultFirewallVersion)

		// Verify version is passed to the installation script
		if !strings.Contains(stepStr, expectedVersion) {
			t.Errorf("Expected to pass version %s to installation script, but it was not found", expectedVersion)
		}

		// Verify it calls the install_awf_binary.sh script
		if !strings.Contains(stepStr, "install_awf_binary.sh") {
			t.Error("Expected to call install_awf_binary.sh script")
		}

		// Verify it uses the script from ${RUNNER_TEMP}/gh-aw/actions/
		if !strings.Contains(stepStr, "${RUNNER_TEMP}/gh-aw/actions/install_awf_binary.sh") {
			t.Error("Expected to call script from ${RUNNER_TEMP}/gh-aw/actions/ directory")
		}

		// Ensure it's NOT using inline bash or the old unverified installer script
		if strings.Contains(stepStr, "raw.githubusercontent.com") {
			t.Error("Should NOT download installer script from raw.githubusercontent.com")
		}
	})

	t.Run("uses specified version when provided", func(t *testing.T) {
		customVersion := "v0.2.0"
		step := generateAWFInstallationStep(customVersion, nil)
		stepStr := strings.Join(step, "\n")

		// Verify custom version is passed to the script
		if !strings.Contains(stepStr, customVersion) {
			t.Errorf("Expected to pass custom version %s to installation script", customVersion)
		}

		// Verify it calls the install_awf_binary.sh script
		if !strings.Contains(stepStr, "install_awf_binary.sh") {
			t.Error("Expected to call install_awf_binary.sh script")
		}

		// Ensure it's NOT using the old unverified installer pattern
		if strings.Contains(stepStr, "raw.githubusercontent.com") {
			t.Error("Should NOT download installer script from raw.githubusercontent.com")
		}
	})
}

// TestGenerateAWFInstallationSteps verifies the multi-step AWF installation includes caching
func TestGenerateAWFInstallationSteps(t *testing.T) {
	t.Run("returns two steps: cache restore and install", func(t *testing.T) {
		steps := generateAWFInstallationSteps("", nil)

		if len(steps) != 2 {
			t.Fatalf("Expected 2 steps (cache restore + install), got %d", len(steps))
		}

		cacheStepStr := strings.Join(steps[0], "\n")
		installStepStr := strings.Join(steps[1], "\n")

		// Cache step should use actions/cache
		if !strings.Contains(cacheStepStr, "actions/cache") {
			t.Error("Cache step should use actions/cache action")
		}

		// Cache step should have an id for output reference
		if !strings.Contains(cacheStepStr, "id: awf-cache") {
			t.Error("Cache step should have id: awf-cache")
		}

		// Cache key should include runner os+arch and version
		expectedVersion := string(constants.DefaultFirewallVersion)
		if !strings.Contains(cacheStepStr, expectedVersion) {
			t.Errorf("Cache key should include version %s", expectedVersion)
		}
		if !strings.Contains(cacheStepStr, "runner.os") {
			t.Error("Cache key should include runner.os")
		}
		if !strings.Contains(cacheStepStr, "runner.arch") {
			t.Error("Cache key should include runner.arch")
		}

		// Cache step should have restore-keys for stale fallback
		if !strings.Contains(cacheStepStr, "restore-keys") {
			t.Error("Cache step should have restore-keys for stale-version fallback")
		}

		// Install step should reference cache hit output
		if !strings.Contains(installStepStr, "awf-cache.outputs.cache-hit") {
			t.Error("Install step should reference cache hit output from cache step")
		}

		// Install step should set AWF_CACHE_DIR env var
		if !strings.Contains(installStepStr, "AWF_CACHE_DIR") {
			t.Error("Install step should set AWF_CACHE_DIR env var")
		}

		// Install step should still call the install script
		if !strings.Contains(installStepStr, "install_awf_binary.sh") {
			t.Error("Install step should call install_awf_binary.sh")
		}

		if !strings.Contains(installStepStr, expectedVersion) {
			t.Errorf("Install step should pass version %s to script", expectedVersion)
		}
	})

	t.Run("returns nil when custom command specified", func(t *testing.T) {
		steps := generateAWFInstallationSteps("", &AgentSandboxConfig{Command: "custom-awf"})

		if len(steps) != 0 {
			t.Errorf("Expected 0 steps when custom command specified, got %d", len(steps))
		}
	})

	t.Run("cache key uses provided version", func(t *testing.T) {
		customVersion := "v0.99.0"
		steps := generateAWFInstallationSteps(customVersion, nil)

		if len(steps) != 2 {
			t.Fatalf("Expected 2 steps, got %d", len(steps))
		}

		cacheStepStr := strings.Join(steps[0], "\n")
		installStepStr := strings.Join(steps[1], "\n")

		if !strings.Contains(cacheStepStr, customVersion) {
			t.Errorf("Cache key should contain version %s", customVersion)
		}
		if !strings.Contains(installStepStr, customVersion) {
			t.Errorf("Install step should pass version %s to script", customVersion)
		}
	})
}

// TestCopilotEngineFirewallInstallation verifies that Copilot engine includes AWF installation when firewall is enabled
func TestCopilotEngineFirewallInstallation(t *testing.T) {
	t.Run("includes AWF cache and installation steps when firewall enabled", func(t *testing.T) {
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
		var foundCacheStep bool
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install AWF binary") {
				foundAWFStep = true
				awfStepStr = stepStr
			}
			if strings.Contains(stepStr, "Restore AWF binary from cache") {
				foundCacheStep = true
			}
		}

		if !foundAWFStep {
			t.Fatal("Expected to find AWF installation step when firewall is enabled")
		}

		if !foundCacheStep {
			t.Error("Expected to find AWF cache restore step when firewall is enabled")
		}

		// Verify it passes the default version to the script
		if !strings.Contains(awfStepStr, string(constants.DefaultFirewallVersion)) {
			t.Errorf("AWF installation step should pass default version %s to script", string(constants.DefaultFirewallVersion))
		}
		// Verify it calls the install_awf_binary.sh script
		if !strings.Contains(awfStepStr, "install_awf_binary.sh") {
			t.Error("AWF installation should call install_awf_binary.sh script")
		}
		// Verify it's NOT using the old unverified installer script pattern
		if strings.Contains(awfStepStr, "raw.githubusercontent.com") {
			t.Error("AWF installation should NOT download from raw.githubusercontent.com")
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
			if strings.Contains(stepStr, "Install AWF binary") {
				foundAWFStep = true
				awfStepStr = stepStr
				break
			}
		}

		if !foundAWFStep {
			t.Fatal("Expected to find AWF installation step when firewall is enabled")
		}

		// Verify it passes the custom version to the script
		if !strings.Contains(awfStepStr, customVersion) {
			t.Errorf("AWF installation step should pass custom version %s to script", customVersion)
		}

		// Verify it calls the install_awf_binary.sh script
		if !strings.Contains(awfStepStr, "install_awf_binary.sh") {
			t.Error("AWF installation should call install_awf_binary.sh script")
		}

		// Verify it's NOT using the old unverified installer script pattern
		if strings.Contains(awfStepStr, "raw.githubusercontent.com") {
			t.Error("AWF installation should NOT download from raw.githubusercontent.com")
		}
	})

	t.Run("uses sandbox.agent.version when firewall version is not specified", func(t *testing.T) {
		engine := NewCopilotEngine()
		sandboxAgentVersion := "v0.30.2"
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
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type:    SandboxTypeAWF,
					Version: sandboxAgentVersion,
				},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)

		var foundAWFStep bool
		var awfStepStr string
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install AWF binary") {
				foundAWFStep = true
				awfStepStr = stepStr
				break
			}
		}

		if !foundAWFStep {
			t.Fatal("Expected to find AWF installation step when firewall is enabled")
		}

		if !strings.Contains(awfStepStr, sandboxAgentVersion) {
			t.Errorf("AWF installation step should pass sandbox.agent.version %s to script", sandboxAgentVersion)
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
			if strings.Contains(stepStr, "Install AWF binary") {
				t.Error("Should not include AWF installation step when firewall is disabled")
			}
		}
	})
}
