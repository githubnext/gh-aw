// Package workflow provides Copilot engine installation logic.
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
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
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
		DocsURL:         "https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default",
		NpmPackage:      "@github/copilot",
		Version:         string(constants.DefaultCopilotVersion),
		Name:            "GitHub Copilot CLI",
		CliName:         "copilot",
		InstallStepName: "Install GitHub Copilot CLI",
	}

	// Add secret validation step
	secretValidation := GenerateMultiSecretValidationStep(
		config.Secrets,
		config.Name,
		config.DocsURL,
	)
	steps = append(steps, secretValidation)

	// Determine Copilot version
	copilotVersion := config.Version
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		copilotVersion = workflowData.EngineConfig.Version
	}

	// Determine if Copilot should be installed globally or locally
	// For SRT, install locally so npx can find it without network access
	installGlobally := !isSRTEnabled(workflowData)

	// Generate install steps based on installation scope
	var npmSteps []GitHubActionStep
	if installGlobally {
		// Use the new installer script for global installation
		copilotInstallLog.Print("Using new installer script for Copilot installation")
		npmSteps = GenerateCopilotInstallerSteps(copilotVersion, config.InstallStepName)
	} else {
		// For SRT: install locally with npm without -g flag
		copilotInstallLog.Print("Using local Copilot installation for SRT compatibility")
		npmSteps = GenerateNpmInstallStepsWithScope(
			config.NpmPackage,
			copilotVersion,
			config.InstallStepName,
			config.CliName,
			true,  // Include Node.js setup
			false, // Install locally, not globally
		)
	}

	// Add Node.js setup step first (before sandbox installation)
	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Setup Node.js step
	}

	// Add sandbox installation steps
	// SRT and AWF are mutually exclusive (validated earlier)
	if isSRTEnabled(workflowData) {
		// Install Sandbox Runtime (SRT)
		agentConfig := getAgentConfig(workflowData)

		// Skip standard installation if custom command is specified
		if agentConfig == nil || agentConfig.Command == "" {
			copilotInstallLog.Print("Adding Sandbox Runtime (SRT) system dependencies step")
			srtSystemDeps := generateSRTSystemDepsStep()
			steps = append(steps, srtSystemDeps)

			copilotInstallLog.Print("Adding Sandbox Runtime (SRT) system configuration step")
			srtSystemConfig := generateSRTSystemConfigStep()
			steps = append(steps, srtSystemConfig)

			copilotInstallLog.Print("Adding Sandbox Runtime (SRT) installation step")
			srtInstall := generateSRTInstallationStep()
			steps = append(steps, srtInstall)
		} else {
			copilotInstallLog.Print("Skipping SRT installation (custom command specified)")
		}
	} else if isFirewallEnabled(workflowData) {
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

	return steps
}

// generateAWFInstallationStep creates a GitHub Actions step to install the AWF binary
// with SHA256 checksum verification to protect against supply chain attacks.
//
// Instead of piping an unverified installer script to bash, this function downloads
// the binary directly from GitHub releases, verifies its checksum against the official
// checksums.txt file, and installs it. This approach:
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
		"        run: |",
		fmt.Sprintf("          echo \"Installing awf binary with checksum verification (version: %s)\"", version),
		"          ",
		"          # Configuration",
		fmt.Sprintf("          AWF_VERSION=\"%s\"", version),
		"          AWF_REPO=\"githubnext/gh-aw-firewall\"",
		"          AWF_BINARY=\"awf-linux-x64\"",
		"          AWF_INSTALL_DIR=\"/usr/local/bin\"",
		"          AWF_INSTALL_NAME=\"awf\"",
		"          ",
		"          # Download URLs",
		"          BASE_URL=\"https://github.com/${AWF_REPO}/releases/download/${AWF_VERSION}\"",
		"          BINARY_URL=\"${BASE_URL}/${AWF_BINARY}\"",
		"          CHECKSUMS_URL=\"${BASE_URL}/checksums.txt\"",
		"          ",
		"          # Create temp directory",
		"          TEMP_DIR=$(mktemp -d)",
		"          trap 'rm -rf \"$TEMP_DIR\"' EXIT",
		"          ",
		"          # Download binary and checksums",
		"          echo \"Downloading binary from ${BINARY_URL}...\"",
		"          curl -fsSL -o \"${TEMP_DIR}/${AWF_BINARY}\" \"${BINARY_URL}\"",
		"          ",
		"          echo \"Downloading checksums from ${CHECKSUMS_URL}...\"",
		"          curl -fsSL -o \"${TEMP_DIR}/checksums.txt\" \"${CHECKSUMS_URL}\"",
		"          ",
		"          # Verify checksum",
		"          echo \"Verifying SHA256 checksum...\"",
		"          cd \"${TEMP_DIR}\"",
		"          EXPECTED_CHECKSUM=$(awk -v fname=\"${AWF_BINARY}\" '$2 == fname {print $1; exit}' checksums.txt | tr 'A-F' 'a-f')",
		"          ",
		"          if [ -z \"$EXPECTED_CHECKSUM\" ]; then",
		"            echo \"ERROR: Could not find checksum for ${AWF_BINARY} in checksums.txt\"",
		"            exit 1",
		"          fi",
		"          ",
		"          ACTUAL_CHECKSUM=$(sha256sum \"${AWF_BINARY}\" | awk '{print $1}' | tr 'A-F' 'a-f')",
		"          ",
		"          if [ \"$EXPECTED_CHECKSUM\" != \"$ACTUAL_CHECKSUM\" ]; then",
		"            echo \"ERROR: Checksum verification failed!\"",
		"            echo \"  Expected: $EXPECTED_CHECKSUM\"",
		"            echo \"  Got:      $ACTUAL_CHECKSUM\"",
		"            echo \"  The downloaded file may be corrupted or tampered with\"",
		"            exit 1",
		"          fi",
		"          ",
		"          echo \"✓ Checksum verification passed\"",
		"          ",
		"          # Make binary executable and install",
		"          chmod +x \"${AWF_BINARY}\"",
		"          sudo mv \"${AWF_BINARY}\" \"${AWF_INSTALL_DIR}/${AWF_INSTALL_NAME}\"",
		"          ",
		"          # Verify installation",
		"          which awf",
		"          awf --version",
		"          ",
		"          echo \"✓ AWF installation complete\"",
	}

	return GitHubActionStep(stepLines)
}
