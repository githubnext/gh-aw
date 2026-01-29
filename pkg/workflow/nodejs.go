package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var nodejsLog = logger.New("workflow:nodejs")

// GenerateNodeJsSetupStep creates a GitHub Actions step for setting up Node.js
// Returns a step that installs Node.js using the default version from constants.DefaultNodeVersion
// Caching is disabled by default to prevent cache poisoning vulnerabilities in release workflows
func GenerateNodeJsSetupStep() GitHubActionStep {
	return GitHubActionStep{
		"      - name: Setup Node.js",
		fmt.Sprintf("        uses: %s", GetActionPin("actions/setup-node")),
		"        with:",
		fmt.Sprintf("          node-version: '%s'", constants.DefaultNodeVersion),
		"          package-manager-cache: false",
	}
}

// GenerateNpmInstallSteps creates GitHub Actions steps for installing an npm package globally
// Parameters:
//   - packageName: The npm package name (e.g., "@anthropic-ai/claude-code")
//   - version: The package version to install
//   - stepName: The name to display for the install step (e.g., "Install Claude Code CLI")
//   - cacheKeyPrefix: The prefix for the cache key (unused, kept for API compatibility)
//   - includeNodeSetup: If true, includes Node.js setup step before npm install
//
// Returns steps for installing the npm package (optionally with Node.js setup)
func GenerateNpmInstallSteps(packageName, version, stepName, cacheKeyPrefix string, includeNodeSetup bool) []GitHubActionStep {
	return GenerateNpmInstallStepsWithScope(packageName, version, stepName, cacheKeyPrefix, includeNodeSetup, true)
}

// GenerateNpmInstallStepsWithScope generates npm installation steps with control over global vs local installation
// Parameters:
//   - packageName: The npm package name
//   - version: The package version (can be a runtime expression like "${{ env.VAR }}")
//   - stepName: The display name for the install step
//   - cacheKeyPrefix: Unused, kept for API compatibility
//   - includeNodeSetup: If true, includes Node.js setup step
//   - isGlobal: If true, installs globally with -g flag
//
// Returns steps for installing the npm package with optional Node.js setup
func GenerateNpmInstallStepsWithScope(packageName, version, stepName, cacheKeyPrefix string, includeNodeSetup bool, isGlobal bool) []GitHubActionStep {
	nodejsLog.Printf("Generating npm install steps: package=%s, version=%s, includeNodeSetup=%v, isGlobal=%v", packageName, version, includeNodeSetup, isGlobal)

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
	installCmd := fmt.Sprintf("npm install %s--silent %s@%s", globalFlag, packageName, version)
	steps = append(steps, GitHubActionStep{
		fmt.Sprintf("      - name: %s", stepName),
		fmt.Sprintf("        run: %s", installCmd),
	})

	return steps
}

// GenerateNpmInstallStepsWithEnvOverride generates npm installation steps with environment variable override support
// This function allows runtime version override through environment variables like GH_AW_CLAUDE_VERSION
//
// Parameters:
//   - packageName: The npm package name (e.g., "@anthropic-ai/claude-code")
//   - defaultVersion: The default version to use if env var is not set
//   - envVarName: The environment variable name for version override (e.g., "GH_AW_CLAUDE_VERSION")
//   - stepName: The display name for the install step
//   - cacheKeyPrefix: Unused, kept for API compatibility
//   - includeNodeSetup: If true, includes Node.js setup step
//   - isGlobal: If true, installs globally with -g flag
//
// Returns steps for installing the npm package with optional Node.js setup
func GenerateNpmInstallStepsWithEnvOverride(packageName, defaultVersion, envVarName, stepName, cacheKeyPrefix string, includeNodeSetup bool, isGlobal bool) []GitHubActionStep {
	nodejsLog.Printf("Generating npm install steps with env override: package=%s, defaultVersion=%s, envVar=%s", packageName, defaultVersion, envVarName)

	var steps []GitHubActionStep

	// Add Node.js setup if requested
	if includeNodeSetup {
		nodejsLog.Print("Including Node.js setup step")
		steps = append(steps, GenerateNodeJsSetupStep())
	}

	// Add npm install step with environment variable override
	globalFlag := ""
	if isGlobal {
		globalFlag = "-g "
	}

	// Use GitHub Actions environment variable syntax: ${{ env.VAR || 'default' }}
	stepLines := []string{
		fmt.Sprintf("      - name: %s", stepName),
		"        env:",
		fmt.Sprintf("          CLI_VERSION: ${{ env.%s || '%s' }}", envVarName, defaultVersion),
		fmt.Sprintf("        run: npm install %s--silent %s@\"${CLI_VERSION}\"", globalFlag, packageName),
	}

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}
