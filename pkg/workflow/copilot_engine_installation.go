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
	return GetStandardSecretValidationStep(
		copilotInstallLog,
		workflowData,
		[]string{"COPILOT_GITHUB_TOKEN"},
		"GitHub Copilot CLI",
		"https://github.github.com/gh-aw/reference/engines/#github-copilot-default",
		func(wd *WorkflowData) bool {
			return isFeatureEnabled(constants.CopilotRequestsFeatureFlag, wd)
		},
		"Skipping secret validation step: copilot-requests feature enabled, using GitHub Actions token",
	)
}

// GetInstallationSteps generates the complete installation workflow for Copilot CLI.
// This includes Node.js setup, sandbox installation (SRT or AWF), and Copilot CLI installation.
// Secret validation is handled separately in the activation job via GetSecretValidationStep.
// The installation order is:
// 1. Node.js setup
// 2. Sandbox installation (SRT or AWF, if needed)
// 3. Copilot CLI installation
//
// If a custom command is specified in the engine configuration, this function returns
// an empty list of steps, skipping the standard installation process.
func (e *CopilotEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	copilotInstallLog.Printf("Generating installation steps for Copilot engine: workflow=%s", workflowData.Name)

	// Build plugin installation steps (if any) to pass as post-install steps.
	var pluginSteps []GitHubActionStep
	if workflowData.PluginInfo != nil && len(workflowData.PluginInfo.Plugins) > 0 {
		copilotInstallLog.Printf("Adding plugin installation steps: %d plugins", len(workflowData.PluginInfo.Plugins))
		tokenToUse := workflowData.PluginInfo.CustomToken
		pluginSteps = GeneratePluginInstallationSteps(workflowData.PluginInfo.Plugins, "copilot", tokenToUse)
	}

	config := EngineInstallConfig{
		Version:         string(constants.DefaultCopilotVersion),
		InstallStepName: "Install GitHub Copilot CLI",
	}

	return BuildEngineInstallationSteps(
		copilotInstallLog,
		workflowData,
		func() []GitHubActionStep {
			version := config.Version
			if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
				version = workflowData.EngineConfig.Version
			}
			copilotInstallLog.Print("Using new installer script for Copilot installation")
			return GenerateCopilotInstallerSteps(version, config.InstallStepName)
		},
		pluginSteps,
	)
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
