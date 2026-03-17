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
	// GH_HOST is explicitly set to github.com so that a workflow-level GH_HOST override
	// (e.g. a GHES hostname) does not leak into this step. The Copilot CLI binary is always
	// downloaded from github.com and requires github.com authentication. This step-level
	// env override only affects the install_copilot_cli.sh execution and has no impact on
	// other workflow steps.
	stepLines := []string{
		"      - name: " + stepName,
		"        run: ${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh " + version,
		"        env:",
		"          GH_HOST: github.com",
	}

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}
