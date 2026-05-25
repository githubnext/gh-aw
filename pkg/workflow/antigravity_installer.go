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

	// Always pass the version via an env var rather than direct shell interpolation.
	// This prevents injection from user-supplied engine.version values (e.g. values
	// with spaces or shell metacharacters) and also handles GitHub Actions expressions
	// like ${{ inputs.engine-version }} safely.
	installStep := GitHubActionStep([]string{
		"      - name: " + stepName,
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/install_antigravity_cli.sh" "${ENGINE_VERSION}"`,
		"        env:",
		"          ENGINE_VERSION: " + version,
	})

	return []GitHubActionStep{installStep}
}
