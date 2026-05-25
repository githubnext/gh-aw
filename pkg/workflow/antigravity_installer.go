package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var antigravityInstallerLog = logger.New("workflow:antigravity_installer")

// GenerateAntigravityInstallerSteps creates GitHub Actions steps to install the Antigravity CLI
// using the official binary from Google Cloud Storage.
func GenerateAntigravityInstallerSteps(version, stepName string) []GitHubActionStep {
	// If no version is specified, use the pinned default version from constants.
	if version == "" {
		version = string(constants.DefaultAntigravityVersion)
		antigravityInstallerLog.Printf("No version specified, using default: %s", version)
	}

	antigravityInstallerLog.Printf("Generating Antigravity installer steps using install_antigravity_cli.sh: version=%s", version)

	// Use the install_antigravity_cli.sh script from actions/setup/sh
	// This script downloads the Antigravity CLI binary directly from Google Cloud Storage.
	var installStep GitHubActionStep
	if ExpressionPattern.MatchString(version) {
		// Version is a GitHub Actions expression (e.g. ${{ inputs.engine-version }}).
		// Pass it via an env var instead of direct shell interpolation to prevent injection.
		antigravityInstallerLog.Printf("Version contains GitHub Actions expression, using env var for injection safety: %s", version)
		installStep = GitHubActionStep([]string{
			"      - name: " + stepName,
			`        run: bash "${RUNNER_TEMP}/gh-aw/actions/install_antigravity_cli.sh" "${ENGINE_VERSION}"`,
			"        env:",
			"          ENGINE_VERSION: " + version,
		})
	} else {
		installStep = GitHubActionStep([]string{
			"      - name: " + stepName,
			"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/install_antigravity_cli.sh\" " + version,
		})
	}

	return []GitHubActionStep{GenerateNodeJsSetupStep(), installStep}
}
