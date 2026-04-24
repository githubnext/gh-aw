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

// BuildStandardNpmEngineInstallSteps creates standard npm installation steps for engines.
// This helper extracts the common pattern shared by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - packageName: The npm package name (e.g., "@github/copilot")
//   - defaultVersion: The default version constant (e.g., constants.DefaultCopilotVersion)
//   - stepName: The display name for the install step (e.g., "Install GitHub Copilot CLI")
//   - cacheKeyPrefix: The cache key prefix (e.g., "copilot")
//   - workflowData: The workflow data containing engine configuration
//
// Returns:
//   - []GitHubActionStep: The installation steps including Node.js setup
func BuildStandardNpmEngineInstallSteps(
	packageName string,
	defaultVersion string,
	stepName string,
	cacheKeyPrefix string,
	workflowData *WorkflowData,
) []GitHubActionStep {
	nodejsLog.Printf("Building npm engine install steps: package=%s, version=%s", packageName, defaultVersion)

	// Use version from engine config if provided, otherwise default to pinned version
	version := defaultVersion
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
		nodejsLog.Printf("Using engine config version: %s", version)
	}

	// Add npm package installation steps (includes Node.js setup)
	// Always pass false for runInstallScripts: engine CLI installs must never run
	// pre/post install scripts regardless of the workflow's run-install-scripts setting.
	// This is a supply chain security requirement for the engine binary itself.
	return GenerateNpmInstallSteps(
		packageName,
		version,
		stepName,
		cacheKeyPrefix,
		true,  // Include Node.js setup
		false, // Always disable scripts for engine CLI installs
	)
}

// BuildNpmEngineInstallStepsWithAWF injects an AWF installation step between the Node.js
// setup step and the CLI install steps when the firewall is enabled. This eliminates the
// duplicated AWF-injection pattern shared by Claude, Gemini, and Copilot engines.
//
// The expected layout of npmSteps is:
//   - npmSteps[0]  – Node.js setup step
//   - npmSteps[1:] – CLI installation step(s)
//
// Parameters:
//   - npmSteps: Pre-computed npm installation steps (from BuildStandardNpmEngineInstallSteps
//     or GenerateCopilotInstallerSteps)
//   - workflowData: The workflow data (used to determine firewall configuration)
//
// Returns:
//   - []GitHubActionStep: Steps in order: Node.js setup, AWF (if enabled), CLI install
func BuildNpmEngineInstallStepsWithAWF(npmSteps []GitHubActionStep, workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Node.js setup step
	}

	// Inject AWF installation after Node.js setup but before the CLI install steps
	if isFirewallEnabled(workflowData) {
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
	}

	if len(npmSteps) > 1 {
		steps = append(steps, npmSteps[1:]...) // CLI installation and subsequent steps
	}

	return steps
}

// GetNpmBinPathSetup returns a simple shell command that adds hostedtoolcache bin directories
// to PATH. This is specifically for npm-installed CLIs (like Claude and Codex) that need
// to find their binaries installed via `npm install -g`.
//
// Unlike GetHostedToolcachePathSetup(), this does NOT use GH_AW_TOOL_BINS because AWF's
// native chroot mode already handles tool-specific paths (GOROOT, JAVA_HOME, etc.) via
// AWF_HOST_PATH and the entrypoint.sh script. This function only adds the generic
// hostedtoolcache bin directories for npm packages.
//
// Returns:
//   - string: A shell command that exports PATH with hostedtoolcache bin directories prepended
func GetNpmBinPathSetup() string {
	// Find all bin directories in hostedtoolcache (Node.js, Python, etc.)
	// This finds paths like /opt/hostedtoolcache/node/22.13.0/x64/bin
	//
	// After the find, re-prepend GOROOT/bin if set. The find returns directories
	// alphabetically, so go/1.23.12 shadows go/1.25.0. Re-prepending GOROOT/bin
	// ensures the Go version set by actions/setup-go takes precedence.
	// AWF's entrypoint.sh exports GOROOT before the user command runs.
	return `export PATH="$(find /opt/hostedtoolcache -maxdepth 4 -type d -name bin 2>/dev/null | tr '\n' ':')$PATH"; [ -n "$GOROOT" ] && export PATH="$GOROOT/bin:$PATH" || true`
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
