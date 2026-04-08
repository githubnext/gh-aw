// This file provides Copilot engine installation logic.
//
// This file contains functions for generating GitHub Actions steps to install
// the GitHub Copilot CLI and related sandbox infrastructure (AWF or SRT).
//
// Installation order:
//  1. Secret validation (COPILOT_GITHUB_TOKEN) — runs in the activation job
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

// GetSecretValidationStep returns the secret validation step for the Copilot engine.
// Returns an empty step if copilot-requests feature is enabled or custom command is specified.
func (e *CopilotEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	if isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData) {
		copilotInstallLog.Print("Skipping secret validation step: copilot-requests feature enabled, using GitHub Actions token")
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"COPILOT_GITHUB_TOKEN"},
		"GitHub Copilot CLI",
		"https://github.github.com/gh-aw/reference/engines/#github-copilot-default",
	)
}

// GetInstallationSteps generates the complete installation workflow for Copilot CLI.
// This includes Node.js setup, sandbox installation (SRT or AWF), and Copilot CLI installation.
// Secret validation is handled separately in the activation job via GetSecretValidationStep.
// The installation order is:
// 1. Node.js setup
// 2. Sandbox installation (AWF, if needed)
// 3. Copilot CLI installation
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

	// Determine Copilot version
	copilotVersion := string(constants.DefaultCopilotVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		copilotVersion = workflowData.EngineConfig.Version
	}

	// Use the installer script for global installation
	copilotInstallLog.Print("Using new installer script for Copilot installation")
	npmSteps := GenerateCopilotInstallerSteps(copilotVersion, "Install GitHub Copilot CLI")
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
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
		"      - name: Install AWF binary",
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/install_awf_binary.sh\" " + version,
	}

	return GitHubActionStep(stepLines)
}
