package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var nodejsLog = logger.New("workflow:nodejs")

// GenerateNodeJsSetupStep creates a GitHub Actions step for setting up Node.js
// Returns a step that installs Node.js using the default version from constants.DefaultNodeVersion
// Caching is disabled by default to prevent cache poisoning vulnerabilities in release workflows
func GenerateNodeJsSetupStep() GitHubActionStep {
	return GitHubActionStep{
		"      - name: Setup Node.js",
		"        uses: " + getActionPin("actions/setup-node"),
		"        with:",
		fmt.Sprintf("          node-version: '%s'", constants.DefaultNodeVersion),
		"          package-manager-cache: false",
	}
}

// installStepsContainNodeSetup reports whether any of the provided steps is already
// a "Setup Node.js" step. Uses the same extractStepName matcher as
// JobManager.ValidateDuplicateSteps so the guard cannot drift from what the
// validator would flag as a duplicate.
func installStepsContainNodeSetup(steps []GitHubActionStep) bool {
	for _, step := range steps {
		if extractStepName(strings.Join(step, "\n")) == "Setup Node.js" {
			return true
		}
	}
	return false
}

// By default, --ignore-scripts is added to the install command to prevent pre/post install
// scripts from executing (supply chain security). Pass runInstallScripts=true to allow scripts.
// Parameters:
//   - packageName: The npm package name (e.g., "@anthropic-ai/claude-code")
//   - version: The package version to install
//   - stepName: The name to display for the install step (e.g., "Install Claude Code CLI")
//   - cacheKeyPrefix: The prefix for the cache key (unused, kept for API compatibility)
//   - includeNodeSetup: If true, includes Node.js setup step before npm install
//   - runInstallScripts: If true, allow pre/post install scripts (omits --ignore-scripts)
//
// Returns steps for installing the npm package (optionally with Node.js setup)
func GenerateNpmInstallSteps(packageName, version, stepName, cacheKeyPrefix string, includeNodeSetup bool, runInstallScripts bool) []GitHubActionStep {
	return GenerateNpmInstallStepsWithScope(packageName, version, stepName, cacheKeyPrefix, includeNodeSetup, true, runInstallScripts)
}

// GenerateNpmInstallStepsWithScope generates npm installation steps with control over global vs local installation.
// By default, --ignore-scripts is added to the install command to prevent pre/post install
// scripts from executing (supply chain security). Pass runInstallScripts=true to allow scripts.
func GenerateNpmInstallStepsWithScope(packageName, version, stepName, cacheKeyPrefix string, includeNodeSetup bool, isGlobal bool, runInstallScripts bool) []GitHubActionStep {
	nodejsLog.Printf("Generating npm install steps: package=%s, version=%s, includeNodeSetup=%v, isGlobal=%v, runInstallScripts=%v", packageName, version, includeNodeSetup, isGlobal, runInstallScripts)

	var steps []GitHubActionStep

	// Add Node.js setup if requested
	if includeNodeSetup {
		nodejsLog.Print("Including Node.js setup step")
		steps = append(steps, GenerateNodeJsSetupStep())
	}

	// Add npm install step
	globalFlag := ""
	if isGlobal {
		globalFlag = "-g "
	}

	// Add --ignore-scripts by default to prevent pre/post install scripts (supply chain security).
	// runInstallScripts=true disables this protection (emits a warning at compile time).
	ignoreScriptsFlag := "--ignore-scripts "
	if runInstallScripts {
		ignoreScriptsFlag = ""
	}

	var installStep GitHubActionStep
	if ExpressionPattern.MatchString(version) {
		// Version is a GitHub Actions expression (e.g. ${{ inputs.engine-version }}).
		// Pass it via an env var instead of direct shell interpolation to prevent injection:
		// if the expression evaluates to a malicious string, it would otherwise be
		// substituted verbatim into the shell command before the shell parses it.
		nodejsLog.Printf("Version contains GitHub Actions expression, using env var for injection safety: %s", version)
		installCmd := fmt.Sprintf(`npm install %s%s%s@"${ENGINE_VERSION}"`, ignoreScriptsFlag, globalFlag, packageName)
		installStep = GitHubActionStep{
			"      - name: " + stepName,
			"        run: " + installCmd,
			"        env:",
			"          ENGINE_VERSION: " + version,
		}
	} else {
		installCmd := fmt.Sprintf("npm install %s%s%s@%s", ignoreScriptsFlag, globalFlag, packageName, version)
		installStep = GitHubActionStep{
			"      - name: " + stepName,
			"        run: " + installCmd,
		}
	}
	steps = append(steps, installStep)

	return steps
}
