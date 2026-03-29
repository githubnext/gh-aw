package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallerLog = logger.New("workflow:copilot_installer")

// GenerateCopilotInstallerSteps creates GitHub Actions steps to install the Copilot CLI using the official installer.
func GenerateCopilotInstallerSteps(version, stepName string) []GitHubActionStep {
	// If no version is specified, use the default version from constants.
	// "latest" means the installer will use the latest available release.
	if version == "" {
		version = string(constants.DefaultCopilotVersion)
		copilotInstallerLog.Printf("No version specified, using default: %s", version)
	}

	copilotInstallerLog.Printf("Generating Copilot installer steps using install_copilot_cli.sh: version=%s", version)

	// Use the install_copilot_cli.sh script from actions/setup/sh
	// This script includes retry logic for robustness against transient network failures.
	// The script downloads the Copilot CLI using curl with hardcoded github.com URLs
	// and does not use gh CLI, so GH_HOST does not affect the download. No step-level
	// GH_HOST override is needed here; the correct host is already set in GITHUB_ENV
	// by configure_gh_for_ghe.sh (or by the Derive GH_HOST step when DIFC proxy is active).
	stepLines := []string{
		"      - name: " + stepName,
		"        run: ${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh " + version,
	}

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}
