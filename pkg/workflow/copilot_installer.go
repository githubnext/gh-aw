package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallerLog = logger.New("workflow:copilot_installer")

// GenerateCopilotInstallerSteps creates GitHub Actions steps to install the Copilot CLI using the official installer.
// When rootless is true, the script installs into $HOME/.local/bin without sudo.
func GenerateCopilotInstallerSteps(version, stepName string, rootless bool) []GitHubActionStep {
	copilotInstallerLog.Printf("Generating Copilot installer steps using install_copilot_cli.sh: version=%s, rootless=%v", version, rootless)

	rootlessFlag := ""
	if rootless {
		rootlessFlag = " --rootless"
	}
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
			`        run: bash "${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh" "${ENGINE_VERSION}"` + rootlessFlag,
			"        env:",
			"          GH_HOST: github.com",
			"          GH_AW_DEFAULT_COPILOT_VERSION: " + string(constants.DefaultCopilotVersion),
			"          ENGINE_VERSION: " + version,
		}
		return []GitHubActionStep{GitHubActionStep(stepLines)}
	}

	versionArgument := ""
	if version != "" {
		versionArgument = " " + version
	}
	stepLines := []string{
		"      - name: " + stepName,
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\"" + versionArgument + rootlessFlag,
		"        env:",
		"          GH_HOST: github.com",
		"          GH_AW_DEFAULT_COPILOT_VERSION: " + string(constants.DefaultCopilotVersion),
	}

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}
