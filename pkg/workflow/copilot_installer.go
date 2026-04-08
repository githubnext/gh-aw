package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallerLog = logger.New("workflow:copilot_installer")

// GenerateCopilotInstallerSteps creates GitHub Actions steps to install the Copilot CLI using the official installer.
func GenerateCopilotInstallerSteps(version, stepName string) []GitHubActionStep {
	// If no version is specified, use the pinned default version from constants.
	if version == "" {
		version = string(constants.DefaultCopilotVersion)
		copilotInstallerLog.Printf("No version specified, using default: %s", version)
	}

	copilotInstallerLog.Printf("Generating Copilot installer steps using install_copilot_cli.sh: version=%s", version)

	// Use the install_copilot_cli.sh script from actions/setup/sh
	// This script includes retry logic for robustness against transient network failures.
	// The script downloads the Copilot CLI using curl with hardcoded github.com URLs.
	//
	// GH_HOST is pinned to github.com at the step level to prevent any workflow-level
	// env.GH_HOST (common on GHES deployments) from leaking into this step and
	// interfering with the Copilot CLI install/auth path, which requires github.com.
	if ExpressionPattern.MatchString(version) {
		// Version is a GitHub Actions expression (e.g. ${{ inputs.engine-version }}).
		// Pass it via an env var instead of direct shell interpolation to prevent injection.
		copilotInstallerLog.Printf("Version contains GitHub Actions expression, using env var for injection safety: %s", version)
		stepLines := []string{
			"      - name: " + stepName,
			`        run: bash "${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh" "${ENGINE_VERSION}"`,
			"        env:",
			"          GH_HOST: github.com",
			"          ENGINE_VERSION: " + version,
		}
		return []GitHubActionStep{GitHubActionStep(stepLines)}
	}

	stepLines := []string{
		"      - name: " + stepName,
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" " + version,
		"        env:",
		"          GH_HOST: github.com",
	}

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}
