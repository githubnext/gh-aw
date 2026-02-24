// This file provides Copilot engine installation logic.
//
// This file contains functions for generating GitHub Actions steps to install
// the GitHub Copilot CLI and related sandbox infrastructure (AWF or SRT).
//
// Installation order:
//  1. Secret validation (COPILOT_GITHUB_TOKEN)
//  2. Node.js setup
//  3. Sandbox installation (SRT or AWF, if needed)
//  4. Copilot CLI installation
//
// The installation strategy differs based on sandbox mode:
//   - Standard mode: Global installation using official installer script
//   - SRT mode: Local npm installation for offline compatibility
//   - AWF mode: Global installation + AWF binary

package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallLog = logger.New("workflow:copilot_engine_installation")

// GetInstallationSteps generates the complete installation workflow for Copilot CLI.
// This includes secret validation, Node.js setup, sandbox installation (SRT or AWF),
// and Copilot CLI installation. The installation order is critical:
// 1. Secret validation
// 2. Node.js setup
// 3. Sandbox installation (SRT or AWF, if needed)
// 4. Copilot CLI installation
//
// If a custom command is specified in the engine configuration, this function returns
// an empty list of steps, skipping the standard installation process.
func (e *CopilotEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	copilotInstallLog.Printf("Generating installation steps for Copilot engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		copilotInstallLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	var steps []GitHubActionStep

	// Define engine configuration for shared validation
	config := EngineInstallConfig{
		Secrets:         []string{"COPILOT_GITHUB_TOKEN"},
		DocsURL:         "https://github.github.com/gh-aw/reference/engines/#github-copilot-default",
		NpmPackage:      "@github/copilot",
		Version:         string(constants.DefaultCopilotVersion),
		Name:            "GitHub Copilot CLI",
		CliName:         "copilot",
		InstallStepName: "Install GitHub Copilot CLI",
	}

	// Add secret validation step unless copilot-requests feature is enabled.
	// When copilot-requests is enabled, the GitHub Actions token is used directly
	// (no COPILOT_GITHUB_TOKEN secret required).
	if !isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData) {
		secretValidation := GenerateMultiSecretValidationStep(
			config.Secrets,
			config.Name,
			config.DocsURL,
			getEngineEnvOverrides(workflowData),
		)
		steps = append(steps, secretValidation)
	} else {
		copilotInstallLog.Print("Skipping secret validation step: copilot-requests feature enabled, using GitHub Actions token")
	}

	// Determine Copilot version
	copilotVersion := config.Version
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		copilotVersion = workflowData.EngineConfig.Version
	}

	// Determine if Copilot should be installed globally
	// Always install globally now (SRT removed)
	installGlobally := true

	// Generate install steps based on installation scope
	var npmSteps []GitHubActionStep
	if installGlobally {
		// Use the new installer script for global installation
		copilotInstallLog.Print("Using new installer script for Copilot installation")
		npmSteps = GenerateCopilotInstallerSteps(copilotVersion, config.InstallStepName)
	}

	// Add Node.js setup step first (before sandbox installation)
	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Setup Node.js step
	}

	// Add sandbox installation steps (AWF only)
	if isFirewallEnabled(workflowData) {
		// Install AWF after Node.js setup but before Copilot CLI installation
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}

		// Install AWF binary (or skip if custom command is specified)
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
	}

	// Add Copilot CLI installation step after sandbox installation
	if len(npmSteps) > 1 {
		steps = append(steps, npmSteps[1:]...) // Install Copilot CLI and subsequent steps
	}

	// Add plugin installation steps after Copilot CLI installation
	if workflowData.PluginInfo != nil && len(workflowData.PluginInfo.Plugins) > 0 {
		copilotInstallLog.Printf("Adding plugin installation steps: %d plugins", len(workflowData.PluginInfo.Plugins))
		// Use plugin-specific token if provided
		tokenToUse := workflowData.PluginInfo.CustomToken
		pluginSteps := GeneratePluginInstallationSteps(workflowData.PluginInfo.Plugins, "copilot", tokenToUse)
		steps = append(steps, pluginSteps...)
	}

	return steps
}

// generateAWFInstallationStep creates a GitHub Actions step to install the AWF binary
// with SHA256 checksum verification to protect against supply chain attacks.
//
// The installation logic is implemented in a separate shell script (install_awf_binary.sh)
// which downloads the binary directly from GitHub releases, verifies its checksum against
// the official checksums.txt file, and installs it. This approach:
// - Eliminates trust in the installer script itself
// - Provides full transparency of the installation process
// - Protects against tampered or compromised installer scripts
// - Verifies the binary integrity before execution
//
// If a custom command is specified in the agent config, the installation is skipped
// as the custom command replaces the AWF binary.
func generateAWFInstallationStep(version string, agentConfig *AgentSandboxConfig) GitHubActionStep {
	// If custom command is specified, skip installation (command replaces binary)
	if agentConfig != nil && agentConfig.Command != "" {
		copilotInstallLog.Print("Skipping AWF binary installation (custom command specified)")
		// Return empty step - custom command will be used in execution
		return GitHubActionStep([]string{})
	}

	// Use default version for logging when not specified
	if version == "" {
		version = string(constants.DefaultFirewallVersion)
	}

	stepLines := []string{
		"      - name: Install awf binary",
		"        run: bash /opt/gh-aw/actions/install_awf_binary.sh " + version,
	}

	return GitHubActionStep(stepLines)
}
