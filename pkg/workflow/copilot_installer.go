package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallerLog = logger.New("workflow:copilot_installer")

// GenerateCopilotInstallerSteps creates GitHub Actions steps to install the Copilot CLI using the official installer.
func GenerateCopilotInstallerSteps(version, stepName string) []GitHubActionStep {
	// If no version is specified, use the default version from constants
	// This prevents the installer from defaulting to "latest"
	if version == "" {
		version = string(constants.DefaultCopilotVersion)
		copilotInstallerLog.Printf("No version specified, using default: %s", version)
	}

	copilotInstallerLog.Printf("Generating Copilot installer steps using install_copilot_cli.sh: version=%s", version)

	// Use the install_copilot_cli.sh script from actions/setup/sh
	// This script includes retry logic for robustness against transient network failures
	stepLines := []string{
		"      - name: " + stepName,
		"        run: /opt/gh-aw/actions/install_copilot_cli.sh " + version,
	}

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}
