// Package workflow provides Copilot engine installation logic.
//
// This file contains functions for generating GitHub Actions steps to install
// the GitHub Copilot CLI and related sandbox infrastructure (AWF or SRT).
//
// Installation order:
//  1. Secret/App variable validation
//  2. Node.js setup
//  3. App token minting (if engine.app is configured)
//  4. Sandbox installation (SRT or AWF, if needed)
//  5. Copilot CLI installation
//
// The installation strategy differs based on sandbox mode:
//   - Standard mode: Global installation using official installer script
//   - SRT mode: Local npm installation for offline compatibility
//   - AWF mode: Global installation + AWF binary
package workflow

import (
	"fmt"
	"strings"

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

	// Check if engine.app is configured
	hasEngineApp := workflowData.EngineConfig != nil && workflowData.EngineConfig.App != nil

	// Define engine configuration for shared validation
	config := EngineInstallConfig{
		DocsURL:         "https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default",
		NpmPackage:      "@github/copilot",
		Version:         string(constants.DefaultCopilotVersion),
		Name:            "GitHub Copilot CLI",
		CliName:         "copilot",
		InstallStepName: "Install GitHub Copilot CLI",
	}

	// Add secret validation step only if app is not configured
	// When app is configured, we'll validate app variables instead
	if !hasEngineApp {
		config.Secrets = []string{"COPILOT_GITHUB_TOKEN"}
		secretValidation := GenerateMultiSecretValidationStep(
			config.Secrets,
			config.Name,
			config.DocsURL,
		)
		steps = append(steps, secretValidation)
	} else {
		// Generate app variable validation step
		appValidationStep := generateAppVariableValidationStep(workflowData.EngineConfig.App, config.Name, config.DocsURL)
		steps = append(steps, appValidationStep)
	}

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

	// Add Node.js setup step first (before sandbox installation and token minting)
	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Setup Node.js step
	}

	// Add GitHub App token minting step if engine.app is configured
	// This must come after Node.js setup but before sandbox installation
	if hasEngineApp && workflowData.EngineConfig.App != nil {
		copilotInstallLog.Print("Adding GitHub App token minting step for Copilot engine")
		tokenMintSteps := e.buildCopilotEngineAppTokenMintStep(workflowData.EngineConfig.App)
		for _, tokenStep := range tokenMintSteps {
			steps = append(steps, GitHubActionStep([]string{tokenStep}))
		}
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
		fmt.Sprintf("        run: bash /opt/gh-aw/actions/install_awf_binary.sh %s", version),
	}

	return GitHubActionStep(stepLines)
}

// generateAppVariableValidationStep creates a GitHub Actions step to validate GitHub App variables
// This validates that the required app-id and private-key variables/secrets are configured
func generateAppVariableValidationStep(app *GitHubAppConfig, engineName, docsURL string) GitHubActionStep {
	if app == nil {
		copilotInstallLog.Print("WARNING: generateAppVariableValidationStep called with nil app config")
		return GitHubActionStep([]string{})
	}

	// Use shell script for validation logic
	stepLines := []string{
		"      - name: Validate GitHub App variables",
		"        id: validate-secret",
		fmt.Sprintf("        run: bash /opt/gh-aw/actions/validate_app_support_engine_field.sh \"%s\" \"%s\"", engineName, docsURL),
		"        env:",
		fmt.Sprintf("          APP_ID: %s", app.AppID),
		fmt.Sprintf("          APP_PRIVATE_KEY: %s", app.PrivateKey),
	}

	return GitHubActionStep(stepLines)
}

// buildCopilotEngineAppTokenMintStep generates the step to mint a GitHub App installation access token for Copilot
// This token will have "copilot-requests: read" permission
func (e *CopilotEngine) buildCopilotEngineAppTokenMintStep(app *GitHubAppConfig) []string {
	copilotInstallLog.Print("Building Copilot engine GitHub App token mint step")
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token for Copilot\n")
	steps = append(steps, "        id: copilot-engine-app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Add repositories - default to current repository name if not specified
	if len(app.Repositories) > 0 {
		reposStr := strings.Join(app.Repositories, ",")
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", reposStr))
	} else {
		// Extract repo name from github.repository (which is "owner/repo")
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	// Add copilot-requests permission (read access)
	steps = append(steps, "          permission-copilot-requests: read\n")

	return steps
}

// buildCopilotEngineAppTokenInvalidationStep generates the step to invalidate the Copilot engine GitHub App token
// This step always runs (even on failure) to ensure tokens are properly cleaned up
func (e *CopilotEngine) buildCopilotEngineAppTokenInvalidationStep() []string {
	var steps []string

	steps = append(steps, "      - name: Invalidate Copilot engine GitHub App token\n")
	steps = append(steps, "        if: always() && steps.copilot-engine-app-token.outputs.token != ''\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, "          TOKEN: ${{ steps.copilot-engine-app-token.outputs.token }}\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Revoking Copilot engine GitHub App installation token...\"\n")
	steps = append(steps, "          # GitHub CLI will auth with the token being revoked.\n")
	steps = append(steps, "          gh api \\\n")
	steps = append(steps, "            --method DELETE \\\n")
	steps = append(steps, "            -H \"Authorization: token $TOKEN\" \\\n")
	steps = append(steps, "            /installation/token || echo \"Token revoke may already be expired.\"\n")
	steps = append(steps, "          \n")
	steps = append(steps, "          echo \"Token invalidation step complete.\"\n")

	return steps
}
